package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/Tencent/WeKnora/internal/agent/approval"
	"github.com/Tencent/WeKnora/internal/agent/tools"
	"github.com/Tencent/WeKnora/internal/application/service"
	apperrors "github.com/Tencent/WeKnora/internal/errors"
	"github.com/Tencent/WeKnora/internal/handler/session"
	"github.com/Tencent/WeKnora/internal/middleware"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Platform owns access and immutable versions; these private handlers reuse
// the native service and never list other users' agents or expose native auth.
type PlatformAgentHandler struct {
	service   interfaces.CustomAgentService
	models    interfaces.ModelService
	sessions  interfaces.SessionService
	chat      *session.Handler
	mcp       interfaces.MCPServiceService
	skills    interfaces.SkillService
	approvals *approval.Gate
}

func NewPlatformAgentHandler(s interfaces.CustomAgentService, models interfaces.ModelService, sessions interfaces.SessionService, chat *session.Handler, mcp interfaces.MCPServiceService, skills interfaces.SkillService, approvals *approval.Gate) *PlatformAgentHandler {
	return &PlatformAgentHandler{service: s, models: models, sessions: sessions, chat: chat, mcp: mcp, skills: skills, approvals: approvals}
}

// Execute delegates to the original QA handler. The platform supplies the
// caller's bounded source set; configuration is never rewritten for sharing.
func (h *PlatformAgentHandler) Execute(c *gin.Context) {
	grant, ok := middleware.PlatformAuthorizationContextFrom(c)
	if !ok || grant.Operation != middleware.OperationExecuteAgent {
		agentAPIError(c, 403, "AUTH_FORBIDDEN")
		return
	}
	var body struct {
		RunContext     string                           `json:"run_context"`
		UserID         string                           `json:"user_id"`
		RunID          string                           `json:"run_id"`
		SessionID      string                           `json:"session_id"`
		MaxSteps       int                              `json:"max_steps"`
		KnowledgeBases map[string][]string              `json:"knowledge_bases"`
		Request        session.CreateKnowledgeQARequest `json:"request"`
	}
	d := json.NewDecoder(http.MaxBytesReader(c.Writer, c.Request.Body, 72_000_000))
	d.DisallowUnknownFields()
	if d.Decode(&body) != nil || d.Decode(&struct{}{}) != io.EOF || body.UserID != grant.UserID || body.RunID == "" || body.SessionID == "" || body.RunContext == "" || body.MaxSteps < 1 || body.MaxSteps > 100 || body.Request.AgentID != c.Param("id") || body.Request.AgentSourceTenantID != 0 {
		agentAPIError(c, 400, "REQUEST_INVALID")
		return
	}
	if _, err := uuid.Parse(body.RunContext); err != nil {
		agentAPIError(c, 400, "REQUEST_INVALID")
		return
	}
	platformURL := strings.TrimRight(os.Getenv("DIXIAN_PLATFORM_INTERNAL_URL"), "/")
	token := os.Getenv("DIXIAN_DOCS_SERVICE_TOKEN")
	if platformURL == "" || token == "" {
		agentAPIError(c, 503, "DEPENDENCY_UNAVAILABLE")
		return
	}
	client := &http.Client{Timeout: 30 * time.Second}
	scope := &types.PlatformAgentScope{KnowledgeBases: body.KnowledgeBases, RequestContext: c.Request.Context(), MaxSteps: body.MaxSteps}
	scope.Authorize = func(ctx context.Context, action, id string, args json.RawMessage) error {
		payload, err := json.Marshal(map[string]interface{}{"user_id": body.UserID, "run_id": body.RunID, "session_id": body.SessionID, "action": action, "resource_id": id, "arguments": args})
		if err != nil {
			return err
		}
		r, err := http.NewRequestWithContext(ctx, http.MethodPost, platformURL+"/internal/v1/native-runs/"+body.RunContext+"/authorize", bytes.NewReader(payload))
		if err != nil {
			return err
		}
		r.Header.Set("Authorization", "Bearer "+token)
		r.Header.Set("X-Dixian-Service-Identity", "dixian-knowledge")
		r.Header.Set("X-Request-ID", c.GetHeader("X-Request-ID"))
		r.Header.Set("X-Log-Number", grant.LogNumber)
		r.Header.Set("Content-Type", "application/json")
		response, err := client.Do(r)
		if err != nil {
			return errors.New("platform authority unavailable")
		}
		defer response.Body.Close()
		if response.StatusCode != http.StatusOK {
			return errors.New("execution not authorized")
		}
		return nil
	}
	ctx := types.WithPlatformAgentScope(c.Request.Context(), scope)
	if err := types.AuthorizePlatformAgentAction(ctx, "step", "", nil); err != nil {
		agentAPIError(c, 403, "AUTH_FORBIDDEN")
		return
	}
	nativeSessionID := uuid.NewSHA1(uuid.NameSpaceOID, []byte(grant.OrganizationID+":"+body.UserID+":"+body.SessionID)).String()
	_, err := h.sessions.GetOwnedSession(ctx, nativeSessionID)
	if errors.Is(err, apperrors.ErrSessionNotFound) {
		_, err = h.sessions.CreateSession(ctx, &types.Session{ID: nativeSessionID, TenantID: types.MustTenantIDFromContext(ctx), UserID: body.UserID, Title: "专家会话"})
	}
	if err != nil {
		agentServiceError(c, err)
		return
	}
	data, err := json.Marshal(body.Request)
	if err != nil {
		agentAPIError(c, 400, "REQUEST_INVALID")
		return
	}
	c.Request = c.Request.WithContext(ctx)
	c.Request.Body = io.NopCloser(bytes.NewReader(data))
	c.Request.ContentLength = int64(len(data))
	c.Params = append(c.Params, gin.Param{Key: "session_id", Value: nativeSessionID})
	h.chat.AgentQA(c)
}

