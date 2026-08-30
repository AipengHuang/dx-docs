package router

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/dig"

	"github.com/Tencent/WeKnora/internal/handler"
	"github.com/Tencent/WeKnora/internal/middleware"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
)

// RouterParams 只保留统一知识服务需要的处理器。
type RouterParams struct {
	dig.In

	TenantService                interfaces.TenantService
	KBHandler                    *handler.KnowledgeBaseHandler
	KnowledgeHandler             *handler.KnowledgeHandler
	ChunkHandler                 *handler.ChunkHandler
	FAQHandler                   *handler.FAQHandler
	TagHandler                   *handler.TagHandler
	WikiPageHandler              *handler.WikiPageHandler
	DataSourceHandler            *handler.DataSourceHandler
	DataSourceCredentialsHandler *handler.DataSourceCredentialsHandler
	AuditLogHandler              *handler.AuditLogHandler
}

// NewRouter 创建仅供 Platform 调用的知识执行路由。
func NewRouter(params RouterParams) *gin.Engine {
	router := gin.New()
	router.ContextWithFallback = true
	router.Use(
		middleware.RequestID(),
		middleware.Language(),
		middleware.Logger(),
		middleware.Recovery(),
		middleware.ErrorHandler(),
		middleware.PlatformService(),
	)
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ready"})
	})

	internal := router.Group("/internal/v1")
	registerInternalKnowledgeRoutes(internal, params)
	return router
}
