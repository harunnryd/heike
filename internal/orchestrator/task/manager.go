package task

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"regexp"
	"strings"
	"time"
	"unicode"

	"github.com/harunnryd/heike/internal/cognitive"
	"github.com/harunnryd/heike/internal/config"
	"github.com/harunnryd/heike/internal/model/contract"
	"github.com/harunnryd/heike/internal/orchestrator/session"
	"github.com/harunnryd/heike/internal/skill"
	"github.com/harunnryd/heike/internal/tool"
)

type Manager interface {
	HandleRequest(ctx context.Context, sessionID string, goal string) error
}

type SkillProvider interface {
	Get(name string) (*skill.Skill, error)
	GetRelevant(query string, limit int) ([]*skill.Skill, error)
}

type DefaultTaskManager struct {
	engine      cognitive.Engine
	decomposer  TaskDecomposer
	coordinator *Coordinator
	session     session.Manager
	tools       []tool.ToolDescriptor
	toolBroker  ToolBroker
	skills      SkillProvider
}

func NewManager(
	e cognitive.Engine,
	d TaskDecomposer,
	s session.Manager,
	tools []tool.ToolDescriptor,
	toolBroker ToolBroker,
	skills SkillProvider,
	subTaskRetryMax int,
	subTaskRetryBackoff time.Duration,
) *DefaultTaskManager {
	clonedTools := append([]tool.ToolDescriptor(nil), tools...)
	return &DefaultTaskManager{
		engine:      e,
		decomposer:  d,
		coordinator: NewCoordinator(e, subTaskRetryMax, subTaskRetryBackoff),
		session:     s,
		tools:       clonedTools,
		toolBroker:  toolBroker,
		skills:      skills,
	}
}

func (tm *DefaultTaskManager) HandleRequest(ctx context.Context, sessionID string, goal string) error {
	// Build Context
	cCtx, err := tm.session.GetContext(ctx, sessionID)
	if err != nil {
		return fmt.Errorf("failed to load context: %w", err)
	}
	tm.applySkillContext(cCtx, goal)

	// Decide: Simple or Complex?
	if tm.decomposer.ShouldDecompose(goal) {
		return tm.executeComplexTask(ctx, cCtx, goal)
	}

	return tm.executeSimpleTask(ctx, cCtx, goal)
}

func (tm *DefaultTaskManager) executeSimpleTask(ctx context.Context, cCtx *cognitive.CognitiveContext, goal string) error {
	slog.Info("Executing simple task", "goal", goal)

	// Reuse Cognitive Engine directly
	result, err := tm.engine.Run(ctx, goal, func(c *cognitive.CognitiveContext) {
		*c = *cCtx // Inject session context
		tm.applyToolDefinitions(c, goal)
	})

	if err != nil {
		return tm.session.AppendInteraction(ctx, cCtx.SessionID, "system", fmt.Sprintf("Error: %v", err))
	}

	return tm.session.AppendInteraction(ctx, cCtx.SessionID, "assistant", result.Content)
}

func (tm *DefaultTaskManager) executeComplexTask(ctx context.Context, cCtx *cognitive.CognitiveContext, goal string) error {
	slog.Info("Executing complex task", "goal", goal)
	tm.applyToolDefinitions(cCtx, goal)

	subTasks, err := tm.decomposer.Decompose(ctx, goal)
	if err != nil {
		return err
	}

	tm.session.AppendInteraction(ctx, cCtx.SessionID, "system", fmt.Sprintf("Task decomposed into %d sub-tasks.", len(subTasks)))

	results, err := tm.coordinator.ExecuteDAG(ctx, cCtx, subTasks)
	if err != nil {
		return fmt.Errorf("DAG execution failed: %w", err)
	}

	// Aggregate results
	var sb strings.Builder
	sb.WriteString("Sub-task results:\n")
	for _, res := range results {
		status := "Success"
		if !res.Success {
			status = fmt.Sprintf("Failed (%v)", res.Error)
		}
		sb.WriteString(fmt.Sprintf("- Task %s: %s\n", res.ID, status))
		if res.Output != "" {
			sb.WriteString(fmt.Sprintf("  Output: %s\n", res.Output))
		}
	}

	return tm.session.AppendInteraction(ctx, cCtx.SessionID, "assistant", sb.String())
}

