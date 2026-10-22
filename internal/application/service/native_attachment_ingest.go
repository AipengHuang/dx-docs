package service

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/Tencent/WeKnora/internal/infrastructure/chunker"
	"github.com/Tencent/WeKnora/internal/logger"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
)

// nativeAttachmentKBName is the engine-owned knowledge base that receives
// oversized platform-run chat attachments so agents retrieve them with
// grep_chunks/list_knowledge_chunks instead of a truncated prompt.
const (
	nativeAttachmentKBName = "会话附件（临时）"
	// nativeAttachmentKBMarker guards against reusing a same-named tenant KB
	// that a user created by hand: only engine-owned bases are eligible.
	nativeAttachmentKBMarker = "dixian-engine:native-temp-attachments"
)

const (
	nativeAttachmentChunkSize   = 1600
	nativeAttachmentOverlap     = 160
	nativeAttachmentDefaultTTL  = 24 * time.Hour
	nativeAttachmentTTLEnvHours = "DIXIAN_NATIVE_ATTACHMENT_TTL_HOURS"
)

type NativeAttachmentIngestService struct {
	kbService        interfaces.KnowledgeBaseService
	knowledgeService interfaces.KnowledgeService
	modelService     interfaces.ModelService
	ttl              time.Duration

	mu      sync.Mutex
	kbCache map[uint64]string
}

func NewNativeAttachmentIngestService(
	kbService interfaces.KnowledgeBaseService,
	knowledgeService interfaces.KnowledgeService,
	modelService interfaces.ModelService,
) *NativeAttachmentIngestService {
	return newNativeAttachmentIngestServiceWithTTL(kbService, knowledgeService, modelService, nativeAttachmentTTLFromEnv())
}

func newNativeAttachmentIngestServiceWithTTL(
	kbService interfaces.KnowledgeBaseService,
	knowledgeService interfaces.KnowledgeService,
	modelService interfaces.ModelService,
	ttl time.Duration,
) *NativeAttachmentIngestService {
	return &NativeAttachmentIngestService{
		kbService:        kbService,
		knowledgeService: knowledgeService,
		modelService:     modelService,
		ttl:              ttl,
		kbCache:          make(map[uint64]string),
	}
}

func nativeAttachmentTTLFromEnv() time.Duration {
	if raw := os.Getenv(nativeAttachmentTTLEnvHours); raw != "" {
		if hours, err := strconv.Atoi(raw); err == nil && hours > 0 {
			return time.Duration(hours) * time.Hour
		}
	}
	return nativeAttachmentDefaultTTL
}

func withNativeAttachmentTenant(ctx context.Context, tenantID uint64) context.Context {
	ctx = context.WithValue(ctx, types.TenantIDContextKey, tenantID)
	// DeleteKnowledge asserts TenantInfoContextKey; keep the deletion path
	// (including the TTL sweeper) from panicking when callers omit it.
	if ctx.Value(types.TenantInfoContextKey) == nil {
		ctx = context.WithValue(ctx, types.TenantInfoContextKey, &types.Tenant{ID: tenantID})
	}
	return ctx
}

func (s *NativeAttachmentIngestService) EphemeralKB(ctx context.Context, tenantID uint64) (string, error) {
	return s.ephemeralKB(ctx, tenantID, true)
}

func (s *NativeAttachmentIngestService) FindEphemeralKB(ctx context.Context, tenantID uint64) (string, error) {
	return s.ephemeralKB(ctx, tenantID, false)
}

// ephemeralKB finds the engine-owned attachment KB; ensure=true creates it on
// first use, ensure=false is a lookup used by the TTL sweeper so it never
// provisions KBs for tenants that never ingested anything.
func (s *NativeAttachmentIngestService) ephemeralKB(ctx context.Context, tenantID uint64, ensure bool) (string, error) {
	ctx = withNativeAttachmentTenant(ctx, tenantID)
	s.mu.Lock()
	defer s.mu.Unlock()
	if id, ok := s.kbCache[tenantID]; ok {
		return id, nil
	}
	kbs, err := s.kbService.ListKnowledgeBasesByTenantID(ctx, tenantID)
	if err != nil {
		return "", err
	}
	for _, kb := range kbs {
		if kb.Name == nativeAttachmentKBName && kb.Description == nativeAttachmentKBMarker {
			s.kbCache[tenantID] = kb.ID
			return kb.ID, nil
		}
	}
	if !ensure {
		return "", nil
	}
	embeddingModelID, err := s.defaultEmbeddingModelID(ctx)
	if err != nil {
		return "", err
	}
	kb, err := s.kbService.CreateKnowledgeBase(ctx, &types.KnowledgeBase{
		Name:            nativeAttachmentKBName,
		Description:     nativeAttachmentKBMarker,
		Type:            "document",
		TenantID:        tenantID,
		EmbeddingModelID: embeddingModelID,
	})
	if err != nil {
		return "", err
	}
	s.kbCache[tenantID] = kb.ID
	return kb.ID, nil
}

