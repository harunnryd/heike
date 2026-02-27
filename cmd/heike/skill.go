package main

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/harunnryd/heike/internal/config"
	"github.com/harunnryd/heike/internal/runtime/discovery"
	skillmodel "github.com/harunnryd/heike/internal/skill"
	"github.com/harunnryd/heike/internal/skill/domain"
	"github.com/harunnryd/heike/internal/skill/formatter"
	"github.com/harunnryd/heike/internal/skill/loader"
	"github.com/harunnryd/heike/internal/skill/parser"

	"github.com/spf13/cobra"
)

var skillCmd = &cobra.Command{
	Use:   "skill",
	Short: "Manage skills",
	Long:  `Manage skills including install, uninstall, search, and show.`,
}

var skillInstallCmd = &cobra.Command{
	Use:   "install [path]",
	Short: "Install an external skill from path",
	Long:  `Install an external skill into the runtime project skill source. Bundled skills are auto-loaded and do not need installation.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		sourceDir, skillFile, err := resolveSkillSource(args[0])
		if err != nil {
			return err
		}

		parsedSkill, err := skillmodel.LoadSkillFromFile(skillFile)
		if err != nil {
			return fmt.Errorf("failed to parse skill file: %w", err)
		}

		wd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get working directory: %w", err)
		}
		skillSources, err := resolveRuntimeSkillSources(wd)
		if err != nil {
			return fmt.Errorf("resolve runtime skill sources: %w", err)
		}
		bundledSkillsRoot, _ := skillSourcePathByKind(skillSources, discovery.SourceBundled)
		if bundledSkillsRoot != "" && pathWithinDir(sourceDir, bundledSkillsRoot) {
			return fmt.Errorf("skill %q is in bundled runtime source and auto-loaded; install is only for external skills", parsedSkill.Name)
		}
		fmt.Printf("Installing skill from: %s\n", sourceDir)

		projectSkillsRoot, ok := skillSourcePathByKind(skillSources, discovery.SourceProject)
		if !ok {
			return fmt.Errorf("runtime project skill source is not configured")
		}
		installedPath := filepath.Join(projectSkillsRoot, parsedSkill.Name)
		if _, err := os.Stat(installedPath); err == nil {
			return fmt.Errorf("skill already installed: %s", parsedSkill.Name)
		} else if err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to inspect installed skill path: %w", err)
		}

		if err := copySkillDirectory(sourceDir, installedPath); err != nil {
			return fmt.Errorf("failed to copy skill files: %w", err)
		}

		toolLoader := loader.NewToolLoader()
		customTools, err := toolLoader.LoadFromSkill(installedPath)
		if err != nil {
			return fmt.Errorf("failed to load custom tools: %w", err)
		}

		fmt.Printf("Found %d custom tools\n", len(customTools))

		for _, ct := range customTools {
			fmt.Printf("  - %s (%s)\n", ct.Name, ct.Language)
		}

		fmt.Printf("Installed to: %s\n", installedPath)
		fmt.Println("✓ Skill installed successfully")
		return nil
	},
}

var skillUninstallCmd = &cobra.Command{
	Use:   "uninstall [name]",
	Short: "Uninstall a skill",
	Long:  `Remove a skill from the workspace.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		skillName := args[0]

		wd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get working directory: %w", err)
		}

		skillSources, err := resolveRuntimeSkillSources(wd)
		if err != nil {
			return fmt.Errorf("resolve runtime skill sources: %w", err)
		}
		projectSkillsRoot, ok := skillSourcePathByKind(skillSources, discovery.SourceProject)
		if !ok {
			return fmt.Errorf("runtime project skill source is not configured")
		}
		skillPath := filepath.Join(projectSkillsRoot, skillName)
		if _, err := os.Stat(skillPath); os.IsNotExist(err) {
			return fmt.Errorf("skill not found: %s", skillName)
		}

		if err := os.RemoveAll(skillPath); err != nil {
			return fmt.Errorf("failed to remove skill: %w", err)
		}

		fmt.Printf("✓ Skill '%s' uninstalled successfully\n", skillName)
		return nil
	},
}