func (tm *DefaultTaskManager) applyToolDefinitions(cCtx *cognitive.CognitiveContext, goal string) {
	if cCtx == nil {
		return
	}

	selected := append([]tool.ToolDescriptor(nil), tm.tools...)
	selectionDetails := []ToolSelectionDetail(nil)
	if tm.toolBroker != nil {
		if explainable, ok := tm.toolBroker.(ExplainableToolBroker); ok {
			result := explainable.SelectWithMetadata(goal, selected)
			if len(result.Tools) > 0 {
				selected = result.Tools
			}
			selectionDetails = result.Details
		} else {
			brokerSelected := tm.toolBroker.Select(goal, selected)
			if len(brokerSelected) > 0 {
				selected = brokerSelected
			}
		}
	}

	defs := toolDefinitionsFromDescriptors(selected)
	cCtx.AvailableTools = defs

	slog.Debug("Tool selection applied",
		"goal_preview", previewString(goal, 80),
		"selected_tools", len(defs),
		"total_tools", len(tm.tools),
		"selected_names", toolNames(defs),
		"selection_details", formatSelectionDetails(selectionDetails, 5))
}

func previewString(s string, n int) string {
	s = strings.TrimSpace(s)
	if n <= 0 || len(s) <= n {
		return s
	}
	return s[:n]
}

func toolNames(toolDefs []contract.ToolDef) []string {
	names := make([]string, 0, len(toolDefs))
	for _, t := range toolDefs {
		names = append(names, t.Name)
	}
	return names
}

func toolDefinitionsFromDescriptors(descriptors []tool.ToolDescriptor) []contract.ToolDef {
	defs := make([]contract.ToolDef, 0, len(descriptors))
	for _, descriptor := range descriptors {
		defs = append(defs, descriptor.Definition)
	}
	return defs
}

func formatSelectionDetails(details []ToolSelectionDetail, limit int) []string {
	if len(details) == 0 || limit <= 0 {
		return nil
	}

	if limit > len(details) {
		limit = len(details)
	}

	out := make([]string, 0, limit)
	for i := 0; i < limit; i++ {
		d := details[i]
		reason := strings.Join(d.Reasons, "|")
		out = append(out, fmt.Sprintf("%s(score=%d,reasons=%s)", d.Name, d.Score, reason))
	}
	return out
}

var skillMentionPattern = regexp.MustCompile(`\$([A-Za-z0-9_-]+)`)

const (
	defaultSkillSelectionLimit = 4
	maxSkillContentChars       = 600
)

func (tm *DefaultTaskManager) applySkillContext(cCtx *cognitive.CognitiveContext, goal string) {
	if cCtx == nil || tm.skills == nil {
		return
	}

	selected, err := tm.selectSkills(goal, defaultSkillSelectionLimit)
	if err != nil || len(selected) == 0 {
		if err != nil {
			slog.Warn("Skill selection failed", "error", err)
		}
		return
	}

	names := make([]string, 0, len(selected))
	details := make([]string, 0, len(selected))
	for _, item := range selected {
		if item == nil {
			continue
		}

		name := strings.TrimSpace(item.Name)
		if name == "" {
			continue
		}

		names = append(names, name)
		details = append(details, formatSkillContextLine(item))
	}

	if len(names) == 0 {
		return
	}

	if cCtx.Metadata == nil {
		cCtx.Metadata = make(map[string]string)
	}
	cCtx.AvailableSkills = names
	cCtx.Metadata["skills_context"] = strings.Join(details, "\n")

	slog.Debug("Skill context applied",
		"goal_preview", previewString(goal, 80),
		"skills", names,
		"count", len(names))
}