func (s *NativeAttachmentIngestService) defaultEmbeddingModelID(ctx context.Context) (string, error) {
	models, err := s.modelService.ListModels(ctx)
	if err != nil {
		return "", err
	}
	for _, model := range models {
		if model.Type == types.ModelTypeEmbedding && model.Status == types.ModelStatusActive {
			return model.ID, nil
		}
	}
	return "", fmt.Errorf("no active embedding model for %s", nativeAttachmentKBName)
}

func (s *NativeAttachmentIngestService) Ingest(
	ctx context.Context, tenantID uint64, kbID, content, fileName, conversationID string,
) (types.NativeAttachmentIngestResult, error) {
	ctx = withNativeAttachmentTenant(ctx, tenantID)
	cfg := chunker.DefaultConfig()
	cfg.Strategy = chunker.StrategyAuto
	cfg.ChunkSize = nativeAttachmentChunkSize
	cfg.ChunkOverlap = nativeAttachmentOverlap
	parts := chunker.Split(content, cfg)
	if len(parts) == 0 {
		return types.NativeAttachmentIngestResult{}, fmt.Errorf("attachment %s produced no chunks", fileName)
	}
	passages := make([]string, 0, len(parts))
	for _, part := range parts {
		if text := part.EmbeddingContent(); text != "" {
			passages = append(passages, text)
		}
	}
	knowledge, err := s.knowledgeService.CreateKnowledgeFromPassageSync(ctx, kbID, passages, types.ChannelAPI)
	if err != nil {
		return types.NativeAttachmentIngestResult{}, err
	}
	// The sync path reports processing failures through the row state, not
	// its error: never promise retrieval unless chunks are actually enabled.
	if knowledge == nil || knowledge.ID == "" || knowledge.EnableStatus != "enabled" {
		return types.NativeAttachmentIngestResult{}, fmt.Errorf("attachment %s was not indexed (status=%s)", fileName, knowledgeStatus(knowledge))
	}

	knowledge.Title = fileName
	meta, err := json.Marshal(map[string]string{
		"kind":              "native_temp_attachment",
		"conversation_id":   conversationID,
		"original_filename": fileName,
		"expires_at":        time.Now().Add(s.ttl).Format(time.RFC3339),
	})
	if err != nil {
		return types.NativeAttachmentIngestResult{KnowledgeID: knowledge.ID, ChunkCount: len(passages)}, nil
	}
	knowledge.CustomMetadata = types.JSON(meta)
	if err := s.knowledgeService.UpdateKnowledge(ctx, knowledge); err != nil {
		// The document is already indexed and retrievable; only its label is stale.
		logger.Warnf(ctx, "native attachment metadata update failed: knowledge=%s err=%v", knowledge.ID, err)
	}
	return types.NativeAttachmentIngestResult{KnowledgeID: knowledge.ID, ChunkCount: len(passages)}, nil
}

func knowledgeStatus(k *types.Knowledge) string {
	if k == nil {
		return "nil"
	}
	return fmt.Sprintf("parse=%s enable=%s", k.ParseStatus, k.EnableStatus)
}

func (s *NativeAttachmentIngestService) ListTemporaryDocuments(ctx context.Context, tenantID uint64, kbID string) ([]*types.Knowledge, error) {
	ctx = withNativeAttachmentTenant(ctx, tenantID)
	return s.knowledgeService.ListKnowledgeByKnowledgeBaseID(ctx, kbID)
}

func (s *NativeAttachmentIngestService) CleanupExpired(ctx context.Context, tenantID uint64) {
	ctx = withNativeAttachmentTenant(ctx, tenantID)
	kbID, err := s.ephemeralKB(ctx, tenantID, false)
	if err != nil || kbID == "" {
		return
	}
	docs, err := s.knowledgeService.ListKnowledgeByKnowledgeBaseID(ctx, kbID)
	if err != nil {
		logger.Warnf(ctx, "native attachment cleanup list failed: %v", err)
		return
	}
	// The KB is engine-owned (marker-checked at lookup/creation), so every
	// expired document in it is a temporary attachment; delete by age alone.
	cutoff := time.Now().Add(-s.ttl)
	for _, doc := range docs {
		if !doc.CreatedAt.Before(cutoff) {
			continue
		}
		if err := s.knowledgeService.DeleteKnowledge(ctx, doc.ID); err != nil {
			logger.Warnf(ctx, "native attachment cleanup delete failed: knowledge=%s err=%v", doc.ID, err)
		}
	}
}