func (h *PlatformAgentHandler) Schema(c *gin.Context) {
	presets := []*types.CustomAgent{}
	for _, id := range types.GetBuiltinAgentIDs() {
		if a := types.GetBuiltinAgent(id, 0); a != nil {
			presets = append(presets, a)
		}
	}
	models, err := h.models.ListModels(c.Request.Context())
	if err != nil {
		agentServiceError(c, err)
		return
	}
	options := []gin.H{}
	for _, model := range models {
		if model.Status == types.ModelStatusActive {
			options = append(options, gin.H{"id": model.ID, "name": model.Name, "display_name": model.DisplayName, "type": model.Type})
		}
	}
	mcpOptions := []gin.H{}
	if h.mcp != nil {
		services, err := h.mcp.ListMCPServices(c.Request.Context(), types.MustTenantIDFromContext(c.Request.Context()))
		if err != nil {
			agentServiceError(c, err)
			return
		}
		for _, item := range services {
			mcpOptions = append(mcpOptions, gin.H{"id": item.ID, "name": item.Name, "description": item.Description, "enabled": item.Enabled})
		}
		sort.Slice(mcpOptions, func(i, j int) bool { return mcpOptions[i]["name"].(string) < mcpOptions[j]["name"].(string) })
	}
	skillOptions := []gin.H{}
	if h.skills != nil {
		items, err := h.skills.ListPreloadedSkills(c.Request.Context())
		if err != nil {
			agentServiceError(c, err)
			return
		}
		for _, item := range items {
			skillOptions = append(skillOptions, gin.H{"name": item.Name, "description": item.Description})
		}
		sort.Slice(skillOptions, func(i, j int) bool { return skillOptions[i]["name"].(string) < skillOptions[j]["name"].(string) })
	}
	c.JSON(http.StatusOK, gin.H{"fields": types.PlatformAgentConfigSchema(), "presets": presets, "models": options, "tools": tools.AvailableToolDefinitions(), "default_tools": tools.DefaultAllowedTools(), "mcp_services": mcpOptions, "skills": skillOptions})
}

// ResolveToolApproval resumes the exact native MCP call for the platform
// principal bound by the one-time execution grant.
func (h *PlatformAgentHandler) ResolveToolApproval(c *gin.Context) {
	if h.approvals == nil {
		agentAPIError(c, http.StatusServiceUnavailable, "DEPENDENCY_UNAVAILABLE")
		return
	}
	pendingID := c.Param("pending_id")
	if _, err := uuid.Parse(pendingID); err != nil {
		agentAPIError(c, http.StatusBadRequest, "REQUEST_INVALID")
		return
	}
	var body struct {
		Decision string `json:"decision"`
		Reason   string `json:"reason"`
	}
	d := json.NewDecoder(http.MaxBytesReader(c.Writer, c.Request.Body, 8_000))
	d.DisallowUnknownFields()
	if d.Decode(&body) != nil || d.Decode(&struct{}{}) != io.EOF || (body.Decision != "approve" && body.Decision != "reject") || len(body.Reason) > 500 {
		agentAPIError(c, http.StatusBadRequest, "REQUEST_INVALID")
		return
	}
	principal, ok := types.PrincipalFromContext(c.Request.Context())
	if !ok {
		agentAPIError(c, http.StatusForbidden, "AUTH_FORBIDDEN")
		return
	}
	err := h.approvals.Resolve(types.MustTenantIDFromContext(c.Request.Context()), principal.StorageID(), pendingID, approval.Decision{Approved: body.Decision == "approve", Reason: body.Reason})
	switch {
	case err == nil:
		c.JSON(http.StatusOK, gin.H{"resolved": true})
	case errors.Is(err, approval.ErrPendingNotFound):
		agentAPIError(c, http.StatusNotFound, "RESOURCE_NOT_FOUND")
	case errors.Is(err, approval.ErrAlreadyResolved):
		agentAPIError(c, http.StatusConflict, "RESOURCE_CONFLICT")
	default:
		agentAPIError(c, http.StatusForbidden, "AUTH_FORBIDDEN")
	}
}

