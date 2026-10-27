package handler

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/Tencent/WeKnora/internal/agent/tools"
	"github.com/Tencent/WeKnora/internal/middleware"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type KnowledgeToolsHandler struct {
	bases     interfaces.KnowledgeBaseService
	knowledge interfaces.KnowledgeService
	chunks    interfaces.ChunkService
	wiki      interfaces.WikiPageService
	db        *gorm.DB
}

func NewKnowledgeToolsHandler(bases interfaces.KnowledgeBaseService, knowledge interfaces.KnowledgeService, chunks interfaces.ChunkService, wiki interfaces.WikiPageService, db *gorm.DB) *KnowledgeToolsHandler {
	return &KnowledgeToolsHandler{bases, knowledge, chunks, wiki, db}
}

func (h *KnowledgeToolsHandler) Execute(c *gin.Context) {
	grant, ok := middleware.PlatformAuthorizationContextFrom(c)
	kbID, name := c.Param("id"), c.Param("tool")
	if !ok || grant == nil || grant.Operation != middleware.OperationSearchKnowledge || kbID == "" || grant.ResourceID != kbID || grant.Target.Type != "knowledge_base" || grant.Target.ResourceID != kbID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Knowledge execution grant is required"})
		return
	}
	var body struct {
		Arguments map[string]json.RawMessage `json:"arguments"`
	}
	decoder := json.NewDecoder(http.MaxBytesReader(c.Writer, c.Request.Body, 64<<10))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&body); err != nil || body.Arguments == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid tool arguments"})
		return
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid tool request"})
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 120*time.Second)
	defer cancel()
	targets := types.SearchTargets{&types.SearchTarget{Type: types.SearchTargetTypeKnowledgeBase, KnowledgeBaseID: kbID}}
	var tool types.Tool
	switch name {
	case tools.ToolGetDocumentInfo:
		tool = tools.NewGetDocumentInfoTool(h.knowledge, h.chunks, targets)
	case tools.ToolListKnowledgeChunks:
		tool = tools.NewListKnowledgeChunksTool(h.knowledge, h.chunks, targets)
	case tools.ToolGrepChunks:
		kb, err := h.bases.GetKnowledgeBaseByID(ctx, kbID)
		if err != nil || kb == nil {
			c.JSON(http.StatusForbidden, gin.H{"error": "Knowledge base unavailable"})
			return
		}
		targets[0].TenantID = kb.TenantID
		tool = tools.NewGrepChunksTool(h.db, targets)
	case tools.ToolQueryKnowledgeGraph:
		tool = tools.NewQueryKnowledgeGraphTool(h.bases, targets).WithKnowledgeScope(h.knowledge)
	case tools.ToolWikiReadPage:
		tool = tools.NewWikiReadPageTool(h.wiki, h.knowledge, tools.NewWikiScopesFromSearchTargets(targets, []string{kbID}), tools.NewWikiRouteResolver())
	case tools.ToolWikiSearch:
		tool = tools.NewWikiSearchTool(h.wiki, h.knowledge, tools.NewWikiScopesFromSearchTargets(targets, []string{kbID}), tools.NewWikiRouteResolver())
	case tools.ToolWikiReadSourceDoc:
		tool = tools.NewWikiReadSourceDocTool(h.knowledge, h.chunks, targets)
	case tools.ToolWikiReadIssue:
		tool = tools.NewWikiReadIssueTool(h.wiki, []string{kbID})
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Unsupported knowledge tool"})
		return
	}
	scope := &types.PlatformAgentScope{RequestContext: ctx, KnowledgeBases: map[string][]string{kbID: nil}}
	scope.Authorize = func(_ context.Context, action, resource string, _ json.RawMessage) error {
		if (action == "step" && resource == "") || (action == "tool" && resource == name) || (action == "knowledge" && resource == kbID) {
			return nil
		}
		if action == "document" {
			// Membership lookup uses the original grant context to avoid re-entering this authorizer.
			document, err := h.knowledge.GetKnowledgeByIDOnly(ctx, resource)
			if err == nil && document != nil && document.KnowledgeBaseID == kbID {
				return nil
			}
		}
		return errors.New("knowledge tool action is outside the granted scope")
	}
	registry := tools.NewToolRegistry()
	registry.RegisterTool(tool)
	args, _ := json.Marshal(body.Arguments)
	result, err := registry.ExecuteTool(types.WithPlatformAgentScope(ctx, scope), name, args)
	if result == nil {
		result = &types.ToolResult{Success: false, Error: "Knowledge tool execution failed"}
	} else if err != nil {
		result.Success = false
	}
	c.JSON(http.StatusOK, result)
}
