package types

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"slices"
	"strings"
)

// ParsePlatformAgentConfig validates the private API without maintaining a
// second configuration model. Native defaults remain owned by CustomAgent.
func ParsePlatformAgentConfig(data json.RawMessage) (CustomAgentConfig, error) {
	var config CustomAgentConfig
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 || len(data) > 256_000 || trimmed[0] != '{' {
		return config, fmt.Errorf("agent config must be a JSON object")
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&config); err != nil {
		return config, err
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return config, fmt.Errorf("agent config must contain one JSON object")
	}
	for field, setting := range map[string]struct {
		value   string
		allowed []string
	}{
		"agent_mode":            {config.AgentMode, []string{"", AgentModeQuickAnswer, AgentModeSmartReasoning}},
		"agent_type":            {config.AgentType, []string{"", AgentTypeRAGQA, AgentTypeWikiQA, AgentTypeHybridRAGWiki, AgentTypeDataAnalysis, AgentTypeCustom}},
		"kb_selection_mode":     {config.KBSelectionMode, []string{"", "all", "selected", "none"}},
		"mcp_selection_mode":    {config.MCPSelectionMode, []string{"", "all", "selected", "none"}},
		"skills_selection_mode": {config.SkillsSelectionMode, []string{"", "all", "selected", "none"}},
		"fallback_strategy":     {config.FallbackStrategy, []string{"", "fixed", "model"}},
	} {
		if !slices.Contains(setting.allowed, setting.value) {
			return config, fmt.Errorf("invalid %s", field)
		}
	}
	if config.Temperature < 0 || config.Temperature > 1 || config.MaxIterations < 0 || config.MaxIterations > 100 ||
		config.HistoryTurns < 0 || config.HistoryTurns > 100 || config.LLMCallTimeout < 0 || config.LLMCallTimeout > 3600 {
		return config, fmt.Errorf("agent model or iteration settings are outside allowed bounds")
	}
	if err := config.QuestionSuggestions.Validate(); err != nil {
		return config, err
	}
	agent := CustomAgent{Config: config}
	agent.EnsureDefaults()
	if agent.Config.KBSelectionMode == "none" {
		agent.Config.RerankModelID = ""
	}
	return agent.Config, nil
}

// PlatformAgentConfigSchema enumerates every native field, including nested
// settings. The portal can render these without a parallel list of 60 fields.
func PlatformAgentConfigSchema() map[string]any {
	return platformConfigFields(reflect.TypeOf(CustomAgentConfig{}))
}

func platformConfigFields(t reflect.Type) map[string]any {
	fields := make(map[string]any, t.NumField())
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		name := strings.Split(field.Tag.Get("json"), ",")[0]
		if name == "" || name == "-" {
			continue
		}
		kind := field.Type
		nullable := kind.Kind() == reflect.Pointer
		if nullable {
			kind = kind.Elem()
		}
		definition := map[string]any{"type": kind.Kind().String(), "nullable": nullable}
		if kind.Kind() == reflect.Struct {
			definition["properties"] = platformConfigFields(kind)
		}
		fields[name] = definition
	}
	return fields
}
