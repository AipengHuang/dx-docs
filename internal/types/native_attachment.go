package types

import "context"

// NativeAttachmentIngestResult reports what an oversized chat attachment became.
type NativeAttachmentIngestResult struct {
	KnowledgeID string
	ChunkCount  int
}

// NativeAttachmentIngestor turns oversized platform-run attachments into
// retrievable temporary documents instead of prompt-inlined truncated text.
type NativeAttachmentIngestor interface {
	EphemeralKB(ctx context.Context, tenantID uint64) (string, error)
	Ingest(ctx context.Context, tenantID uint64, kbID string, content, fileName, conversationID string) (NativeAttachmentIngestResult, error)
	CleanupExpired(ctx context.Context, tenantID uint64)
}
