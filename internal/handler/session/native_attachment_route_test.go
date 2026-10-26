package session

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
)

type stubNativeIngestor struct {
	kbID        string
	ensureKBID  string
	ensureCalls int
	findCalls   int
	ingestErr   error
	calls       []string
	chunkCount  int
	aliveDocs   []string
}

func (s *stubNativeIngestor) EphemeralKB(context.Context, uint64) (string, error) {
	s.ensureCalls++
	if s.kbID == "" {
		s.kbID = s.ensureKBID
	}
	return s.kbID, nil
}

func (s *stubNativeIngestor) FindEphemeralKB(context.Context, uint64) (string, error) {
	s.findCalls++
	return s.kbID, nil
}

func (s *stubNativeIngestor) Ingest(_ context.Context, _ uint64, _ string, content, fileName, conversationID string) (types.NativeAttachmentIngestResult, error) {
	s.calls = append(s.calls, fileName+"|"+conversationID+"|"+content)
	if s.ingestErr != nil {
		return types.NativeAttachmentIngestResult{}, s.ingestErr
	}
	id := "k-" + fileName
	s.aliveDocs = append(s.aliveDocs, id)
	return types.NativeAttachmentIngestResult{KnowledgeID: id, ChunkCount: s.chunkCount}, nil
}

func (stubNativeIngestor) CleanupExpired(context.Context, uint64) {}

func (s *stubNativeIngestor) ListTemporaryDocuments(_ context.Context, _ uint64, _ string) ([]*types.Knowledge, error) {
	docs := make([]*types.Knowledge, 0, len(s.aliveDocs))
	for _, id := range s.aliveDocs {
		docs = append(docs, &types.Knowledge{ID: id})
	}
	return docs, nil
}

type stubMessageService struct {
	interfaces.MessageService
	messages []*types.Message
}

func (s *stubMessageService) GetRecentMessagesBySession(_ context.Context, _ string, _ int) ([]*types.Message, error) {
	return s.messages, nil
}

// 追问轮(无新超长附件)必须继承同会话历史里的临时文档检索范围。
func TestFollowUpTurnInheritsSessionTemporaryDocuments(t *testing.T) {
	rc := oversizedRequestContext()
	rc.attachments = nil // 追问轮不带附件
	scope := &types.PlatformAgentScope{KnowledgeBases: map[string][]string{"kb-1": nil}}
	ingestor := &stubNativeIngestor{kbID: "kb-temp", aliveDocs: []string{"k-old-1", "k-old-2"}}
	messages := &stubMessageService{messages: []*types.Message{
		{Attachments: types.MessageAttachments{
			{FileName: "a.xlsx", ContentMode: "indexed_document", IndexedKnowledgeID: "k-old-1"},
			{FileName: "b.csv", ContentMode: "indexed_document", IndexedKnowledgeID: "k-old-2"},
			{FileName: "dead.docx", ContentMode: "indexed_document", IndexedKnowledgeID: "k-deleted"},
		}},
	}}
	h := &Handler{nativeAttachments: ingestor, messageService: messages}

	h.routeOversizedAttachments(context.Background(), rc, scope, 10000, "conv-1")

	if ingestor.ensureCalls != 0 || ingestor.findCalls != 1 {
		t.Fatalf("follow-up must only find the existing KB: ensure=%d find=%d", ingestor.ensureCalls, ingestor.findCalls)
	}
	owned := scope.OwnedKnowledgeBases["kb-temp"]
	if len(owned) != 2 || owned[0] != "k-old-1" || owned[1] != "k-old-2" {
		t.Fatalf("session documents not re-scoped (dead docs must drop): %v", owned)
	}
	if !containsIDs(rc.knowledgeIDs, "k-old-1") || containsIDs(rc.knowledgeIDs, "k-deleted") {
		t.Fatalf("knowledge ids wrong: %v", rc.knowledgeIDs)
	}
	if !containsIDs(rc.knowledgeBaseIDs, "kb-temp") {
		t.Fatalf("kb not appended: %v", rc.knowledgeBaseIDs)
	}
}

