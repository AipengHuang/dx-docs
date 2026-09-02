package middleware

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"github.com/gin-gonic/gin"
)

func platformContext(operation ServiceOperation, audience string, expiresAt int64) PlatformAuthorizationContext {
	value := "org-1"
	return PlatformAuthorizationContext{
		Version:           1,
		ContextID:         "context-1",
		Audience:          audience,
		UserID:            "user-1",
		Username:          "user",
		OrganizationID:    "org-1",
		PermissionCodes:   []string{"knowledge:base:view"},
		RoleAssignmentIDs: []string{"assignment-1"},
		Permissions: []AuthorizationPermission{{
			PermissionCode: "knowledge:base:view",
			Grants: []AuthorizationGrant{{
				AssignmentID: "assignment-1",
				RoleCode:     "agent_user",
				SourceType:   "manual",
				ValidFrom:    "2026-09-01T00:00:00.000Z",
				Scopes:       []AuthorizationScope{{Type: "company", Value: &value}},
			}},
		}},
		ResourceScope: map[string][]string{
			"company": {"org-1"},
			"self":    {"user-1"},
		},
		Operation: operation,
		Target: AuthorizationTarget{
			Type:       "knowledge_base",
			ResourceID: "kb-1",
		},
		ResourceID: "kb-1",
		LogNumber:  "log-1",
		IssuedAt:   time.Now().Unix() - 1,
		ExpiresAt:  expiresAt,
	}
}

func signedPlatformContext(t *testing.T, context PlatformAuthorizationContext, secret string) string {
	t.Helper()
	payload, err := json.Marshal(context)
	if err != nil {
		t.Fatal(err)
	}
	encoded := base64.RawURLEncoding.EncodeToString(payload)
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(encoded))
	return encoded + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

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
	secret := "docs-token"
	token := signedPlatformContext(t, platformContext(
		OperationManageKnowledgeBase,
		"dixian-knowledge",
		time.Now().Unix()+60,
	), secret)
	platform := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(executionGrant{AuthorizationContext: token})
	}))
	defer platform.Close()
	t.Setenv("DIXIAN_PLATFORM_INTERNAL_URL", platform.URL)
	t.Setenv("DIXIAN_DOCS_SERVICE_TOKEN", secret)

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

func TestAuthorizationContextRejectsTamperingExpiryAndCrossService(t *testing.T) {
	secret := "docs-token"
	valid := platformContext(
		OperationViewKnowledgeBase,
		"dixian-knowledge",
		time.Now().Unix()+60,
	)
	token := signedPlatformContext(t, valid, secret)
	if _, err := verifyAuthorizationContext(token, secret, time.Now().Unix()); err != nil {
		t.Fatal(err)
	}
	if _, err := verifyAuthorizationContext(token+"changed", secret, time.Now().Unix()); err == nil {
		t.Fatal("tampered context was accepted")
	}
	expired := platformContext(OperationViewKnowledgeBase, "dixian-knowledge", time.Now().Unix())
	if _, err := verifyAuthorizationContext(
		signedPlatformContext(t, expired, secret), secret, time.Now().Unix(),
	); err == nil {
		t.Fatal("expired context was accepted")
	}
	wrongAudience := platformContext(
		OperationViewKnowledgeBase,
		"dixian-runtime",
		time.Now().Unix()+60,
	)
	if _, err := verifyAuthorizationContext(
		signedPlatformContext(t, wrongAudience, secret), secret, time.Now().Unix(),
	); err == nil {
		t.Fatal("cross-service context was accepted")
	}
}

func TestExecutionGrantUsesViewerAndExposesAuthorizationContext(t *testing.T) {
	gin.SetMode(gin.TestMode)
	secret := "docs-token"
	token := signedPlatformContext(t, platformContext(
		OperationViewKnowledgeBase,
		"dixian-knowledge",
		time.Now().Unix()+60,
	), secret)
	platform := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(executionGrant{AuthorizationContext: token})
	}))
	defer platform.Close()
	t.Setenv("DIXIAN_PLATFORM_INTERNAL_URL", platform.URL)
	t.Setenv("DIXIAN_DOCS_SERVICE_TOKEN", secret)

	service := &platformTenantStub{tenants: []*types.Tenant{{
		ID: 7, Name: "org-1", PlatformOrganizationID: "org-1",
	}}}
	router := gin.New()
	router.GET(
		"/knowledge-bases/:id",
		RequireExecutionGrant(OperationViewKnowledgeBase, "id", service),
		func(c *gin.Context) {
			context, ok := PlatformAuthorizationContextFrom(c)
			if !ok || context.UserID != "user-1" {
				c.Status(http.StatusInternalServerError)
				return
			}
			if types.TenantRoleFromContext(c.Request.Context()) != types.TenantRoleViewer {
				c.Status(http.StatusInternalServerError)
				return
			}
			c.Status(http.StatusOK)
		},
	)
	request := httptest.NewRequest(http.MethodGet, "/knowledge-bases/kb-1", nil)
	request.Header.Set(HeaderExecutionHandle, "one-time-handle")
	request.Header.Set(HeaderOrganizationID, "org-1")
	request.Header.Set(HeaderRequestID, "request-1")
	request.Header.Set(HeaderLogNumber, "log-1")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
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
