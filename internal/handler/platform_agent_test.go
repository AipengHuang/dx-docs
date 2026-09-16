package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Tencent/WeKnora/internal/agent/approval"
	"github.com/Tencent/WeKnora/internal/agent/skills"
	"github.com/Tencent/WeKnora/internal/application/service"
	"github.com/Tencent/WeKnora/internal/config"
	"github.com/Tencent/WeKnora/internal/event"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"github.com/gin-gonic/gin"
)

type platformAgentServiceStub struct {
	interfaces.CustomAgentService
	agent  *types.CustomAgent
	writes int
}

type platformAgentModelsStub struct {
	interfaces.ModelService
	model *types.Model
}

func (s *platformAgentModelsStub) ListModels(context.Context) ([]*types.Model, error) {
	if s.model == nil {
		return nil, nil
	}
	return []*types.Model{s.model}, nil
}

type platformAgentMCPStub struct{ interfaces.MCPServiceService }

func (platformAgentMCPStub) ListMCPServices(context.Context, uint64) ([]*types.MCPService, error) {
	secret := "https://secret.example/mcp"
	return []*types.MCPService{{ID: "mcp-1", Name: "合同系统", Description: "查询合同状态", Enabled: true, URL: &secret}}, nil
}

type platformAgentSkillStub struct{ interfaces.SkillService }

type platformApprovalChecker struct{}

func (platformApprovalChecker) IsRequired(context.Context, uint64, string, string) (bool, error) {
	return true, nil
}

func (platformAgentSkillStub) ListPreloadedSkills(context.Context) ([]*skills.SkillMetadata, error) {
	return []*skills.SkillMetadata{{Name: "contract-review", Description: "检查合同条款", BasePath: "/secret/skills/contract-review"}}, nil
}

func (s *platformAgentModelsStub) GetModelByID(context.Context, string) (*types.Model, error) {
	return s.model, nil
}

func (s *platformAgentServiceStub) GetAgentByID(context.Context, string) (*types.CustomAgent, error) {
	if s.agent == nil {
		return nil, service.ErrAgentNotFound
	}
	return s.agent, nil
}
func (s *platformAgentServiceStub) CreateAgent(_ context.Context, a *types.CustomAgent) (*types.CustomAgent, error) {
	s.agent = a
	s.writes++
	return a, nil
}
func (s *platformAgentServiceStub) UpdateAgent(_ context.Context, a *types.CustomAgent) (*types.CustomAgent, error) {
	s.agent = a
	s.writes++
	return a, nil
}

func TestPlatformAgentProvisionUsesNativeServiceAndRejectsUnknownConfig(t *testing.T) {
	gin.SetMode(gin.TestMode)
	s := &platformAgentServiceStub{}
	h := NewPlatformAgentHandler(s, &platformAgentModelsStub{model: &types.Model{ID: "m-1", Type: types.ModelTypeKnowledgeQA, Status: types.ModelStatusActive}}, nil, nil, nil, nil, nil)
	r := gin.New()
	r.PUT("/agents/:id", h.Save)
	for _, body := range []string{`{"name":"质量专家","config":{"agent_mode":"smart-reasoning","model_id":"m-1","citation_enabled":false}}`, `{"name":"质量专家","config":{"agent_mode":"quick-answer","model_id":"m-1"}}`} {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodPut, "/agents/7c156fca-80f9-49d8-9147-394e89fa81b4", strings.NewReader(body)))
		if w.Code != http.StatusOK {
			t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
		}
	}
	if s.writes != 2 || s.agent.ID != "7c156fca-80f9-49d8-9147-394e89fa81b4" || s.agent.Config.AgentMode != types.AgentModeQuickAnswer {
		t.Fatal("native version provisioning failed")
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPut, "/agents/7c156fca-80f9-49d8-9147-394e89fa81b4", strings.NewReader(`{"name":"bad","config":{"secret_override":true}}`)))
	if w.Code != http.StatusBadRequest || s.writes != 2 {
		t.Fatal("invalid native settings reached persistence")
	}
}

