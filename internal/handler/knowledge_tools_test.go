package handler

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/Tencent/WeKnora/internal/application/service"
	"github.com/Tencent/WeKnora/internal/middleware"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"github.com/gin-gonic/gin"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type bridgeKnowledge struct{ interfaces.KnowledgeService }

func (bridgeKnowledge) GetKnowledgeByIDOnly(context.Context, string) (*types.Knowledge, error) {
	return &types.Knowledge{ID: "foreign-doc", KnowledgeBaseID: "other-kb"}, nil
}

type bridgeWiki struct{ interfaces.WikiPageService }

func (bridgeWiki) ListIssues(context.Context, string, string, string) ([]*types.WikiPageIssue, error) {
	return nil, nil
}
func TestKnowledgeToolsBoundary(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewKnowledgeToolsHandler(nil, bridgeKnowledge{}, nil, bridgeWiki{}, nil)
	for _, tc := range []struct {
		name, tool, body string
		grant            bool
		status           int
		contains         string
	}{
		{"missing grant", "wiki_read_issue", `{"arguments":{"slug":"page"}}`, false, 403, ""},
		{"wrong resource", "wiki_read_issue", `{"arguments":{"slug":"page"}}`, true, 403, ""},
		{"unknown", "unknown", `{"arguments":{}}`, true, 400, ""},
		{"write denied", "wiki_write_page", `{"arguments":{}}`, true, 400, ""},
		{"unknown envelope", "wiki_read_issue", `{"arguments":{},"scope":"other-kb"}`, true, 400, ""},
		{"invalid args", "wiki_read_issue", `{"arguments":[]}`, true, 400, ""},
		{"missing args", "wiki_read_issue", `{}`, true, 400, ""},
		{"trailing json", "wiki_read_issue", `{"arguments":{}} {}`, true, 400, ""},
		{"oversized", "wiki_read_issue", `{"arguments":{"slug":"` + strings.Repeat("x", 65536) + `"}}`, true, 400, ""},
		{"cross KB", "query_knowledge_graph", `{"arguments":{"query":"secret","knowledge_base_ids":["other-kb"]}}`, true, 200, `"success":false`},
		{"foreign doc", "list_knowledge_chunks", `{"arguments":{"knowledge_id":"foreign-doc"}}`, true, 200, `"success":false`},
		{"success", "wiki_read_issue", `{"arguments":{"slug":"page"}}`, true, 200, `"success":true`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := gin.New()
			r.POST("/knowledge-bases/:id/tools/:tool", func(c *gin.Context) {
				if tc.grant {
					if tc.name == "wrong resource" {
						c.Params[0].Value = "other-kb"
					}
					c.Set("dixian.platform_authorization_context", &middleware.PlatformAuthorizationContext{Operation: middleware.OperationSearchKnowledge, ResourceID: "kb", Target: middleware.AuthorizationTarget{Type: "knowledge_base", ResourceID: "kb"}})
				}
				h.Execute(c)
			})
			w := httptest.NewRecorder()
			r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/knowledge-bases/kb/tools/"+tc.tool, strings.NewReader(tc.body)))
			if w.Code != tc.status || (tc.contains != "" && !strings.Contains(w.Body.String(), tc.contains)) {
				t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
			}
		})
	}
}

type bridgeAuthRepository struct {
	interfaces.KnowledgeRepository
	calls int
}

func (r *bridgeAuthRepository) GetKnowledgeByID(context.Context, uint64, string) (*types.Knowledge, error) {
	r.calls++
	if r.calls > 2 {
		return nil, errors.New("recursive document authorization")
	}
	return &types.Knowledge{ID: "doc", KnowledgeBaseID: "kb"}, nil
}
func (r *bridgeAuthRepository) GetKnowledgeByIDOnly(context.Context, string) (*types.Knowledge, error) {
	return &types.Knowledge{ID: "doc", KnowledgeBaseID: "kb"}, nil
}

type bridgeAuthBases struct {
	interfaces.KnowledgeBaseService
	knowledge interfaces.KnowledgeService
	action    string
}

func (b *bridgeAuthBases) GetKnowledgeBaseByIDOnly(context.Context, string) (*types.KnowledgeBase, error) {
	return &types.KnowledgeBase{ID: "kb", ExtractConfig: &types.ExtractConfig{Nodes: []*types.GraphNode{{}}}}, nil
}
func (b *bridgeAuthBases) HybridSearch(ctx context.Context, _ string, _ types.SearchParams) ([]*types.SearchResult, error) {
	if b.action == "step" {
		return nil, types.AuthorizePlatformAgentAction(ctx, "step", "", nil)
	}
	_, err := b.knowledge.GetKnowledgeByIDOnly(ctx, "doc")
	return nil, err
}
func TestKnowledgeToolsServiceAuthorization(t *testing.T) {
	for _, action := range []string{"step", "document"} {
		t.Run(action, func(t *testing.T) {
			repo := &bridgeAuthRepository{}
			knowledge, err := service.NewKnowledgeService(nil, repo, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
			if err != nil {
				t.Fatal(err)
			}
			bases := &bridgeAuthBases{knowledge: knowledge, action: action}
			h := NewKnowledgeToolsHandler(bases, knowledge, nil, nil, nil)
			r := gin.New()
			r.POST("/knowledge-bases/:id/tools/:tool", func(c *gin.Context) {
				c.Set("dixian.platform_authorization_context", &middleware.PlatformAuthorizationContext{Operation: middleware.OperationSearchKnowledge, ResourceID: "kb", Target: middleware.AuthorizationTarget{Type: "knowledge_base", ResourceID: "kb"}})
				c.Request = c.Request.WithContext(context.WithValue(c.Request.Context(), types.TenantIDContextKey, uint64(1)))
				h.Execute(c)
			})
			w := httptest.NewRecorder()
			r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/knowledge-bases/kb/tools/query_knowledge_graph", strings.NewReader(`{"arguments":{"query":"test","knowledge_base_ids":["kb"]}}`)))
			var result struct {
				Success bool
				Data    struct{ Errors []string }
			}
			if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
				t.Fatal(err)
			}
			if w.Code != 200 || !result.Success || len(result.Data.Errors) > 0 {
				t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
			}
			if repo.calls > 1 {
				t.Fatalf("recursive membership check: %d", repo.calls)
			}
		})
	}
}
