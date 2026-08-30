package router

import (
	"github.com/gin-gonic/gin"

	"github.com/Tencent/WeKnora/internal/middleware"
)

func registerInternalKnowledgeRoutes(r *gin.RouterGroup, params RouterParams) {
	registerInternalKnowledgeBaseRoutes(r, params)
	registerInternalKnowledgeFileRoutes(r, params)
	registerInternalMetadataRoutes(r, params)
	registerInternalDataSourceRoutes(r, params)
}

func grant(params RouterParams, operation middleware.ServiceOperation, resourceParam string) gin.HandlerFunc {
	return middleware.RequireExecutionGrant(operation, resourceParam, params.TenantService)
}

func registerInternalKnowledgeBaseRoutes(r *gin.RouterGroup, params RouterParams) {
	h := params.KBHandler
	r.GET("/knowledge-bases", grant(params, middleware.OperationListKnowledgeBases, ""), h.ListKnowledgeBases)
	r.POST("/knowledge-bases", grant(params, middleware.OperationManageKnowledgeBase, ""), h.CreateKnowledgeBase)
	r.GET("/knowledge-bases/:id", grant(params, middleware.OperationViewKnowledgeBase, "id"), h.GetKnowledgeBase)
	r.PUT("/knowledge-bases/:id", grant(params, middleware.OperationManageKnowledgeBase, "id"), h.UpdateKnowledgeBase)
	r.DELETE("/knowledge-bases/:id", grant(params, middleware.OperationManageKnowledgeBase, "id"), h.DeleteKnowledgeBase)
	r.POST("/knowledge-bases/:id/hybrid-search", grant(params, middleware.OperationSearchKnowledge, "id"), h.HybridSearch)
	r.GET("/knowledge-bases/:id/activity", grant(params, middleware.OperationViewKnowledgeMetadata, "id"), params.AuditLogHandler.ListPlatformKnowledgeBaseActivity)
}

