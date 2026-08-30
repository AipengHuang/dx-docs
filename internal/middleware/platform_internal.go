package middleware

import (
	"bytes"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"github.com/gin-gonic/gin"
)

const (
	HeaderExecutionHandle = "X-Dixian-Execution-Handle"
	HeaderLogNumber       = "X-Log-Number"
	HeaderOrganizationID  = "X-Dixian-Organization-ID"
	HeaderRequestID       = "X-Request-ID"
	HeaderServiceIdentity = "X-Dixian-Service-Identity"
)

type ServiceOperation string

const (
	OperationListKnowledgeBases    ServiceOperation = "knowledge.base:list"
	OperationViewKnowledgeBase     ServiceOperation = "knowledge.base:view"
	OperationManageKnowledgeBase   ServiceOperation = "knowledge.base:manage"
	OperationListKnowledgeFiles    ServiceOperation = "knowledge.file:list"
	OperationViewKnowledgeFile     ServiceOperation = "knowledge.file:view"
	OperationManageKnowledgeFile   ServiceOperation = "knowledge.file:manage"
	OperationSearchKnowledge       ServiceOperation = "knowledge.search"
	OperationViewKnowledgeMetadata ServiceOperation = "knowledge.metadata:view"
	OperationManageKnowledgeMeta   ServiceOperation = "knowledge.metadata:manage"
	OperationManageKnowledgeSource ServiceOperation = "knowledge.source:manage"
)

type executionGrant struct {
	UserID         string           `json:"user_id"`
	Username       string           `json:"username"`
	OrganizationID string           `json:"organization_id"`
	Operation      ServiceOperation `json:"operation"`
	ResourceID     string           `json:"resource_id"`
	LogNumber      string           `json:"log_number"`
}

var internalHTTPClient = &http.Client{Timeout: 5 * time.Second}

// PlatformService 验证唯一控制面对服务的调用身份。
func PlatformService() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !validPlatformRequest(c.Request) {
			abortInternal(c, http.StatusUnauthorized, "AUTH_UNAUTHORIZED", "Invalid service identity")
			return
		}
		requestID := c.GetHeader(HeaderRequestID)
		logNumber := c.GetHeader(HeaderLogNumber)
		if requestID == "" || logNumber == "" {
			abortInternal(c, http.StatusBadRequest, "REQUEST_CONTEXT_MISSING", "Request context is required")
			return
		}
		c.Header(HeaderRequestID, requestID)
		c.Header(HeaderLogNumber, logNumber)
		c.Next()
	}
}

// RequireExecutionGrant 兑换一次性授权并绑定到明确的操作和资源。
func RequireExecutionGrant(
	operation ServiceOperation,
	resourceParam string,
	tenantService interfaces.TenantService,
) gin.HandlerFunc {
	return func(c *gin.Context) {
		grant, err := redeemExecutionGrant(c.Request)
		if err != nil {
			abortInternal(c, http.StatusForbidden, "AUTH_FORBIDDEN", "Execution grant is invalid")
			return
		}
		organizationID := c.GetHeader(HeaderOrganizationID)
		resourceID := organizationID
		if resourceParam != "" {
			resourceID = c.Param(resourceParam)
		}
		if organizationID == "" || resourceID == "" ||
			grant.Operation != operation ||
			grant.OrganizationID != organizationID ||
			grant.ResourceID != resourceID ||
			grant.LogNumber != c.GetHeader(HeaderLogNumber) {
			abortInternal(c, http.StatusForbidden, "AUTH_FORBIDDEN", "Execution grant scope is invalid")
			return
		}
		tenant, err := resolvePlatformTenant(c, tenantService, organizationID)
		if err != nil {
			abortInternal(c, http.StatusInternalServerError, "TENANT_UNAVAILABLE", "Organization workspace is unavailable")
			return
		}
		user := &types.User{
			ID:       grant.UserID,
			Username: grant.Username,
			Email:    grant.UserID + "@platform.internal",
			TenantID: tenant.ID,
			IsActive: true,
		}
		applyAuthSession(c, authSession{
			User:      user,
			Principal: types.Principal{Type: types.PrincipalAPIPlatform, ID: grant.UserID},
			TenantID:  tenant.ID,
			Tenant:    tenant,
			Role:      types.TenantRoleOwner,
		})
		c.Next()
	}
}

func validPlatformRequest(request *http.Request) bool {
	scheme, token, found := strings.Cut(request.Header.Get("Authorization"), " ")
	if !found || scheme != "Bearer" || token == "" {
		return false
	}
	return equalSecret(token, os.Getenv("DIXIAN_PLATFORM_KNOWLEDGE_TOKEN")) &&
		equalSecret(request.Header.Get(HeaderServiceIdentity), os.Getenv("DIXIAN_PLATFORM_SERVICE_IDENTITY"))
}

func equalSecret(actual, expected string) bool {
	if actual == "" || expected == "" || len(actual) != len(expected) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(actual), []byte(expected)) == 1
}

func redeemExecutionGrant(request *http.Request) (*executionGrant, error) {
	handle := request.Header.Get(HeaderExecutionHandle)
	if handle == "" {
		return nil, errors.New("execution handle is required")
	}
	body, err := json.Marshal(map[string]string{"execution_handle": handle})
	if err != nil {
		return nil, err
	}
	url := strings.TrimRight(os.Getenv("DIXIAN_PLATFORM_INTERNAL_URL"), "/") +
		"/internal/v1/execution-handles/redeem"
	req, err := http.NewRequestWithContext(request.Context(), http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+os.Getenv("DIXIAN_DOCS_SERVICE_TOKEN"))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(HeaderServiceIdentity, "dixian-knowledge")
	req.Header.Set(HeaderRequestID, request.Header.Get(HeaderRequestID))
	req.Header.Set(HeaderLogNumber, request.Header.Get(HeaderLogNumber))
	response, err := internalHTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusCreated && response.StatusCode != http.StatusOK {
		return nil, errors.New("execution handle redemption failed")
	}
	var grant executionGrant
	if err := json.NewDecoder(response.Body).Decode(&grant); err != nil {
		return nil, err
	}
	return &grant, nil
}

func resolvePlatformTenant(c *gin.Context, tenantService interfaces.TenantService, organizationID string) (*types.Tenant, error) {
	tenants, err := tenantService.ListAllTenants(c.Request.Context())
	if err != nil {
		return nil, err
	}
	for _, tenant := range tenants {
		if tenant.PlatformOrganizationID == organizationID {
			return tenant, nil
		}
	}
	tenant, createErr := tenantService.CreateTenant(c.Request.Context(), &types.Tenant{
		Name:                   organizationID,
		Description:            "Platform organization workspace",
		PlatformOrganizationID: organizationID,
	})
	if createErr == nil {
		return tenant, nil
	}
	// 并发首次访问时唯一索引只允许一个映射，失败方重新读取即可。
	tenants, listErr := tenantService.ListAllTenants(c.Request.Context())
	if listErr != nil {
		return nil, createErr
	}
	for _, existing := range tenants {
		if existing.PlatformOrganizationID == organizationID {
			return existing, nil
		}
	}
	return nil, createErr
}

func abortInternal(c *gin.Context, status int, code, message string) {
	c.AbortWithStatusJSON(status, gin.H{
		"error_code": code,
		"message":    message,
		"log_number": c.GetHeader(HeaderLogNumber),
	})
}
