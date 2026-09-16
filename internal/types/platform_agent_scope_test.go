package types

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
)

func TestPlatformAgentScopeRejectsMissingRevokedAndBroadenedAuthority(t *testing.T) {
	ctx := RequirePlatformAgentScope(context.Background())
	if err := AuthorizePlatformAgentAction(ctx, "step", "", nil); err == nil {
		t.Fatal("missing authority was accepted")
	}
	calls := 0
	scope := &PlatformAgentScope{KnowledgeBases: map[string][]string{"kb-1": {"doc-1"}}, Authorize: func(context.Context, string, string, json.RawMessage) error { calls++; return errors.New("revoked") }}
	ctx = WithPlatformAgentScope(ctx, scope)
	if err := AuthorizePlatformAgentAction(ctx, "step", "", nil); err == nil || calls != 1 {
		t.Fatal("revocation was not checked")
	}
	if _, err := scope.SearchKnowledgeIDs("kb-2", nil); err == nil {
		t.Fatal("another KB was authorized")
	}
	ids, err := scope.SearchKnowledgeIDs("kb-1", nil)
	if err != nil || len(ids) != 1 || ids[0] != "doc-1" {
		t.Fatal("document restriction was lost")
	}
	if _, err := scope.SearchKnowledgeIDs("kb-1", []string{"doc-2"}); err == nil {
		t.Fatal("document scope was broadened")
	}
	cloned := CopyPlatformAgentScope(ctx, context.Background())
	if err := AuthorizePlatformAgentAction(cloned, "step", "", nil); err == nil || calls != 2 {
		t.Fatal("background execution lost platform authority")
	}
}
