package service

import (
	"context"
	"testing"

	"github.com/Tencent/WeKnora/internal/types"
)

func TestConfigureSkillsAllowsReadingWithoutScriptSandbox(t *testing.T) {
	t.Setenv("DIXIAN_KNOWLEDGE_SANDBOX_MODE", "disabled")
	config := &types.AgentConfig{}
	(&sessionService{}).configureSkillsFromAgent(context.Background(), config, &types.CustomAgent{
		Config: types.CustomAgentConfig{SkillsSelectionMode: "all"},
	})

	if !config.SkillsEnabled || len(config.SkillDirs) != 1 {
		t.Fatalf("read-only skills were disabled: %#v", config)
	}
}