func (tm *DefaultTaskManager) selectSkills(goal string, limit int) ([]*skill.Skill, error) {
	if tm.skills == nil || limit <= 0 {
		return nil, nil
	}

	selected := make([]*skill.Skill, 0, limit)
	seen := make(map[string]struct{}, limit)
	add := func(candidate *skill.Skill) {
		if candidate == nil {
			return
		}
		name := strings.ToLower(strings.TrimSpace(candidate.Name))
		if name == "" {
			return
		}
		if _, exists := seen[name]; exists {
			return
		}
		seen[name] = struct{}{}
		selected = append(selected, candidate)
	}

	for _, mention := range extractSkillMentions(goal) {
		if len(selected) >= limit {
			break
		}
		explicitSkill, err := tm.resolveSkillMention(mention)
		if err == nil {
			add(explicitSkill)
		}
	}

	if len(selected) >= limit {
		return selected[:limit], nil
	}

	relevant, err := tm.skills.GetRelevant(goal, limit*2)
	if err != nil {
		return selected, err
	}
	for _, candidate := range relevant {
		add(candidate)
		if len(selected) >= limit {
			break
		}
	}

	return selected, nil
}

func (tm *DefaultTaskManager) resolveSkillMention(name string) (*skill.Skill, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("empty skill mention")
	}

	sk, err := tm.skills.Get(name)
	if err == nil {
		return sk, nil
	}

	alternatives, altErr := tm.skills.GetRelevant(name, 1)
	if altErr != nil {
		return nil, altErr
	}
	if len(alternatives) == 0 {
		return nil, err
	}
	return alternatives[0], nil
}

func extractSkillMentions(goal string) []string {
	matches := skillMentionPattern.FindAllStringSubmatch(goal, -1)
	if len(matches) == 0 {
		return nil
	}

	out := make([]string, 0, len(matches))
	seen := make(map[string]struct{}, len(matches))
	for _, match := range matches {
		if len(match) < 2 {
			continue
		}
		name := strings.TrimSpace(match[1])
		key := strings.ToLower(name)
		if name == "" {
			continue
		}
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, name)
	}

	return out
}

func formatSkillContextLine(item *skill.Skill) string {
	if item == nil {
		return ""
	}

	var parts []string
	name := strings.TrimSpace(item.Name)
	description := strings.TrimSpace(item.Description)
	if description == "" {
		description = "No description provided"
	}
	parts = append(parts, fmt.Sprintf("- %s: %s", name, description))

	if len(item.Tools) > 0 {
		parts = append(parts, fmt.Sprintf("tools=%s", strings.Join(item.Tools, ", ")))
	}
	if len(item.Tags) > 0 {
		parts = append(parts, fmt.Sprintf("tags=%s", strings.Join(item.Tags, ", ")))
	}

	content := compactSkillContent(item.Content)
	if content != "" {
		parts = append(parts, "guidance="+content)
	}

	return strings.Join(parts, " | ")
}

func compactSkillContent(content string) string {
	clean := strings.Join(strings.Fields(strings.TrimSpace(content)), " ")
	if len(clean) <= maxSkillContentChars {
		return clean
	}
	return clean[:maxSkillContentChars] + "..."
}

// Re-using decomposition logic but decoupled
type TaskDecomposer interface {
	ShouldDecompose(task string) bool
	Decompose(ctx context.Context, task string) ([]*SubTask, error)
}

type SubTask struct {
	ID           string   `json:"id"`
	Description  string   `json:"description"`
	Priority     int      `json:"priority"`
	Dependencies []string `json:"dependencies"` // IDs of tasks that must complete first
}

type LLMDecomposer struct {
	llm       cognitive.LLMClient
	threshold int
	promptCfg DecomposerPromptConfig
}

type DecomposerPromptConfig struct {
	System       string
	Requirements string
}

func NewDecomposer(llm cognitive.LLMClient, threshold int, promptCfg DecomposerPromptConfig) *LLMDecomposer {
	if threshold <= 0 {
		threshold = config.DefaultOrchestratorDecomposeWordThresh
	}
	if strings.TrimSpace(promptCfg.System) == "" {
		promptCfg.System = config.DefaultDecomposerSystemPrompt
	}
	if strings.TrimSpace(promptCfg.Requirements) == "" {
		promptCfg.Requirements = config.DefaultDecomposerRequirementsPrompt
	}

	return &LLMDecomposer{
		llm:       llm,
		threshold: threshold,
		promptCfg: promptCfg,
	}
}