var skillSearchCmd = &cobra.Command{
	Use:   "search [query]",
	Short: "Search for skills",
	Long:  `Search for skills in workspace and global skill directories.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		query := strings.ToLower(strings.TrimSpace(args[0]))
		if query == "" {
			return fmt.Errorf("query cannot be empty")
		}

		wd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get working directory: %w", err)
		}

		skillRegistry, warnings, err := loadRuntimeSkillRegistry(wd)
		if err != nil {
			return fmt.Errorf("load runtime skills: %w", err)
		}
		printSkillLoadWarnings(warnings)

		allSkills, err := skillRegistry.GetRelevant(query, 0)
		if err != nil {
			return fmt.Errorf("search skills: %w", err)
		}

		if len(allSkills) == 0 {
			fmt.Println("No skills found.")
			return nil
		}

		fmt.Println("=== Search Results ===")
		for _, item := range allSkills {
			if item == nil {
				continue
			}
			fmt.Printf("  - %s: %s\n", item.Name, item.Description)
		}

		return nil
	},
}

var skillShowCmd = &cobra.Command{
	Use:   "show [name]",
	Short: "Show skill details",
	Long:  `Display detailed information about a specific skill.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		skillName := args[0]

		wd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get working directory: %w", err)
		}

		skillSources, err := resolveRuntimeSkillSources(wd)
		if err != nil {
			return fmt.Errorf("resolve runtime skill sources: %w", err)
		}
		skillPath, err := resolveRuntimeSkillPath(skillName, skillSources)
		if err != nil {
			return fmt.Errorf("skill not found: %s", skillName)
		}

		fmt.Printf("=== Skill Details: %s ===\n", skillName)

		content, err := os.ReadFile(skillPath)
		if err != nil {
			return fmt.Errorf("failed to read skill file: %w", err)
		}

		skillParser := parser.NewYAMLFrontmatterParser()
		skill, err := skillParser.Parse(string(content))
		if err != nil {
			return fmt.Errorf("failed to parse skill: %w", err)
		}

		fmt.Printf("Name: %s\n", skill.Name)
		fmt.Printf("Description: %s\n", skill.Description)
		fmt.Printf("Tags: %v\n", skill.Tags)
		fmt.Printf("Version: %s\n", skill.Version)
		fmt.Printf("Author: %s\n", skill.Author)

		toolLoader := loader.NewToolLoader()
		customTools, err := toolLoader.LoadFromSkill(filepath.Dir(skillPath))
		if err != nil {
			return fmt.Errorf("failed to load custom tools: %w", err)
		}

		if len(customTools) > 0 {
			fmt.Println("\n=== Custom Tools ===")
			for _, ct := range customTools {
				fmt.Printf("  - %s (%s): %s\n", ct.Name, ct.Language, ct.Description)
			}
		}

		return nil
	},
}

