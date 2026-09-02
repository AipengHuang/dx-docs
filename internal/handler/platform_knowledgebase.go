package handler

import (
	"errors"
	"net/http"
	"strings"

	"github.com/Tencent/WeKnora/internal/application/repository"
	apperrors "github.com/Tencent/WeKnora/internal/errors"
	"github.com/Tencent/WeKnora/internal/types"
	secutils "github.com/Tencent/WeKnora/internal/utils"
	"github.com/gin-gonic/gin"
)

// ProvisionPlatformKnowledgeBase 按 Platform 已授权的项目资源创建文档知识库。
func (h *KnowledgeBaseHandler) ProvisionPlatformKnowledgeBase(c *gin.Context) {
	knowledgeBaseID := secutils.SanitizeForLog(c.Param("id"))
	tenantID := c.GetUint64(types.TenantIDContextKey.String())
	if knowledgeBaseID == "" || tenantID == 0 {
		c.Error(apperrors.NewBadRequestError("Knowledge base context is required"))
		return
	}
	existing, err := h.service.GetKnowledgeBaseByID(c.Request.Context(), knowledgeBaseID)
	if err == nil {
		if existing.TenantID != tenantID || existing.Type != types.KnowledgeBaseTypeDocument {
			c.Error(apperrors.NewConflictError("Knowledge base id is already in use"))
			return
		}
		c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"id": existing.ID}})
		return
	}
	if !errors.Is(err, repository.ErrKnowledgeBaseNotFound) {
		c.Error(err)
		return
	}
	var request types.KnowledgeBase
	if err := c.ShouldBindJSON(&request); err != nil {
		c.Error(apperrors.NewBadRequestError("Invalid knowledge base request").WithDetails(err.Error()))
		return
	}
	request.ID = knowledgeBaseID
	request.Name = strings.TrimSpace(request.Name)
	if request.Name == "" || request.Type != types.KnowledgeBaseTypeDocument {
		c.Error(apperrors.NewBadRequestError("Document knowledge base name and type are required"))
		return
	}
	created, err := h.service.CreateKnowledgeBase(c.Request.Context(), &request)
	if err != nil {
		c.Error(err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"success": true, "data": gin.H{"id": created.ID}})
}