func (h *PlatformAgentHandler) Validate(c *gin.Context) {
	data, err := io.ReadAll(http.MaxBytesReader(c.Writer, c.Request.Body, 256_000))
	if err != nil {
		agentAPIError(c, http.StatusBadRequest, "REQUEST_INVALID")
		return
	}
	config, err := types.ParsePlatformAgentConfig(data)
	if err != nil {
		agentAPIError(c, http.StatusBadRequest, "REQUEST_INVALID")
		return
	}
	c.JSON(http.StatusOK, gin.H{"config": config})
}

func (h *PlatformAgentHandler) Get(c *gin.Context) {
	a, err := h.service.GetAgentByID(c.Request.Context(), c.Param("id"))
	if err != nil {
		agentServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": a})
}

func (h *PlatformAgentHandler) Save(c *gin.Context) {
	if _, err := uuid.Parse(c.Param("id")); err != nil {
		agentAPIError(c, http.StatusBadRequest, "REQUEST_INVALID")
		return
	}
	var body struct {
		Name        string          `json:"name"`
		Description string          `json:"description"`
		Avatar      string          `json:"avatar"`
		Config      json.RawMessage `json:"config"`
	}
	data, err := io.ReadAll(http.MaxBytesReader(c.Writer, c.Request.Body, 300_000))
	if err != nil {
		agentAPIError(c, http.StatusBadRequest, "REQUEST_INVALID")
		return
	}
	d := json.NewDecoder(bytes.NewReader(data))
	d.DisallowUnknownFields()
	if err := d.Decode(&body); err != nil {
		agentAPIError(c, http.StatusBadRequest, "REQUEST_INVALID")
		return
	}
	if err := d.Decode(&struct{}{}); err != io.EOF {
		agentAPIError(c, http.StatusBadRequest, "REQUEST_INVALID")
		return
	}
	config, err := types.ParsePlatformAgentConfig(body.Config)
	if err != nil || strings.TrimSpace(body.Name) == "" || len(body.Name) > 255 || len(body.Description) > 100_000 || len(body.Avatar) > 64 {
		agentAPIError(c, http.StatusBadRequest, "REQUEST_INVALID")
		return
	}
	if !h.validateModels(c, config) {
		return
	}
	a, err := h.service.GetAgentByID(c.Request.Context(), c.Param("id"))
	if errors.Is(err, service.ErrAgentNotFound) {
		a, err = h.service.CreateAgent(c.Request.Context(), &types.CustomAgent{ID: c.Param("id"), Name: body.Name, Description: body.Description, Avatar: body.Avatar, Config: config})
	} else if err == nil {
		a.Name, a.Description, a.Avatar, a.Config = body.Name, body.Description, body.Avatar, config
		a, err = h.service.UpdateAgent(c.Request.Context(), a)
	}
	if err != nil {
		agentServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": a})
}

// Provisioning is the publication boundary. Draft parsing may omit a model,
// but a runnable version must use an active model from this native tenant.
func (h *PlatformAgentHandler) validateModels(c *gin.Context, config types.CustomAgentConfig) bool {
	if strings.TrimSpace(config.ModelID) == "" {
		agentAPIError(c, http.StatusBadRequest, "REQUEST_INVALID")
		return false
	}
	for _, setting := range []struct {
		id        string
		modelType types.ModelType
	}{
		{config.ModelID, types.ModelTypeKnowledgeQA}, {config.QueryUnderstandModelID, types.ModelTypeKnowledgeQA},
		{config.RerankModelID, types.ModelTypeRerank}, {config.VLMModelID, types.ModelTypeVLLM}, {config.ASRModelID, types.ModelTypeASR},
	} {
		if setting.id == "" {
			continue
		}
		model, err := h.models.GetModelByID(c.Request.Context(), setting.id)
		if err != nil || model == nil || model.Status != types.ModelStatusActive || model.Type != setting.modelType {
			agentAPIError(c, http.StatusBadRequest, "REQUEST_INVALID")
			return false
		}
	}
	return true
}

func (h *PlatformAgentHandler) Delete(c *gin.Context) {
	if err := h.service.DeleteAgent(c.Request.Context(), c.Param("id")); err != nil {
		agentServiceError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func agentServiceError(c *gin.Context, err error) {
	if errors.Is(err, service.ErrAgentNotFound) {
		agentAPIError(c, http.StatusNotFound, "RESOURCE_NOT_FOUND")
		return
	}
	agentAPIError(c, http.StatusInternalServerError, "DEPENDENCY_UNAVAILABLE")
}
func agentAPIError(c *gin.Context, status int, code string) {
	c.AbortWithStatusJSON(status, gin.H{"error_code": code, "log_number": c.GetHeader("X-Log-Number")})
}
