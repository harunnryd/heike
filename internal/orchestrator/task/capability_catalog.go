package task

import (
	"sort"
	"strings"

	"github.com/harunnryd/heike/internal/tool"
)

// Capability describes a normalized tool capability for broker selection.
type Capability struct {
	Name         string
	Description  string
	Tags         []string
	Capabilities []string
	Source       string
	Risk         tool.RiskLevel
}

// CapabilityCatalog is a deterministic index of tool capabilities.
type CapabilityCatalog struct {
	ordered []Capability
	byName  map[string]Capability
}

func BuildCapabilityCatalog(tools []tool.ToolDescriptor) *CapabilityCatalog {
	unique := make(map[string]Capability, len(tools))
	for _, descriptor := range tools {
		def := descriptor.Definition
		name := strings.TrimSpace(strings.ToLower(def.Name))
		if name == "" {
			continue
		}

		if _, exists := unique[name]; exists {
			continue
		}

		unique[name] = Capability{
			Name:         name,
			Description:  strings.TrimSpace(def.Description),
			Tags:         deriveTags(descriptor),
			Capabilities: append([]string(nil), descriptor.Metadata.Capabilities...),
			Source:       inferSource(name, descriptor.Metadata),
			Risk:         descriptor.Metadata.Risk,
		}
	}

	names := make([]string, 0, len(unique))
	for name := range unique {
		names = append(names, name)
	}
	sort.Strings(names)

	caps := make([]Capability, 0, len(names))
	byName := make(map[string]Capability, len(names))
	for _, name := range names {
		c := unique[name]
		caps = append(caps, c)
		byName[name] = c
	}

	return &CapabilityCatalog{
		ordered: caps,
		byName:  byName,
	}
}

func (c *CapabilityCatalog) Capabilities() []Capability {
	if c == nil {
		return nil
	}
	out := make([]Capability, len(c.ordered))
	copy(out, c.ordered)
	return out
}

func (c *CapabilityCatalog) Find(name string) (Capability, bool) {
	if c == nil {
		return Capability{}, false
	}
	capability, ok := c.byName[strings.TrimSpace(strings.ToLower(name))]
	return capability, ok
}

func deriveTags(descriptor tool.ToolDescriptor) []string {
	def := descriptor.Definition
	tagSet := map[string]struct{}{}
	addTag := func(tag string) {
		tag = strings.TrimSpace(strings.ToLower(tag))
		if tag == "" {
			return
		}
		tagSet[tag] = struct{}{}
	}

	nameTokens := tokenize(def.Name)
	descTokens := tokenize(def.Description)

	for _, t := range nameTokens {
		addTag(t)
	}
	for _, t := range descTokens {
		// Keep description tags focused to avoid noise.
		if len(t) >= 4 {
			addTag(t)
		}
	}
	for _, capability := range descriptor.Metadata.Capabilities {
		for _, token := range tokenize(capability) {
			addTag(token)
		}
	}

	tags := make([]string, 0, len(tagSet))
	for t := range tagSet {
		tags = append(tags, t)
	}
	sort.Strings(tags)
	return tags
}

func inferSource(name string, meta tool.ToolMetadata) string {
	if source := strings.TrimSpace(strings.ToLower(meta.Source)); source != "" {
		return source
	}

	if tool.IsBuiltinName(name) {
		return "builtin"
	}

	switch {
	case strings.HasPrefix(name, "builtin."):
		return "builtin"
	case strings.HasPrefix(name, "skill."):
		return "skill"
	case strings.HasPrefix(name, "community."):
		return "community"
	case strings.HasPrefix(name, "org."):
		return "organization"
	default:
		return "runtime"
	}
}

func tokenize(text string) []string {
	normalized := strings.ToLower(strings.TrimSpace(text))
	if normalized == "" {
		return nil
	}
	replacer := strings.NewReplacer(
		".", " ",
		"_", " ",
		"-", " ",
		"/", " ",
		":", " ",
		",", " ",
	)
	normalized = replacer.Replace(normalized)
	return strings.Fields(normalized)
}