func (d *LLMDecomposer) ShouldDecompose(task string) bool {
	return len(strings.Fields(task)) > d.threshold
}

func (d *LLMDecomposer) Decompose(ctx context.Context, task string) ([]*SubTask, error) {
	prompt := fmt.Sprintf(`
%s
GOAL: %s

%s
`, d.promptCfg.System, task, d.promptCfg.Requirements)

	response, err := d.llm.Complete(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("decomposition failed: %w", err)
	}

	normalized := cleanModelJSONBlock(response)
	subTasks, mode := parseDecompositionResponse(normalized, task)
	if mode != decompositionParseModeJSONArray {
		slog.Debug("Decomposer fallback parser used", "mode", mode, "sub_tasks", len(subTasks))
	}

	return subTasks, nil
}

type decompositionParseMode string

const (
	decompositionParseModeJSONArray  decompositionParseMode = "json_array"
	decompositionParseModeJSONObject decompositionParseMode = "json_object"
	decompositionParseModeExtracted  decompositionParseMode = "json_extracted"
	decompositionParseModeLineSplit  decompositionParseMode = "line_split"
	decompositionParseModeDefault    decompositionParseMode = "goal_default"
)

type subTaskPayload struct {
	SubTasksSnake []SubTask `json:"sub_tasks"`
	SubTasks      []SubTask `json:"subtasks"`
	Tasks         []SubTask `json:"tasks"`
	Items         []SubTask `json:"items"`
	Steps         []SubTask `json:"steps"`
	Plan          []SubTask `json:"plan"`
}

func parseDecompositionResponse(raw string, goal string) ([]*SubTask, decompositionParseMode) {
	normalized := cleanModelJSONBlock(raw)

	if tasks, ok := parseSubTaskArrayJSON(normalized); ok {
		return tasks, decompositionParseModeJSONArray
	}
	if tasks, ok := parseSubTaskObjectJSON(normalized); ok {
		return tasks, decompositionParseModeJSONObject
	}

	if extracted := extractFirstBalancedJSON(normalized, '[', ']'); extracted != "" {
		if tasks, ok := parseSubTaskArrayJSON(extracted); ok {
			return tasks, decompositionParseModeExtracted
		}
	}
	if extracted := extractFirstBalancedJSON(normalized, '{', '}'); extracted != "" {
		if tasks, ok := parseSubTaskObjectJSON(extracted); ok {
			return tasks, decompositionParseModeExtracted
		}
	}

	if tasks := parseSubTaskLines(normalized); len(tasks) > 0 {
		if len(tasks) == 1 && looksLikeControlToken(tasks[0].Description) {
			return defaultSubTasks(goal), decompositionParseModeDefault
		}
		return tasks, decompositionParseModeLineSplit
	}

	return defaultSubTasks(goal), decompositionParseModeDefault
}

func parseSubTaskArrayJSON(raw string) ([]*SubTask, bool) {
	if strings.TrimSpace(raw) == "" {
		return nil, false
	}
	var tasks []SubTask
	if err := json.Unmarshal([]byte(raw), &tasks); err != nil {
		return nil, false
	}
	normalized := normalizeSubTasks(tasks)
	if len(normalized) == 0 {
		return nil, false
	}
	return normalized, true
}

func parseSubTaskObjectJSON(raw string) ([]*SubTask, bool) {
	if strings.TrimSpace(raw) == "" {
		return nil, false
	}

	var payload subTaskPayload
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return nil, false
	}

	candidates := [][]SubTask{
		payload.SubTasksSnake,
		payload.SubTasks,
		payload.Tasks,
		payload.Items,
		payload.Steps,
		payload.Plan,
	}
	for _, candidate := range candidates {
		normalized := normalizeSubTasks(candidate)
		if len(normalized) > 0 {
			return normalized, true
		}
	}
	return nil, false
}

func parseSubTaskLines(raw string) []*SubTask {
	lines := strings.Split(raw, "\n")
	out := make([]*SubTask, 0, len(lines))
	for _, line := range lines {
		description := normalizeTaskLine(line)
		if description == "" {
			continue
		}
		out = append(out, &SubTask{
			ID:           fmt.Sprintf("task-%d", len(out)+1),
			Description:  description,
			Priority:     len(out) + 1,
			Dependencies: nil,
		})
	}
	return out
}

