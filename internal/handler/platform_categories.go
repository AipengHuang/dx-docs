package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/Tencent/WeKnora/internal/errors"
	"github.com/Tencent/WeKnora/internal/types"
	secutils "github.com/Tencent/WeKnora/internal/utils"
)

type platformCategoryInput struct {
	ID        string `json:"id" binding:"required"`
	Name      string `json:"name" binding:"required"`
	SortOrder int    `json:"sort_order"`
}

type platformCategoryRequest struct {
	Categories []platformCategoryInput `json:"categories" binding:"required,min=1,max=100,dive"`
}

// CreatePlatformCategories 批量创建由 Platform 控制的项目分类标签。
func (h *TagHandler) CreatePlatformCategories(c *gin.Context) {
	var request platformCategoryRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.Error(errors.NewBadRequestError("Invalid project categories").WithDetails(err.Error()))
		return
	}
	kbID := secutils.SanitizeForLog(c.Param("id"))
	created := make([]*types.KnowledgeTag, 0, len(request.Categories))
	for _, category := range request.Categories {
		tag, err := h.tagService.CreatePlatformTag(
			c.Request.Context(), kbID, category.ID, category.Name, category.SortOrder,
		)
		if err != nil {
			c.Error(err)
			return
		}
		created = append(created, tag)
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": created})
}
