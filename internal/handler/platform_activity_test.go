package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Tencent/WeKnora/internal/middleware"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"github.com/gin-gonic/gin"
)

func TestPlatformKnowledgeBaseActivityUsesGrantContext(t *testing.T) {
	service := &stubAuditService{list: func(
		_ context.Context,
		tenantID uint64,
		query *interfaces.AuditLogQuery,
	) ([]*types.AuditLog, error) {
		if tenantID != 7 || query.ScopeType != "knowledge_base" || query.ScopeID != "kb-1" {
			t.Fatalf("unexpected activity scope: tenant=%d type=%q id=%q", tenantID, query.ScopeType, query.ScopeID)
		}
		return []*types.AuditLog{{ID: 21, TenantID: tenantID}}, nil
	}}

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(middleware.ErrorHandler())
	router.Use(func(c *gin.Context) {
		c.Set(types.TenantIDContextKey.String(), uint64(7))
		c.Next()
	})
	router.GET("/knowledge-bases/:id/activity", NewAuditLogHandler(service).ListPlatformKnowledgeBaseActivity)

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/knowledge-bases/kb-1/activity", nil)
	router.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", response.Code, response.Body.String())
	}
}