func normalizeSubTasks(tasks []SubTask) []*SubTask {
	out := make([]*SubTask, 0, len(tasks))
	used := make(map[string]struct{}, len(tasks))
	for _, task := range tasks {
		description := strings.TrimSpace(task.Description)
		if description == "" {
			continue
		}

		id := strings.TrimSpace(task.ID)
		if id == "" || id == "<nil>" {
			id = fmt.Sprintf("task-%d", len(out)+1)
		}
		if _, exists := used[id]; exists {
			id = fmt.Sprintf("task-%d", len(out)+1)
		}
		used[id] = struct{}{}

		priority := task.Priority
		if priority <= 0 {
			priority = len(out) + 1
		}

		deps := make([]string, 0, len(task.Dependencies))
		seenDeps := make(map[string]struct{}, len(task.Dependencies))
		for _, dep := range task.Dependencies {
			cleanDep := strings.TrimSpace(dep)
			if cleanDep == "" || cleanDep == id {
				continue
			}
			if _, exists := seenDeps[cleanDep]; exists {
				continue
			}
			seenDeps[cleanDep] = struct{}{}
			deps = append(deps, cleanDep)
		}

		out = append(out, &SubTask{
			ID:           id,
			Description:  description,
			Priority:     priority,
			Dependencies: deps,
		})
	}
	return out
}

func defaultSubTasks(goal string) []*SubTask {
	description := strings.TrimSpace(goal)
	if description == "" {
		description = "Execute the user goal safely."
	}
	return []*SubTask{
		{
			ID:           "task-1",
			Description:  description,
			Priority:     1,
			Dependencies: nil,
		},
	}
}

func normalizeTaskLine(line string) string {
	clean := strings.TrimSpace(line)
	if clean == "" {
		return ""
	}

	for {
		updated := false
		for _, prefix := range []string{"- ", "* ", "> "} {
			if strings.HasPrefix(clean, prefix) {
				clean = strings.TrimSpace(clean[len(prefix):])
				updated = true
			}
		}
		if !updated {
			break
		}
	}

	clean = trimLineNumberPrefix(clean)
	clean = strings.TrimSpace(clean)
	if clean == "" {
		return ""
	}
	return clean
}

func trimLineNumberPrefix(line string) string {
	if line == "" || !unicode.IsDigit(rune(line[0])) {
		return line
	}

	i := 0
	for i < len(line) && unicode.IsDigit(rune(line[i])) {
		i++
	}
	if i >= len(line) {
		return line
	}

	switch line[i] {
	case '.', ')', '-', ':':
		i++
	default:
		return line
	}

	for i < len(line) && unicode.IsSpace(rune(line[i])) {
		i++
	}
	if i >= len(line) {
		return ""
	}
	return line[i:]
}

func cleanModelJSONBlock(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "```json")
	s = strings.TrimPrefix(s, "```")
	s = strings.TrimSuffix(s, "```")
	return strings.TrimSpace(s)
}

func extractFirstBalancedJSON(input string, open, close byte) string {
	start := -1
	depth := 0
	inString := false
	escaped := false

	for i := 0; i < len(input); i++ {
		ch := input[i]
		if inString {
			if escaped {
				escaped = false
				continue
			}
			if ch == '\\' {
				escaped = true
				continue
			}
			if ch == '"' {
				inString = false
			}
			continue
		}

		switch ch {
		case '"':
			inString = true
		case open:
			if depth == 0 {
				start = i
			}
			depth++
		case close:
			if depth == 0 {
				continue
			}
			depth--
			if depth == 0 && start >= 0 {
				return strings.TrimSpace(input[start : i+1])
			}
		}
	}
	return ""
}

func looksLikeControlToken(s string) bool {
	token := strings.TrimSpace(s)
	if token == "" || strings.Contains(token, " ") || len(token) > 80 {
		return false
	}
	for _, r := range token {
		if unicode.IsUpper(r) || unicode.IsDigit(r) || r == '_' || r == '-' {
			continue
		}
		return false
	}
	return true
}