func TestPlatformAgentProvisionRejectsUnavailableModelBeforeWriting(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, model := range []*types.Model{nil, {ID: "m-1", Type: types.ModelTypeEmbedding, Status: types.ModelStatusActive}} {
		s := &platformAgentServiceStub{}
		h := NewPlatformAgentHandler(s, &platformAgentModelsStub{model: model}, nil, nil, nil, nil, nil)
		r := gin.New()
		r.PUT("/agents/:id", h.Save)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodPut, "/agents/7c156fca-80f9-49d8-9147-394e89fa81b4", strings.NewReader(`{"name":"专家","config":{"model_id":"m-1"}}`)))
		if w.Code != http.StatusBadRequest || s.writes != 0 {
			t.Fatalf("unavailable model was provisioned: status=%d writes=%d", w.Code, s.writes)
		}
	}
}

func TestPlatformAgentExecutionRequiresOriginalCallerGrant(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewPlatformAgentHandler(&platformAgentServiceStub{}, nil, nil, nil, nil, nil, nil)
	r := gin.New()
	r.POST("/agents/:id/execute", h.Execute)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/agents/7c156fca-80f9-49d8-9147-394e89fa81b4/execute", strings.NewReader(`{"user_id":"another-user"}`)))
	if w.Code != http.StatusForbidden {
		t.Fatalf("execution without original authority: %d", w.Code)
	}
}

func TestPlatformAgentSchemaListsSafeNativeCapabilities(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewPlatformAgentHandler(
		&platformAgentServiceStub{},
		&platformAgentModelsStub{model: &types.Model{ID: "m-1", Name: "glm", DisplayName: "GLM", Type: types.ModelTypeKnowledgeQA, Status: types.ModelStatusActive}},
		nil,
		nil,
		platformAgentMCPStub{},
		platformAgentSkillStub{},
		nil,
	)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Request = c.Request.WithContext(context.WithValue(c.Request.Context(), types.TenantIDContextKey, uint64(1)))
		c.Next()
	})
	r.GET("/agent-config", h.Schema)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/agent-config", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	encoded := w.Body.String()
	if !strings.Contains(encoded, `"name":"合同系统"`) || !strings.Contains(encoded, `"name":"contract-review"`) || !strings.Contains(encoded, `"name":"knowledge_search"`) {
		t.Fatalf("native capability catalogs missing: %s", encoded)
	}
	if strings.Contains(encoded, "secret.example") || strings.Contains(encoded, "/secret/skills") {
		t.Fatalf("native capability catalog exposed private configuration: %s", encoded)
	}
}

func TestPlatformAgentResolvesNativeToolApprovalForOriginalPrincipal(t *testing.T) {
	gin.SetMode(gin.TestMode)
	gate := approval.NewGate(&config.Config{Agent: &config.AgentConfig{ToolApprovalTimeoutSeconds: 5}}, platformApprovalChecker{}, nil)
	bus := event.NewEventBus()
	pending := make(chan string, 1)
	bus.On(event.EventToolApprovalRequired, func(_ context.Context, value event.Event) error {
		pending <- value.Data.(event.ToolApprovalRequiredData).PendingID
		return nil
	})
	decision := make(chan approval.Decision, 1)
	go func() {
		result, _ := gate.RequestAndWait(context.Background(), approval.PendingRequest{
			TenantID: 1, UserID: "api_platform:user-1", EventBus: bus,
		})
		decision <- result
	}()

	h := NewPlatformAgentHandler(&platformAgentServiceStub{}, nil, nil, nil, nil, nil, gate)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		ctx := context.WithValue(c.Request.Context(), types.TenantIDContextKey, uint64(1))
		ctx = types.WithPrincipal(ctx, types.Principal{Type: types.PrincipalAPIPlatform, ID: "user-1"})
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	})
	r.POST("/approvals/:pending_id", h.ResolveToolApproval)
	id := <-pending
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/approvals/"+id, strings.NewReader(`{"decision":"approve"}`)))
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	select {
	case result := <-decision:
		if !result.Approved {
			t.Fatal("native tool approval was not delivered")
		}
	case <-time.After(time.Second):
		t.Fatal("native tool approval did not resume")
	}
}
