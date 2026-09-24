package types

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestPlatformAgentConfigUsesCompleteNativeContract(t *testing.T) {
	config, err := ParsePlatformAgentConfig(json.RawMessage(`{"agent_mode":"smart-reasoning","agent_type":"wiki-qa","model_id":"model-1","temperature":0.4,"citation_enabled":false,"knowledge_bases":["kb-1"],"kb_selection_mode":"selected","question_suggestions":{"starters":{"enabled":true,"mode":"curated","items":["有哪些风险？"],"count":1},"follow_ups":{"enabled":false,"mode":"hybrid","count":3,"max_context_turns":2}}}`))
	if err != nil {
		t.Fatal(err)
	}
	if config.AgentMode != AgentModeSmartReasoning || config.CitationEnabled == nil || *config.CitationEnabled || config.QuestionSuggestions.Starters.Items[0] != "有哪些风险？" {
		t.Fatalf("native config lost: %#v", config)
	}
	schema := PlatformAgentConfigSchema()
	if len(schema) != reflect.TypeOf(CustomAgentConfig{}).NumField() {
		t.Fatal("schema omitted a native setting")
	}
	for _, raw := range []string{`null`, `[]`, `{} {}`, `{"unknown":true}`, `{"question_suggestions":{"unknown":true}}`, `{"temperature":2}`, `{"max_iterations":-1}`, `{"agent_mode":"local-engine"}`, `{"kb_selection_mode":"everything"}`} {
		if _, err := ParsePlatformAgentConfig(json.RawMessage(raw)); err == nil {
			t.Errorf("accepted invalid config: %s", raw)
		}
	}
}

func TestPlatformAgentConfigDropsUnusedRerankModel(t *testing.T) {
	config, err := ParsePlatformAgentConfig(json.RawMessage(`{"agent_mode":"smart-reasoning","model_id":"chat-model","rerank_model_id":"missing-reranker","kb_selection_mode":"none"}`))
	if err != nil {
		t.Fatal(err)
	}
	if config.RerankModelID != "" {
		t.Fatalf("unused rerank model was retained: %q", config.RerankModelID)
	}
}
