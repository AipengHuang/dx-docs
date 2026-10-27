package router

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestNativeExecutionRetiredAndKnowledgeToolsRetained(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("DIXIAN_PLATFORM_KNOWLEDGE_TOKEN", "test-token")
	t.Setenv("DIXIAN_PLATFORM_SERVICE_IDENTITY", "platform")
	r := NewRouter(RouterParams{})
	for _, path := range []string{"/internal/v1/agents/agent/execute", "/internal/v1/agents/agent/tool-approvals/pending"} {
		req := httptest.NewRequest(http.MethodPost, path, nil)
		req.Header.Set("Authorization", "Bearer test-token")
		req.Header.Set("X-Dixian-Service-Identity", "platform")
		req.Header.Set("X-Request-ID", "request")
		req.Header.Set("X-Log-Number", "log")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusNotFound {
			t.Fatalf("retired route %s status=%d", path, w.Code)
		}
	}
	found := false
	for _, route := range r.Routes() {
		if route.Method == http.MethodPost && route.Path == "/internal/v1/knowledge-bases/:id/tools/:tool" {
			found = true
		}
	}
	if !found {
		t.Fatal("knowledge tool route missing")
	}
}
