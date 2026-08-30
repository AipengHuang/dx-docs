package handler

import (
	"net/http"

	"github.com/Tencent/WeKnora/internal/errors"
	"github.com/Tencent/WeKnora/internal/logger"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"github.com/gin-gonic/gin"
)

// ListPlatformKnowledgeBaseActivity 使用已校验的执行授权读取活动记录。
func (h *AuditLogHandler) ListPlatformKnowledgeBaseActivity(c *gin.Context) {
	tenantID := c.GetUint64(types.TenantIDContextKey.String())
	knowledgeBaseID := c.Param("id")
	if tenantID == 0 || knowledgeBaseID == "" {
		c.Error(errors.NewForbiddenError("Platform activity context is invalid"))
		return
	}

	afterID, limit := parseAuditCursor(c)
	query := &interfaces.AuditLogQuery{
		AfterID:     afterID,
		Limit:       limit,
		Action:      types.AuditAction(c.Query("action")),
		Outcome:     types.AuditOutcome(c.Query("outcome")),
		ActorUserID: c.Query("actor"),
		ScopeType:   "knowledge_base",
		ScopeID:     knowledgeBaseID,
	}
	entries, err := h.auditService.List(c.Request.Context(), tenantID, query)
	if err != nil {
		logger.ErrorWithFields(c.Request.Context(), err, map[string]interface{}{
			"knowledge_base_id": knowledgeBaseID,
		})
		c.Error(errors.NewInternalServerError(err.Error()))
		return
	}

	var nextCursor uint64
	if len(entries) > 0 {
		nextCursor = entries[len(entries)-1].ID
	}
	c.JSON(http.StatusOK, auditLogListResponse{
		Success: true, Data: entries, NextCursor: nextCursor,
	})
}
