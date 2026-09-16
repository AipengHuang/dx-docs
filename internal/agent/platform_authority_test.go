package agent

import (
	"context"
	"github.com/Tencent/WeKnora/internal/types"
	"testing"
)

func TestEveryNativeModelAttemptRequiresPlatformAuthority(t *testing.T) {
	engine := &AgentEngine{}
	if _, err := engine.streamThinkingToEventBus(types.RequirePlatformAgentScope(context.Background()), nil, nil, 0, ""); err == nil {
		t.Fatal("model attempt proceeded without platform authority")
	}
}
