package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Tencent/WeKnora/internal/application/repository"
	"github.com/Tencent/WeKnora/internal/middleware"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type platformKnowledgeBaseService struct {
	interfaces.KnowledgeBaseService
	existing *types.KnowledgeBase
	created  *types.KnowledgeBase
}

func (s *platformKnowledgeBaseService) GetKnowledgeBaseByID(
	context.Context, string,
) (*types.KnowledgeBase, error) {
	if s.existing == nil {
		return nil, repository.ErrKnowledgeBaseNotFound
	}
	return s.existing, nil
}

func (s *platformKnowledgeBaseService) CreateKnowledgeBase(
	_ context.Context, knowledgeBase *types.KnowledgeBase,
) (*types.KnowledgeBase, error) {
	s.created = knowledgeBase
	knowledgeBase.TenantID = 7
	return knowledgeBase, nil
}

func platformKnowledgeBaseRouter(service interfaces.KnowledgeBaseService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(middleware.ErrorHandler(), func(c *gin.Context) {
		c.Set(types.TenantIDContextKey.String(), uint64(7))
		c.Request = c.Request.WithContext(
			context.WithValue(c.Request.Context(), types.TenantIDContextKey, uint64(7)),
		)
		c.Next()
	})
	router.POST("/knowledge-bases/:id/provision", (&KnowledgeBaseHandler{service: service}).ProvisionPlatformKnowledgeBase)
	return router
}

func TestProvisionPlatformKnowledgeBaseIsIdempotent(t *testing.T) {
	service := &platformKnowledgeBaseService{existing: &types.KnowledgeBase{
		ID: "22222222-2222-4222-8222-222222222222", TenantID: 7,
		Type: types.KnowledgeBaseTypeDocument,
	}}
	request := httptest.NewRequest(http.MethodPost,
		"/knowledge-bases/22222222-2222-4222-8222-222222222222/provision",
		strings.NewReader(`{"name":"Project files","type":"document"}`))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()

	platformKnowledgeBaseRouter(service).ServeHTTP(response, request)

	require.Equal(t, http.StatusOK, response.Code)
	assert.Nil(t, service.created)
}

func TestProvisionPlatformKnowledgeBaseUsesPathID(t *testing.T) {
	service := &platformKnowledgeBaseService{}
	request := httptest.NewRequest(http.MethodPost,
		"/knowledge-bases/22222222-2222-4222-8222-222222222222/provision",
		strings.NewReader(`{"id":"ignored","name":"Project files","type":"document"}`))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()

	platformKnowledgeBaseRouter(service).ServeHTTP(response, request)

	require.Equal(t, http.StatusCreated, response.Code)
	require.NotNil(t, service.created)
	assert.Equal(t, "22222222-2222-4222-8222-222222222222", service.created.ID)
}