func TestFirstOversizedAttachmentEnsuresTemporaryKnowledgeBase(t *testing.T) {
	rc := oversizedRequestContext()
	scope := &types.PlatformAgentScope{}
	ingestor := &stubNativeIngestor{ensureKBID: "kb-temp", chunkCount: 42}
	h := &Handler{nativeAttachments: ingestor}

	h.routeOversizedAttachments(context.Background(), rc, scope, 10000, "conv-1")

	if ingestor.ensureCalls != 1 || ingestor.findCalls != 0 || len(ingestor.calls) != 1 {
		t.Fatalf("first upload must create its KB and ingest: ensure=%d find=%d ingest=%d", ingestor.ensureCalls, ingestor.findCalls, len(ingestor.calls))
	}
	if rc.attachments[0].ContentMode != "indexed_document" {
		t.Fatalf("first upload was not indexed: %+v", rc.attachments[0])
	}
}

func TestFollowUpWithoutTemporaryKnowledgeBaseDoesNotCreateOne(t *testing.T) {
	rc := &qaRequestContext{}
	scope := &types.PlatformAgentScope{}
	ingestor := &stubNativeIngestor{ensureKBID: "kb-temp"}
	h := &Handler{nativeAttachments: ingestor}

	h.routeOversizedAttachments(context.Background(), rc, scope, 10000, "conv-1")

	if ingestor.ensureCalls != 0 || ingestor.findCalls != 1 || ingestor.kbID != "" {
		t.Fatalf("pure follow-up provisioned a KB: %+v", ingestor)
	}
	if len(scope.OwnedKnowledgeBases) != 0 || len(rc.knowledgeIDs) != 0 {
		t.Fatal("pure follow-up gained document access without an existing KB")
	}
}

func TestNewAttachmentDoesNotGrantOtherSessionDocuments(t *testing.T) {
	rc := oversizedRequestContext()
	scope := &types.PlatformAgentScope{}
	ingestor := &stubNativeIngestor{ensureKBID: "kb-temp", aliveDocs: []string{"k-history", "k-other-session"}}
	messages := &stubMessageService{messages: []*types.Message{{Attachments: types.MessageAttachments{
		{IndexedKnowledgeID: "k-history"},
		{IndexedKnowledgeID: "k-deleted"},
	}}}}
	h := &Handler{nativeAttachments: ingestor, messageService: messages}
	ctx := types.WithPlatformAgentScope(context.Background(), scope)

	h.routeOversizedAttachments(ctx, rc, scope, 10000, "conv-1")

	for _, id := range []string{"k-对账单.xlsx", "k-history"} {
		if err := types.AuthorizePlatformAgentDocument(ctx, "kb-temp", id); err != nil {
			t.Fatalf("current session document %q was rejected: %v", id, err)
		}
	}
	for _, id := range []string{"k-other-session", "k-deleted"} {
		if err := types.AuthorizePlatformAgentDocument(ctx, "kb-temp", id); err == nil {
			t.Fatalf("out-of-scope document %q was granted", id)
		}
		if containsIDs(rc.knowledgeIDs, id) {
			t.Fatalf("out-of-scope document %q reached retrieval targets", id)
		}
	}
}

func oversizedRequestContext() *qaRequestContext {
	return &qaRequestContext{
		knowledgeBaseIDs: []string{"kb-1"},
		knowledgeIDs:     []string{"doc-existing"},
		attachments: types.MessageAttachments{
			{FileName: "对账单.xlsx", Content: "first 500 lines", FullContent: "all 1629 lines", IsTruncated: true, LineCount: 1629},
			{FileName: "small.txt", Content: "hello"},
		},
	}
}