var skillTestCmd = &cobra.Command{
	Use:   "test [name]",
	Short: "Dry-run validation of a skill",
	Long:  `Validate a skill's syntax and structure without executing it.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		skillName := args[0]

		wd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get working directory: %w", err)
		}

		skillSources, err := resolveRuntimeSkillSources(wd)
		if err != nil {
			return fmt.Errorf("resolve runtime skill sources: %w", err)
		}
		skillPath, err := resolveRuntimeSkillPath(skillName, skillSources)
		if err != nil {
			return fmt.Errorf("skill not found: %s", skillName)
		}

		fmt.Printf("Testing skill at: %s\n", skillPath)

		content, err := os.ReadFile(skillPath)
		if err != nil {
			return fmt.Errorf("failed to read skill file: %w", err)
		}

		skillParser := parser.NewYAMLFrontmatterParser()

		if err := skillParser.Validate(string(content)); err != nil {
			return fmt.Errorf("skill validation failed: %w", err)
		}

		skillRegistry, warnings, err := loadRuntimeSkillRegistry(wd)
		if err != nil {
			return fmt.Errorf("load runtime skills: %w", err)
		}
		printSkillLoadWarnings(warnings)

		loadedSkill, err := skillRegistry.Get(skillName)
		if err != nil {
			return fmt.Errorf("skill '%s' not found: %w", skillName, err)
		}
		if loadedSkill == nil || loadedSkill.Name != skillName {
			return fmt.Errorf("skill name mismatch: expected %s, got %s", skillName, loadedSkill.Name)
		}

		fmt.Printf("✓ Skill '%s' syntax is valid\n", skillName)
		fmt.Printf("✓ Skill loaded successfully\n")
		return nil
	},
}

var skillLsCmd = &cobra.Command{
	Use:   "ls",
	Short: "List all available skills",
	Long:  `Display all skills found in workspace and global directories.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		outputFormat, _ := cmd.Flags().GetString("output")

		wd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get working directory: %w", err)
		}

		skillRegistry, warnings, err := loadRuntimeSkillRegistry(wd)
		if err != nil {
			return fmt.Errorf("load runtime skills: %w", err)
		}
		printSkillLoadWarnings(warnings)

		skills, err := listRuntimeDomainSkills(skillRegistry)
		if err != nil {
			return fmt.Errorf("failed to list skills: %w", err)
		}
		if len(skills) == 0 {
			fmt.Println("No skills found in runtime sources.")
			return nil
		}

		formatterFactory := formatter.NewFormatterFactory()
		skillFormatter, err := formatterFactory.Create(formatter.OutputFormat(outputFormat))
		if err != nil {
			return fmt.Errorf("invalid output format: %w", err)
		}

		output, err := skillFormatter.FormatSkills(skills)
		if err != nil {
			return fmt.Errorf("failed to format output: %w", err)
		}

		fmt.Println(output)
		return nil
	},
}

func resolveSkillSource(rawPath string) (string, string, error) {
	clean := filepath.Clean(strings.TrimSpace(rawPath))
	if clean == "" {
		return "", "", fmt.Errorf("skill path cannot be empty")
	}

	info, err := os.Stat(clean)
	if err != nil {
		return "", "", fmt.Errorf("failed to access skill path %q: %w", clean, err)
	}

	if info.IsDir() {
		skillFile := filepath.Join(clean, "SKILL.md")
		if _, err := os.Stat(skillFile); err != nil {
			return "", "", fmt.Errorf("skill path %q is missing SKILL.md", clean)
		}
		return clean, skillFile, nil
	}

	if strings.EqualFold(filepath.Base(clean), "SKILL.md") {
		return filepath.Dir(clean), clean, nil
	}

	return "", "", fmt.Errorf("invalid skill path %q: provide a skill directory or SKILL.md file", clean)
}

func copySkillDirectory(srcDir, destDir string) error {
	srcDir = filepath.Clean(srcDir)
	destDir = filepath.Clean(destDir)

	if err := os.MkdirAll(destDir, 0755); err != nil {
		return err
	}

	return filepath.WalkDir(srcDir, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		rel, err := filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}

		targetPath := filepath.Join(destDir, rel)
		if d.IsDir() {
			return os.MkdirAll(targetPath, 0755)
		}

		info, err := d.Info()
		if err != nil {
			return err
		}
		return copyFile(path, targetPath, info.Mode().Perm())
	})
}

func pathWithinDir(path, root string) bool {
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return false
	}
	absPath, err := filepath.Abs(path)
	if err != nil {
		return false
	}
	rel, err := filepath.Rel(absRoot, absPath)
	if err != nil {
		return false
	}
	rel = filepath.Clean(rel)
	return rel == "." || (!strings.HasPrefix(rel, ".."+string(filepath.Separator)) && rel != "..")
}

func copyFile(src, dest string, perm fs.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, perm)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}

func init() {
	skillLsCmd.Flags().StringP("output", "o", "table", "Output format (table|json|yaml)")

	skillCmd.AddCommand(skillInstallCmd)
	skillCmd.AddCommand(skillUninstallCmd)
	skillCmd.AddCommand(skillSearchCmd)
	skillCmd.AddCommand(skillShowCmd)
	skillCmd.AddCommand(skillTestCmd)
	skillCmd.AddCommand(skillLsCmd)
	rootCmd.AddCommand(skillCmd)
}