func registerInternalKnowledgeFileRoutes(r *gin.RouterGroup, params RouterParams) {
	h := params.KnowledgeHandler
	r.GET("/knowledge-bases/:id/knowledge", grant(params, middleware.OperationListKnowledgeFiles, "id"), h.ListKnowledge)
	r.GET("/knowledge-bases/:id/knowledge/folders", grant(params, middleware.OperationListKnowledgeFiles, "id"), h.ListKnowledgeFolders)
	r.PUT("/knowledge-bases/:id/knowledge/folders", grant(params, middleware.OperationManageKnowledgeFile, "id"), h.RenameKnowledgeFolder)
	r.POST("/knowledge-bases/:id/knowledge/file", grant(params, middleware.OperationManageKnowledgeFile, "id"), h.CreateKnowledgeFromFile)
	r.POST("/knowledge-bases/:id/knowledge/url", grant(params, middleware.OperationManageKnowledgeFile, "id"), h.CreateKnowledgeFromURL)
	r.POST("/knowledge-bases/:id/knowledge/manual", grant(params, middleware.OperationManageKnowledgeFile, "id"), h.CreateManualKnowledge)
	r.DELETE("/knowledge-bases/:id/knowledge", grant(params, middleware.OperationManageKnowledgeFile, "id"), h.ClearKnowledgeBaseContents)

	r.GET("/knowledge/:id", grant(params, middleware.OperationViewKnowledgeFile, "id"), h.GetKnowledge)
	r.GET("/knowledge/:id/stages", grant(params, middleware.OperationViewKnowledgeFile, "id"), h.GetKnowledgeSpans)
	r.GET("/knowledge/:id/spans", grant(params, middleware.OperationViewKnowledgeFile, "id"), h.GetKnowledgeSpans)
	r.GET("/knowledge/:id/preview", grant(params, middleware.OperationViewKnowledgeFile, "id"), h.PreviewKnowledgeFile)
	r.GET("/knowledge/:id/download", grant(params, middleware.OperationViewKnowledgeFile, "id"), h.DownloadKnowledgeFile)
	r.PUT("/knowledge/:id", grant(params, middleware.OperationManageKnowledgeFile, "id"), h.UpdateKnowledge)
	r.DELETE("/knowledge/:id", grant(params, middleware.OperationManageKnowledgeFile, "id"), h.DeleteKnowledge)
	r.POST("/knowledge/:id/reparse", grant(params, middleware.OperationManageKnowledgeFile, "id"), h.ReparseKnowledge)
	r.POST("/knowledge/:id/cancel-parse", grant(params, middleware.OperationManageKnowledgeFile, "id"), h.CancelKnowledgeParse)
	r.POST("/knowledge/:id/regenerate-summary", grant(params, middleware.OperationManageKnowledgeFile, "id"), h.RegenerateKnowledgeSummary)
	r.PUT("/knowledge/manual/:id", grant(params, middleware.OperationManageKnowledgeFile, "id"), h.UpdateManualKnowledge)

	chunks := params.ChunkHandler
	r.GET("/chunks/:knowledge_id", grant(params, middleware.OperationViewKnowledgeFile, "knowledge_id"), chunks.ListKnowledgeChunks)
	r.GET("/chunks/by-id/:id", grant(params, middleware.OperationViewKnowledgeFile, "id"), chunks.GetChunkByIDOnly)
	r.GET("/chunks/:knowledge_id/:id/revisions", grant(params, middleware.OperationViewKnowledgeFile, "knowledge_id"), chunks.ListChunkRevisions)
	r.PUT("/chunks/:knowledge_id/:id", grant(params, middleware.OperationManageKnowledgeFile, "knowledge_id"), chunks.UpdateChunk)
	r.DELETE("/chunks/:knowledge_id/:id", grant(params, middleware.OperationManageKnowledgeFile, "knowledge_id"), chunks.DeleteChunk)
	r.POST("/chunks/:knowledge_id/:id/revert", grant(params, middleware.OperationManageKnowledgeFile, "knowledge_id"), chunks.RevertChunk)
}