func TestOversizedPlatformAttachmentsRouteToTemporaryRetrieval(t *testing.T) {
	rc := oversizedRequestContext()
	scope := &types.PlatformAgentScope{KnowledgeBases: map[string][]string{"kb-1": nil}}
	ingestor := &stubNativeIngestor{kbID: "kb-temp", chunkCount: 42}
	h := &Handler{nativeAttachments: ingestor}

	h.routeOversizedAttachments(context.Background(), rc, scope, 10000, "conv-1")

	if len(ingestor.calls) != 1 || ingestor.calls[0] != "对账单.xlsx|conv-1|all 1629 lines" {
		t.Fatalf("ingest calls: %v", ingestor.calls)
	}
	big := rc.attachments[0]
	if big.IsTruncated || big.Content == "first 500 lines" || big.FullContent != "" {
		t.Fatalf("oversized attachment not rewritten to retrieval stub: %+v", big)
	}
	if !contains(big.Content, "k-对账单.xlsx") || !contains(big.Content, "grep_chunks") {
		t.Fatalf("stub does not point at the indexed document: %q", big.Content)
	}
	if big.TotalChunks != 42 || big.LineCount != 1629 {
		t.Fatalf("stub metadata lost: %+v", big)
	}
	if rc.attachments[1].Content != "hello" || rc.attachments[1].TotalChunks != 0 {
		t.Fatalf("small attachment was rewritten: %+v", rc.attachments[1])
	}
	if !containsIDs(rc.knowledgeIDs, "k-对账单.xlsx") || !containsIDs(rc.knowledgeIDs, "doc-existing") {
		t.Fatalf("knowledge ids not extended: %v", rc.knowledgeIDs)
	}
	if !containsIDs(rc.knowledgeBaseIDs, "kb-temp") || !containsIDs(rc.knowledgeBaseIDs, "kb-1") {
		t.Fatalf("knowledge base ids not extended: %v", rc.knowledgeBaseIDs)
	}
	if owned := scope.OwnedKnowledgeBases["kb-temp"]; len(owned) != 1 || owned[0] != "k-对账单.xlsx" {
		t.Fatalf("owned scope entry: %v", owned)
	}
}

func TestOversizedAttachmentsFallBackToTruncatedInlineWhenIngestFails(t *testing.T) {
	rc := oversizedRequestContext()
	scope := &types.PlatformAgentScope{KnowledgeBases: map[string][]string{"kb-1": nil}}
	ingestor := &stubNativeIngestor{kbID: "kb-temp", ingestErr: errors.New("embedding down")}
	h := &Handler{nativeAttachments: ingestor}

	h.routeOversizedAttachments(context.Background(), rc, scope, 10000, "conv-1")

	if rc.attachments[0].Content != "first 500 lines" || !rc.attachments[0].IsTruncated {
		t.Fatalf("failed ingest must keep the truncated inline fallback: %+v", rc.attachments[0])
	}
	if len(scope.OwnedKnowledgeBases) != 0 || len(rc.knowledgeIDs) != 1 {
		t.Fatalf("scope leaked after ingest failure: %v %v", scope.OwnedKnowledgeBases, rc.knowledgeIDs)
	}
}

func TestRoutingSkipsWithoutPlatformScope(t *testing.T) {
	rc := oversizedRequestContext()
	ingestor := &stubNativeIngestor{kbID: "kb-temp"}
	h := &Handler{nativeAttachments: ingestor}

	h.routeOversizedAttachments(context.Background(), rc, nil, 10000, "conv-1")

	if ingestor.ensureCalls != 0 || ingestor.findCalls != 0 || len(ingestor.calls) != 0 || rc.attachments[0].Content != "first 500 lines" {
		t.Fatal("non-platform requests must keep the legacy inline path")
	}
}

func contains(haystack, needle string) bool { return strings.Contains(haystack, needle) }

func containsIDs(ids []string, want string) bool { return slices.Contains(ids, want) }
