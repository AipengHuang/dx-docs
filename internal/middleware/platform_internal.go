package middleware

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
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

var knowledgeTargetTypes = map[ServiceOperation]string{
	OperationListKnowledgeBases:    "knowledge_base",
	OperationViewKnowledgeBase:     "knowledge_base",
	OperationManageKnowledgeBase:   "knowledge_base",
	OperationListKnowledgeFiles:    "knowledge_file",
	OperationViewKnowledgeFile:     "knowledge_file",
	OperationManageKnowledgeFile:   "knowledge_file",
	OperationSearchKnowledge:       "knowledge_base",
	OperationViewKnowledgeMetadata: "knowledge_metadata",
	OperationManageKnowledgeMeta:   "knowledge_metadata",
	OperationManageKnowledgeSource: "knowledge_source",
}

type executionGrant struct {
	AuthorizationContext string `json:"authorization_context"`
}

type AuthorizationScope struct {
	Type  string  `json:"type"`
	Value *string `json:"value,omitempty"`
}

type AuthorizationGrant struct {
	AssignmentID string               `json:"assignment_id"`
	RoleCode     string               `json:"role_code"`
	SourceType   string               `json:"source_type"`
	SourceEntity *string              `json:"source_entity_id,omitempty"`
	ValidFrom    string               `json:"valid_from"`
	ValidUntil   *string              `json:"valid_until,omitempty"`
	Scopes       []AuthorizationScope `json:"scopes"`
}

type AuthorizationPermission struct {
	PermissionCode string               `json:"permission_code"`
	Grants         []AuthorizationGrant `json:"grants"`
}

type AuthorizationTarget struct {
	Type       string `json:"type"`
	ResourceID string `json:"resource_id"`
	Collection bool   `json:"collection"`
}

type PlatformAuthorizationContext struct {
	Version           int                       `json:"version"`
	ContextID         string                    `json:"context_id"`
	Audience          string                    `json:"audience"`
	UserID            string                    `json:"user_id"`
	Username          string                    `json:"username"`
	OrganizationID    string                    `json:"organization_id"`
	PermissionCodes   []string                  `json:"permission_codes"`
	RoleAssignmentIDs []string                  `json:"role_assignment_ids"`
	Permissions       []AuthorizationPermission `json:"permissions"`
	ResourceScope     map[string][]string       `json:"resource_scope"`
	Operation         ServiceOperation          `json:"operation"`
	Target            AuthorizationTarget       `json:"target"`
	ResourceID        string                    `json:"resource_id"`
	ConversationID    *string                   `json:"conversation_id"`
	ProjectID         *string                   `json:"project_id"`
	LogNumber         string                    `json:"log_number"`
	IssuedAt          int64                     `json:"issued_at"`
	ExpiresAt         int64                     `json:"expires_at"`
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
			grant.Target.Type != knowledgeTargetTypes[operation] ||
			grant.Target.ResourceID != resourceID ||
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
			Role:      types.TenantRoleViewer,
		})
		c.Set(platformAuthorizationContextKey, grant)
		c.Next()
	}
}

const platformAuthorizationContextKey = "dixian.platform_authorization_context"

// PlatformAuthorizationContextFrom 返回本次真实用户的授权快照。
func PlatformAuthorizationContextFrom(c *gin.Context) (*PlatformAuthorizationContext, bool) {
	value, exists := c.Get(platformAuthorizationContextKey)
	context, valid := value.(*PlatformAuthorizationContext)
	return context, exists && valid
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

func redeemExecutionGrant(request *http.Request) (*PlatformAuthorizationContext, error) {
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
	return verifyAuthorizationContext(
		grant.AuthorizationContext,
		os.Getenv("DIXIAN_DOCS_SERVICE_TOKEN"),
		time.Now().Unix(),
	)
}

func verifyAuthorizationContext(token, secret string, now int64) (*PlatformAuthorizationContext, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" || secret == "" {
		return nil, errors.New("authorization context is invalid")
	}
	signature, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, errors.New("authorization context is invalid")
	}
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(parts[0]))
	if !hmac.Equal(signature, mac.Sum(nil)) {
		return nil, errors.New("authorization context is invalid")
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return nil, errors.New("authorization context is invalid")
	}
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.DisallowUnknownFields()
	var context PlatformAuthorizationContext
	if err := decoder.Decode(&context); err != nil {
		return nil, errors.New("authorization context is invalid")
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return nil, errors.New("authorization context is invalid")
	}
	if !validAuthorizationContext(&context, now) {
		return nil, errors.New("authorization context is invalid")
	}
	return &context, nil
}

func validAuthorizationContext(context *PlatformAuthorizationContext, now int64) bool {
	if context.Version != 1 || context.Audience != "dixian-knowledge" ||
		context.ContextID == "" || context.UserID == "" || context.Username == "" ||
		context.OrganizationID == "" || context.ResourceID == "" || context.LogNumber == "" ||
		context.Target.Type == "" || context.Target.ResourceID == "" ||
		context.IssuedAt >= context.ExpiresAt || context.ExpiresAt <= now ||
		len(context.PermissionCodes) == 0 || len(context.RoleAssignmentIDs) == 0 ||
		len(context.Permissions) == 0 ||
		!containsExact(context.ResourceScope["company"], context.OrganizationID) {
		return false
	}
	if selfScope := context.ResourceScope["self"]; len(selfScope) > 0 &&
		!containsExact(selfScope, context.UserID) {
		return false
	}
	permissionCodes := make(map[string]struct{}, len(context.Permissions))
	assignmentIDs := make(map[string]struct{}, len(context.RoleAssignmentIDs))
	for _, permission := range context.Permissions {
		if permission.PermissionCode == "" || len(permission.Grants) == 0 {
			return false
		}
		permissionCodes[permission.PermissionCode] = struct{}{}
		for _, grant := range permission.Grants {
			if grant.AssignmentID == "" || grant.RoleCode == "" || len(grant.Scopes) == 0 {
				return false
			}
			assignmentIDs[grant.AssignmentID] = struct{}{}
		}
	}
	return sameStringSet(context.PermissionCodes, permissionCodes) &&
		sameStringSet(context.RoleAssignmentIDs, assignmentIDs)
}

func containsExact(values []string, expected string) bool {
	for _, value := range values {
		if value == expected {
			return true
		}
	}
	return false
}

func sameStringSet(values []string, expected map[string]struct{}) bool {
	actual := make(map[string]struct{}, len(values))
	for _, value := range values {
		if value == "" {
			return false
		}
		actual[value] = struct{}{}
	}
	if len(actual) != len(expected) {
		return false
	}
	for value := range actual {
		if _, exists := expected[value]; !exists {
			return false
		}
	}
	return true
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
