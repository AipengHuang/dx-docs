package types

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
)

type platformAgentScopeKey struct{}
type platformAgentRequiredKey struct{}

// PlatformAgentScope is attached only after a private execution grant. Nil
// document IDs allow the granted KB; a non-nil list restricts it before recall.
type PlatformAgentScope struct {
	RequestContext context.Context
	MaxSteps       int
	KnowledgeBases map[string][]string
	Authorize      func(context.Context, string, string, json.RawMessage) error
	ListSkills     func(context.Context) ([]PlatformSkillMetadata, error)
	ReadSkill      func(context.Context, string, string) (PlatformSkillContent, error)
	WriteSkill     func(context.Context, string, string) error
}

type PlatformSkillMetadata struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type PlatformSkillContent struct {
	Content string   `json:"content"`
	Files   []string `json:"files"`
}

func RequirePlatformAgentScope(ctx context.Context) context.Context {
	return context.WithValue(ctx, platformAgentRequiredKey{}, true)
}
func WithPlatformAgentScope(ctx context.Context, scope *PlatformAgentScope) context.Context {
	return context.WithValue(RequirePlatformAgentScope(ctx), platformAgentScopeKey{}, scope)
}
func CopyPlatformAgentScope(source, target context.Context) context.Context {
	if source.Value(platformAgentRequiredKey{}) != true {
		return target
	}
	if scope, ok := PlatformAgentScopeFromContext(source); ok {
		return WithPlatformAgentScope(target, scope)
	}
	return RequirePlatformAgentScope(target)
}
func PlatformAgentScopeFromContext(ctx context.Context) (*PlatformAgentScope, bool) {
	scope, ok := ctx.Value(platformAgentScopeKey{}).(*PlatformAgentScope)
	return scope, ok && scope != nil
}
func AuthorizePlatformAgentAction(ctx context.Context, action, resourceID string, args json.RawMessage) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if ctx.Value(platformAgentRequiredKey{}) != true {
		return nil
	}
	scope, ok := PlatformAgentScopeFromContext(ctx)
	if !ok || scope.Authorize == nil {
		return errors.New("platform agent authority is required")
	}
	if scope.RequestContext != nil {
		if err := scope.RequestContext.Err(); err != nil {
			return err
		}
	}
	return scope.Authorize(ctx, action, resourceID, args)
}
func (s *PlatformAgentScope) SearchKnowledgeIDs(kbID string, requested []string) ([]string, error) {
	allowed, ok := s.KnowledgeBases[kbID]
	if !ok || (allowed != nil && len(allowed) == 0) {
		return nil, errors.New("knowledge scope is unavailable")
	}
	if allowed == nil {
		return slices.Clone(requested), nil
	}
	for _, id := range requested {
		if !slices.Contains(allowed, id) {
			return nil, errors.New("document scope cannot be broadened")
		}
	}
	if len(requested) > 0 {
		return slices.Clone(requested), nil
	}
	return slices.Clone(allowed), nil
}

func AuthorizePlatformAgentDocument(ctx context.Context, kbID, documentID string) error {
	if scope, ok := PlatformAgentScopeFromContext(ctx); ok {
		if _, err := scope.SearchKnowledgeIDs(kbID, []string{documentID}); err != nil {
			return err
		}
	}
	args, err := json.Marshal(map[string]string{"knowledge_base_id": kbID})
	if err != nil {
		return err
	}
	return AuthorizePlatformAgentAction(ctx, "document", documentID, args)
}
