package session

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/Tencent/WeKnora/internal/types"
)

type stubNativeIngestor struct {
	kbID       string
	ingestErr  error
	calls      []string
	chunkCount int
}

func (s *stubNativeIngestor) EphemeralKB(context.Context, uint64) (string, error) {
	return s.kbID, nil
}

func (s *stubNativeIngestor) Ingest(_ context.Context, _ uint64, _ string, content, fileName, conversationID string) (types.NativeAttachmentIngestResult, error) {
	s.calls = append(s.calls, fileName+"|"+conversationID+"|"+content)
	if s.ingestErr != nil {
		return types.NativeAttachmentIngestResult{}, s.ingestErr
	}
	return types.NativeAttachmentIngestResult{KnowledgeID: "k-" + fileName, ChunkCount: s.chunkCount}, nil
}

func (stubNativeIngestor) CleanupExpired(context.Context, uint64) {}

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

	if len(ingestor.calls) != 0 || rc.attachments[0].Content != "first 500 lines" {
		t.Fatal("non-platform requests must keep the legacy inline path")
	}
}

func contains(haystack, needle string) bool { return strings.Contains(haystack, needle) }

func containsIDs(ids []string, want string) bool { return slices.Contains(ids, want) }