func registerInternalMetadataRoutes(r *gin.RouterGroup, params RouterParams) {
	faq := params.FAQHandler
	r.GET("/knowledge-bases/:id/faq/entries", grant(params, middleware.OperationViewKnowledgeMetadata, "id"), faq.ListEntries)
	r.GET("/knowledge-bases/:id/faq/entries/export", grant(params, middleware.OperationViewKnowledgeMetadata, "id"), faq.ExportEntries)
	r.GET("/knowledge-bases/:id/faq/entries/:entry_id", grant(params, middleware.OperationViewKnowledgeMetadata, "id"), faq.GetEntry)
	r.POST("/knowledge-bases/:id/faq/search", grant(params, middleware.OperationViewKnowledgeMetadata, "id"), faq.SearchFAQ)
	r.POST("/knowledge-bases/:id/faq/entry", grant(params, middleware.OperationManageKnowledgeMeta, "id"), faq.CreateEntry)
	r.POST("/knowledge-bases/:id/faq/entries", grant(params, middleware.OperationManageKnowledgeMeta, "id"), faq.UpsertEntries)
	r.PUT("/knowledge-bases/:id/faq/entries/:entry_id", grant(params, middleware.OperationManageKnowledgeMeta, "id"), faq.UpdateEntry)
	r.DELETE("/knowledge-bases/:id/faq/entries", grant(params, middleware.OperationManageKnowledgeMeta, "id"), faq.DeleteEntries)

	tags := params.TagHandler
	r.GET("/knowledge-bases/:id/tags", grant(params, middleware.OperationViewKnowledgeMetadata, "id"), tags.ListTags)
	r.POST("/knowledge-bases/:id/tags", grant(params, middleware.OperationManageKnowledgeMeta, "id"), tags.CreateTag)
	r.PUT("/knowledge-bases/:id/tags/:tag_id", grant(params, middleware.OperationManageKnowledgeMeta, "id"), tags.UpdateTag)
	r.DELETE("/knowledge-bases/:id/tags/:tag_id", grant(params, middleware.OperationManageKnowledgeMeta, "id"), tags.DeleteTag)

	wiki := params.WikiPageHandler
	wikiBase := "/knowledgebase/:kb_id/wiki"
	r.GET(wikiBase+"/pages", grant(params, middleware.OperationViewKnowledgeMetadata, "kb_id"), wiki.ListPages)
	r.GET(wikiBase+"/pages/*slug", grant(params, middleware.OperationViewKnowledgeMetadata, "kb_id"), wiki.GetPage)
	r.POST(wikiBase+"/pages", grant(params, middleware.OperationManageKnowledgeMeta, "kb_id"), wiki.CreatePage)
	r.PUT(wikiBase+"/pages/*slug", grant(params, middleware.OperationManageKnowledgeMeta, "kb_id"), wiki.UpdatePage)
	r.DELETE(wikiBase+"/pages/*slug", grant(params, middleware.OperationManageKnowledgeMeta, "kb_id"), wiki.DeletePage)
	r.GET(wikiBase+"/folders", grant(params, middleware.OperationViewKnowledgeMetadata, "kb_id"), wiki.ListFolders)
	r.POST(wikiBase+"/folders", grant(params, middleware.OperationManageKnowledgeMeta, "kb_id"), wiki.CreateFolder)
	r.PUT(wikiBase+"/folders/:folder_id", grant(params, middleware.OperationManageKnowledgeMeta, "kb_id"), wiki.UpdateFolder)
	r.DELETE(wikiBase+"/folders/:folder_id", grant(params, middleware.OperationManageKnowledgeMeta, "kb_id"), wiki.DeleteFolder)
	r.GET(wikiBase+"/search", grant(params, middleware.OperationViewKnowledgeMetadata, "kb_id"), wiki.SearchPages)
	r.GET(wikiBase+"/graph", grant(params, middleware.OperationViewKnowledgeMetadata, "kb_id"), wiki.GetGraph)
	r.GET(wikiBase+"/stats", grant(params, middleware.OperationViewKnowledgeMetadata, "kb_id"), wiki.GetStats)
}

func registerInternalDataSourceRoutes(r *gin.RouterGroup, params RouterParams) {
	h := params.DataSourceHandler
	op := middleware.OperationManageKnowledgeSource
	r.GET("/datasource/types", grant(params, op, ""), h.GetAvailableConnectors)
	r.POST("/datasource/validate-credentials", grant(params, op, ""), h.ValidateCredentials)
	r.GET("/datasource", grant(params, op, ""), h.ListDataSources)
	r.POST("/datasource", grant(params, op, ""), h.CreateDataSource)
	r.GET("/datasource/:id", grant(params, op, ""), h.GetDataSource)
	r.PUT("/datasource/:id", grant(params, op, ""), h.UpdateDataSource)
	r.DELETE("/datasource/:id", grant(params, op, ""), h.DeleteDataSource)
	r.POST("/datasource/:id/validate", grant(params, op, ""), h.ValidateConnection)
	r.GET("/datasource/:id/resources", grant(params, op, ""), h.ListAvailableResources)
	r.POST("/datasource/:id/sync", grant(params, op, ""), h.ManualSync)
	r.POST("/datasource/:id/pause", grant(params, op, ""), h.PauseDataSource)
	r.POST("/datasource/:id/resume", grant(params, op, ""), h.ResumeDataSource)
	r.GET("/datasource/:id/logs", grant(params, op, ""), h.GetSyncLogs)
	r.PUT("/datasource/:id/credentials", grant(params, op, ""), params.DataSourceCredentialsHandler.Put)
	r.DELETE("/datasource/:id/credentials/:field", grant(params, op, ""), params.DataSourceCredentialsHandler.DeleteField)
}