func runtimeSkillDiscoveryOrder() []string {
	if cfg == nil || len(cfg.Discovery.SkillSources) == 0 {
		return []string{"bundled", "global", "workspace", "project"}
	}
	return append([]string(nil), cfg.Discovery.SkillSources...)
}

func runtimeWorkspaceRootPath() string {
	if cfg == nil {
		return ""
	}
	return cfg.Daemon.WorkspacePath
}

func runtimeProjectPath() string {
	if cfg == nil {
		return ""
	}
	return cfg.Discovery.ProjectPath
}

func resolveRuntimeSkillSources(workspacePath string) ([]discovery.SourceDescriptor, error) {
	return discovery.ResolveSkillSources(discovery.ResolveOptions{
		Order:             runtimeSkillDiscoveryOrder(),
		WorkspaceID:       config.DefaultWorkspaceID,
		WorkspaceRootPath: runtimeWorkspaceRootPath(),
		WorkspacePath:     workspacePath,
		ProjectPath:       runtimeProjectPath(),
	})
}

func skillSourcePathByKind(sources []discovery.SourceDescriptor, kind discovery.SourceKind) (string, bool) {
	var resolved string
	for _, source := range sources {
		if source.Kind != kind {
			continue
		}
		resolved = strings.TrimSpace(source.Path)
	}
	if resolved == "" {
		return "", false
	}
	return resolved, true
}

func resolveRuntimeSkillPath(skillName string, sources []discovery.SourceDescriptor) (string, error) {
	name := strings.TrimSpace(skillName)
	if name == "" {
		return "", fmt.Errorf("skill name cannot be empty")
	}

	resolved := ""
	for _, source := range sources {
		candidate := filepath.Join(source.Path, name, "SKILL.md")
		if _, err := os.Stat(candidate); err == nil {
			resolved = candidate
		}
	}
	if resolved == "" {
		return "", fmt.Errorf("skill not found: %s", name)
	}
	return resolved, nil
}

func loadRuntimeSkillRegistry(workspacePath string) (*skillmodel.Registry, []error, error) {
	sources, err := resolveRuntimeSkillSources(workspacePath)
	if err != nil {
		return nil, nil, err
	}

	registry := skillmodel.NewRegistry()
	warnings := make([]error, 0)
	for _, source := range sources {
		if err := registry.Load(source.Path); err != nil {
			warnings = append(warnings, fmt.Errorf("%s (%s): %w", source.Kind, source.Path, err))
		}
	}
	return registry, warnings, nil
}

func printSkillLoadWarnings(warnings []error) {
	for _, warn := range warnings {
		if warn == nil {
			continue
		}
		fmt.Fprintf(os.Stderr, "WARN skill load: %v\n", warn)
	}
}

func listRuntimeDomainSkills(registry *skillmodel.Registry) ([]*domain.Skill, error) {
	if registry == nil {
		return nil, nil
	}

	names, err := registry.List("name")
	if err != nil {
		return nil, err
	}
	sort.Strings(names)

	out := make([]*domain.Skill, 0, len(names))
	for _, name := range names {
		item, err := registry.Get(name)
		if err != nil || item == nil {
			continue
		}
		out = append(out, registrySkillToDomain(item))
	}
	return out, nil
}

func registrySkillToDomain(skill *skillmodel.Skill) *domain.Skill {
	if skill == nil {
		return nil
	}

	tags := make([]domain.Tag, 0, len(skill.Tags))
	for _, tag := range skill.Tags {
		tags = append(tags, domain.Tag(tag))
	}

	tools := make([]domain.ToolRef, 0, len(skill.Tools))
	for _, toolName := range skill.Tools {
		tools = append(tools, domain.ToolRef(toolName))
	}

	return &domain.Skill{
		ID:          domain.SkillID(skill.Name),
		Name:        skill.Name,
		Description: skill.Description,
		Tags:        tags,
		Tools:       tools,
		Content:     skill.Content,
	}
}
