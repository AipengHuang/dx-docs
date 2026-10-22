package types

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
)

func TestOwnedKnowledgeBasesEnforceDocWhitelistWithoutPlatformAuthority(t *testing.T) {
	platformDenied := errors.New("platform denied")
	scope := &PlatformAgentScope{
		KnowledgeBases:       map[string][]string{"kb-platform": nil},
		OwnedKnowledgeBases:  map[string][]string{"kb-temp": {"doc-1"}},
		Authorize: func(context.Context, string, string, json.RawMessage) error {
			return platformDenied
		},
	}

	if ids, err := scope.SearchKnowledgeIDs("kb-temp", nil); err != nil || len(ids) != 1 || ids[0] != "doc-1" {
		t.Fatalf("owned kb unrestricted lookup must stay doc-whitelisted: ids=%v err=%v", ids, err)
	}
	if ids, err := scope.SearchKnowledgeIDs("kb-temp", []string{"doc-1"}); err != nil || len(ids) != 1 || ids[0] != "doc-1" {
		t.Fatalf("owned doc lookup: ids=%v err=%v", ids, err)
	}
	if _, err := scope.SearchKnowledgeIDs("kb-temp", []string{"doc-2"}); err == nil {
		t.Fatal("owned kb doc whitelist was broadened")
	}

	scopedCtx := WithPlatformAgentScope(context.Background(), scope)
	if err := AuthorizePlatformAgentDocument(scopedCtx, "kb-temp", "doc-1"); err != nil {
		t.Fatalf("owned doc required platform authority: %v", err)
	}
	if err := AuthorizePlatformAgentDocument(scopedCtx, "kb-temp", "doc-2"); err == nil {
		t.Fatal("owned kb allowed a document outside the whitelist")
	}
	if err := AuthorizePlatformAgentDocument(scopedCtx, "kb-platform", "doc-x"); !errors.Is(err, platformDenied) {
		t.Fatalf("platform kb bypassed the authorize callback: %v", err)
	}
}

func TestOwnedKnowledgeActionBypassesPlatformAuthorityForHybridSearch(t *testing.T) {
	platformDenied := errors.New("platform denied")
	scope := &PlatformAgentScope{
		OwnedKnowledgeBases: map[string][]string{"kb-temp": {"doc-1"}},
		Authorize: func(context.Context, string, string, json.RawMessage) error {
			return platformDenied
		},
	}
	scopedCtx := WithPlatformAgentScope(context.Background(), scope)
	if err := AuthorizePlatformAgentAction(scopedCtx, "knowledge", "kb-temp", nil); err != nil {
		t.Fatalf("owned KB hybrid search required platform authority: %v", err)
	}
	if err := AuthorizePlatformAgentAction(scopedCtx, "knowledge", "kb-other", nil); !errors.Is(err, platformDenied) {
		t.Fatalf("non-owned KB bypassed the authorize callback: %v", err)
	}
	if err := AuthorizePlatformAgentAction(scopedCtx, "step", "kb-temp", nil); !errors.Is(err, platformDenied) {
		t.Fatalf("non-knowledge actions must still call the platform: %v", err)
	}
}

func TestSearchKnowledgeIDsStillRejectsUnknownKnowledgeBases(t *testing.T) {
	scope := &PlatformAgentScope{
		KnowledgeBases:      map[string][]string{"kb-platform": nil},
		OwnedKnowledgeBases: map[string][]string{"kb-temp": {"doc-1"}},
	}
	if _, err := scope.SearchKnowledgeIDs("kb-other", nil); err == nil {
		t.Fatal("ungranted knowledge base passed scope check")
	}
	if _, err := scope.SearchKnowledgeIDs("kb-platform", nil); err != nil {
		t.Fatalf("platform kb unrestricted lookup failed: %v", err)
	}
}
