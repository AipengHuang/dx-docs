package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
)

type ingestKBStub struct {
	interfaces.KnowledgeBaseService
	kbs     []*types.KnowledgeBase
	created int
}

func (s *ingestKBStub) ListKnowledgeBasesByTenantID(context.Context, uint64) ([]*types.KnowledgeBase, error) {
	return s.kbs, nil
}

func (s *ingestKBStub) CreateKnowledgeBase(_ context.Context, kb *types.KnowledgeBase) (*types.KnowledgeBase, error) {
	kb.ID = fmt.Sprintf("kb-%d", s.created)
	s.created++
	s.kbs = append(s.kbs, kb)
	return kb, nil
}

type ingestKnowledgeStub struct {
	interfaces.KnowledgeService
	passages     [][]string
	updated      []*types.Knowledge
	deleted      []string
	listed       []*types.Knowledge
	failIndexing bool
	lastCtx      context.Context
}

func (s *ingestKnowledgeStub) CreateKnowledgeFromPassageSync(ctx context.Context, kbID string, passage []string, _ string) (*types.Knowledge, error) {
	s.passages = append(s.passages, passage)
	s.lastCtx = ctx
	status := "enabled"
	if s.failIndexing {
		status = "disabled"
	}
	return &types.Knowledge{
		ID:              fmt.Sprintf("k-%d", len(s.passages)),
		KnowledgeBaseID: kbID,
		ParseStatus:     types.ParseStatusCompleted,
		EnableStatus:    status,
	}, nil
}

func (s *ingestKnowledgeStub) UpdateKnowledge(_ context.Context, k *types.Knowledge) error {
	s.updated = append(s.updated, k)
	return nil
}

func (s *ingestKnowledgeStub) DeleteKnowledge(_ context.Context, id string) error {
	s.deleted = append(s.deleted, id)
	return nil
}

func (s *ingestKnowledgeStub) ListKnowledgeByKnowledgeBaseID(_ context.Context, kbID string) ([]*types.Knowledge, error) {
	scoped := make([]*types.Knowledge, 0, len(s.listed))
	for _, doc := range s.listed {
		if doc.KnowledgeBaseID == kbID {
			scoped = append(scoped, doc)
		}
	}
	return scoped, nil
}

type ingestModelStub struct {
	interfaces.ModelService
}

func (ingestModelStub) ListModels(context.Context) ([]*types.Model, error) {
	return []*types.Model{{
		ID:     "bge-m3",
		Type:   types.ModelTypeEmbedding,
		Status: types.ModelStatusActive,
	}}, nil
}

func newIngestService(kb *ingestKBStub, ks *ingestKnowledgeStub, ttl time.Duration) *NativeAttachmentIngestService {
	return newNativeAttachmentIngestServiceWithTTL(kb, ks, ingestModelStub{}, ttl)
}

func TestNativeAttachmentIngestCreatesEphemeralKBOnce(t *testing.T) {
	kb, ks := &ingestKBStub{}, &ingestKnowledgeStub{}
	svc := newIngestService(kb, ks, 24*time.Hour)

	first, err := svc.EphemeralKB(context.Background(), 10000)
	if err != nil {
		t.Fatalf("ephemeral kb create failed: %v", err)
	}
	second, err := svc.EphemeralKB(context.Background(), 10000)
	if err != nil || first != second {
		t.Fatalf("ephemeral kb not reused: first=%s second=%s err=%v", first, second, err)
	}
	if kb.created != 1 {
		t.Fatalf("created %d knowledge bases, want 1", kb.created)
	}
	created := kb.kbs[0]
	if created.Name != nativeAttachmentKBName || created.TenantID != 10000 || created.EmbeddingModelID != "bge-m3" {
		t.Fatalf("unexpected ephemeral kb: %+v", created)
	}
}

func TestNativeAttachmentIngestChunksWholeDocumentAndTagsMetadata(t *testing.T) {
	kb, ks := &ingestKBStub{}, &ingestKnowledgeStub{}
	svc := newIngestService(kb, ks, 24*time.Hour)
	kbID, err := svc.EphemeralKB(context.Background(), 10000)
	if err != nil {
		t.Fatalf("ephemeral kb create failed: %v", err)
	}

	lines := make([]string, 0, 400)
	for i := 0; i < 400; i++ {
		lines = append(lines, fmt.Sprintf("ED-2026%04d | 料号A%d | 数量 128 | 2026-10-%02d", i, i%97, i%28+1))
	}
	content := strings.Join(lines, "\n")

	result, err := svc.Ingest(context.Background(), 10000, kbID, content, "对账单.xlsx", "conv-1")
	if err != nil {
		t.Fatalf("ingest failed: %v", err)
	}
	if result.KnowledgeID != "k-1" || result.ChunkCount < 2 {
		t.Fatalf("ingest result not chunked: %+v", result)
	}
	if len(ks.passages) != 1 || len(ks.passages[0]) != result.ChunkCount {
		t.Fatalf("passages recorded: %+v", ks.passages)
	}
	joined := strings.Join(ks.passages[0], "")
	if !strings.Contains(joined, lines[0]) || !strings.Contains(joined, lines[399]) {
		t.Fatal("chunked passages do not cover the whole document")
	}
	for _, passage := range ks.passages[0] {
		if len([]rune(passage)) > 2000 {
			t.Fatalf("oversized passage: %d runes", len([]rune(passage)))
		}
	}

	if len(ks.updated) != 1 {
		t.Fatalf("knowledge not tagged: %+v", ks.updated)
	}
	tagged := ks.updated[0]
	if tagged.ID != "k-1" || tagged.Title != "对账单.xlsx" || tagged.KnowledgeBaseID != kbID {
		t.Fatalf("unexpected knowledge identity: %+v", tagged)
	}
	var meta map[string]string
	if err := json.Unmarshal(tagged.CustomMetadata, &meta); err != nil {
		t.Fatalf("custom metadata invalid: %v", err)
	}
	if meta["kind"] != "native_temp_attachment" || meta["conversation_id"] != "conv-1" || meta["original_filename"] != "对账单.xlsx" {
		t.Fatalf("unexpected metadata: %v", meta)
	}
	expires, err := time.Parse(time.RFC3339, meta["expires_at"])
	if err != nil || time.Until(expires) <= 0 || time.Until(expires) > 25*time.Hour {
		t.Fatalf("unexpected expiry: %v %v", meta["expires_at"], err)
	}
}

