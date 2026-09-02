package handler

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/Tencent/WeKnora/internal/errors"
	"github.com/Tencent/WeKnora/internal/types"
	secutils "github.com/Tencent/WeKnora/internal/utils"
)

type platformKnowledgeTagUpdate struct {
	Updates map[string][]string `json:"updates" binding:"required,min=1"`
}

// UpdatePlatformKnowledgeTags 只接受执行句柄已经授权的路径知识库。
func (h *KnowledgeHandler) UpdatePlatformKnowledgeTags(c *gin.Context) {
	var request platformKnowledgeTagUpdate
	if err := c.ShouldBindJSON(&request); err != nil {
		c.Error(errors.NewBadRequestError("Invalid knowledge tag update").WithDetails(err.Error()))
		return
	}
	kbID := secutils.SanitizeForLog(c.Param("id"))
	_, _, tenantID, permission, err := h.validateKnowledgeBaseAccessWithKBID(c, kbID)
	if err != nil {
		c.Error(err)
		return
	}
	if permission != types.OrgRoleAdmin && permission != types.OrgRoleEditor {
		c.Error(errors.NewForbiddenError("No permission to update knowledge tags"))
		return
	}
	ctx := context.WithValue(c.Request.Context(), types.TenantIDContextKey, tenantID)
	if err := h.kgService.UpdateKnowledgeTagBatch(ctx, kbID, request.Updates); err != nil {
		c.Error(err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}
