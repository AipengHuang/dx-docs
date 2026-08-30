package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"github.com/gin-gonic/gin"
)

type platformTenantStub struct {
	interfaces.TenantService
	tenants []*types.Tenant
}

func (s *platformTenantStub) ListAllTenants(context.Context) ([]*types.Tenant, error) {
	return s.tenants, nil
}

func (s *platformTenantStub) CreateTenant(context.Context, *types.Tenant) (*types.Tenant, error) {
	s.tenants = []*types.Tenant{{ID: 7, Name: "org-1", PlatformOrganizationID: "org-1"}}
	return nil, errors.New("duplicate organization mapping")
}

func TestExecutionGrantRejectsWrongOperation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	platform := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
            "user_id":"user-1",
            "username":"user",
            "organization_id":"org-1",
            "operation":"knowledge.base:manage",
            "resource_id":"kb-1",
            "log_number":"log-1"
        }`))
	}))
	defer platform.Close()
	t.Setenv("DIXIAN_PLATFORM_INTERNAL_URL", platform.URL)
	t.Setenv("DIXIAN_DOCS_SERVICE_TOKEN", "docs-token")

	router := gin.New()
	router.GET(
		"/knowledge-bases/:id",
		RequireExecutionGrant(OperationViewKnowledgeBase, "id", nil),
		func(c *gin.Context) { c.Status(http.StatusOK) },
	)
	request := httptest.NewRequest(http.MethodGet, "/knowledge-bases/kb-1", nil)
	request.Header.Set(HeaderExecutionHandle, "one-time-handle")
	request.Header.Set(HeaderOrganizationID, "org-1")
	request.Header.Set(HeaderRequestID, "request-1")
	request.Header.Set(HeaderLogNumber, "log-1")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusForbidden)
	}
}

func TestResolvePlatformTenantReadsWinnerAfterConcurrentCreate(t *testing.T) {
	service := &platformTenantStub{}
	ginContext, _ := gin.CreateTestContext(httptest.NewRecorder())
	ginContext.Request = httptest.NewRequest(http.MethodGet, "/", nil)

	tenant, err := resolvePlatformTenant(ginContext, service, "org-1")
	if err != nil {
		t.Fatal(err)
	}
	if tenant.ID != 7 || tenant.PlatformOrganizationID != "org-1" {
		t.Fatalf("tenant = %#v", tenant)
	}
}
