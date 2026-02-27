package tool

import (
	"sort"
	"strings"

	"github.com/harunnryd/heike/internal/model/contract"
)

type RiskLevel string

const (
	RiskLow    RiskLevel = "low"
	RiskMedium RiskLevel = "medium"
	RiskHigh   RiskLevel = "high"
)

type ToolMetadata struct {
	Source       string
	Capabilities []string
	Risk         RiskLevel
}

type MetadataProvider interface {
	ToolMetadata() ToolMetadata
}

type ToolDescriptor struct {
	Definition contract.ToolDef
	Metadata   ToolMetadata
}

func normalizeToolMetadata(meta ToolMetadata) ToolMetadata {
	source := strings.TrimSpace(strings.ToLower(meta.Source))
	if source == "" {
		source = "runtime"
	}

	risk := RiskLevel(strings.TrimSpace(strings.ToLower(string(meta.Risk))))
	switch risk {
	case RiskLow, RiskMedium, RiskHigh:
	default:
		risk = RiskMedium
	}

	seen := make(map[string]struct{}, len(meta.Capabilities))
	capabilities := make([]string, 0, len(meta.Capabilities))
	for _, capability := range meta.Capabilities {
		normalized := normalizeCapability(capability)
		if normalized == "" {
			continue
		}
		if _, ok := seen[normalized]; ok {
			continue
		}
		seen[normalized] = struct{}{}
		capabilities = append(capabilities, normalized)
	}
	sort.Strings(capabilities)

	return ToolMetadata{
		Source:       source,
		Capabilities: capabilities,
		Risk:         risk,
	}
}

func normalizeCapability(in string) string {
	return strings.TrimSpace(strings.ToLower(in))
}