func TestCleanupDoesNotProvisionKnowledgeBases(t *testing.T) {
	kb, ks := &ingestKBStub{}, &ingestKnowledgeStub{}
	svc := newIngestService(kb, ks, 24*time.Hour)

	svc.CleanupExpired(context.Background(), 10000)
	if kb.created != 0 {
		t.Fatalf("sweep created %d knowledge bases", kb.created)
	}
}

func TestIngestRejectsUnindexedKnowledge(t *testing.T) {
	kb, ks := &ingestKBStub{}, &ingestKnowledgeStub{}
	ks.failIndexing = true
	svc := newIngestService(kb, ks, 24*time.Hour)
	kbID, _ := svc.EphemeralKB(context.Background(), 10000)

	if _, err := svc.Ingest(context.Background(), 10000, kbID, strings.Repeat("行数据\n", 700), "a.csv", "conv"); err == nil {
		t.Fatal("unindexed knowledge must surface an error so the inline fallback runs")
	}
}

func TestIngestContextCarriesTenantInfo(t *testing.T) {
	kb, ks := &ingestKBStub{}, &ingestKnowledgeStub{}
	svc := newIngestService(kb, ks, 24*time.Hour)
	kbID, _ := svc.EphemeralKB(context.Background(), 10000)

	if _, err := svc.Ingest(context.Background(), 10000, kbID, strings.Repeat("行数据\n", 700), "a.csv", "conv-9"); err != nil {
		t.Fatalf("ingest failed: %v", err)
	}
	if ks.lastCtx.Value(types.TenantInfoContextKey) == nil || ks.lastCtx.Value(types.TenantIDContextKey) != uint64(10000) {
		t.Fatal("ingest context must carry both tenant keys for the deletion path")
	}
}

func TestSameNamedTenantKBIsNotReused(t *testing.T) {
	kb, ks := &ingestKBStub{}, &ingestKnowledgeStub{}
	kb.kbs = []*types.KnowledgeBase{{ID: "kb-user", Name: nativeAttachmentKBName, TenantID: 10000, Description: "user-made"}}
	svc := newIngestService(kb, ks, 24*time.Hour)

	id, err := svc.EphemeralKB(context.Background(), 10000)
	if err != nil {
		t.Fatalf("ephemeral kb create failed: %v", err)
	}
	if id == "kb-user" || kb.created != 1 {
		t.Fatalf("user-owned same-named KB was reused: id=%s created=%d", id, kb.created)
	}
	if created := kb.kbs[1]; created.Description != nativeAttachmentKBMarker {
		t.Fatalf("ephemeral KB missing ownership marker: %+v", created)
	}
}

func TestNativeAttachmentCleanupDeletesOnlyExpiredDocuments(t *testing.T) {
	kb, ks := &ingestKBStub{}, &ingestKnowledgeStub{}
	svc := newIngestService(kb, ks, 24*time.Hour)
	kbID, _ := svc.EphemeralKB(context.Background(), 10000)

	now := time.Now()
	ks.listed = []*types.Knowledge{
		{ID: "k-old", KnowledgeBaseID: kbID, CreatedAt: now.Add(-25 * time.Hour)},
		{ID: "k-new", KnowledgeBaseID: kbID, CreatedAt: now.Add(-time.Hour)},
		{ID: "k-foreign", KnowledgeBaseID: "kb-other", CreatedAt: now.Add(-48 * time.Hour)},
		// Crash between creation and the metadata update leaves rows without
		// the kind marker; they must still be reclaimed by age.
		{ID: "k-manual", KnowledgeBaseID: kbID, CreatedAt: now.Add(-48 * time.Hour)},
	}

	svc.CleanupExpired(context.Background(), 10000)
	if len(ks.deleted) != 2 || ks.deleted[0] != "k-old" || ks.deleted[1] != "k-manual" {
		t.Fatalf("cleanup deleted: %v", ks.deleted)
	}
}
