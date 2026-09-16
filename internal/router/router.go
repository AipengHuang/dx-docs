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
	AgentHandler                 *handler.PlatformAgentHandler
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
	internal.GET("/agent-config", grant(params, middleware.OperationViewAgent, ""), params.AgentHandler.Schema)
	internal.POST("/agent-config/validate", grant(params, middleware.OperationManageAgent, ""), params.AgentHandler.Validate)
	internal.GET("/agents/:id", grant(params, middleware.OperationViewAgent, "id"), params.AgentHandler.Get)
	internal.PUT("/agents/:id", grant(params, middleware.OperationManageAgent, "id"), params.AgentHandler.Save)
	internal.DELETE("/agents/:id", grant(params, middleware.OperationManageAgent, "id"), params.AgentHandler.Delete)
	internal.POST("/agents/:id/execute", grant(params, middleware.OperationExecuteAgent, "id"), params.AgentHandler.Execute)
	internal.POST("/agents/:id/tool-approvals/:pending_id", grant(params, middleware.OperationExecuteAgent, "id"), params.AgentHandler.ResolveToolApproval)
	return router
}
