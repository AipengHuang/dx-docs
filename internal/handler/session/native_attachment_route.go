package session

import (
	"context"
	"fmt"

	"github.com/Tencent/WeKnora/internal/logger"
	"github.com/Tencent/WeKnora/internal/types"
)

// routeOversizedAttachments replaces truncated inline attachments with
// engine-ingested temporary documents scoped to this single run, so the agent
// retrieves the full text with grep_chunks/list_knowledge_chunks instead of
// reasoning over a prompt-inlined 500-line cut. Failures fall back to the
// legacy truncated inline content.
func (h *Handler) routeOversizedAttachments(
	ctx context.Context,
	rc *qaRequestContext,
	scope *types.PlatformAgentScope,
	tenantID uint64,
	conversationID string,
) {
	if scope == nil || h.nativeAttachments == nil {
		return
	}
	kbID, err := h.nativeAttachments.EphemeralKB(ctx, tenantID)
	if err != nil {
		logger.Warnf(ctx, "native attachment ephemeral KB unavailable, keeping inline path: %v", err)
		return
	}
	owned := make([]string, 0, len(rc.attachments))
	for i := range rc.attachments {
		attachment := &rc.attachments[i]
		if !attachment.IsTruncated || attachment.FullContent == "" {
			continue
		}
		result, err := h.nativeAttachments.Ingest(ctx, tenantID, kbID, attachment.FullContent, attachment.FileName, conversationID)
		if err != nil {
			logger.Warnf(ctx, "native attachment ingest failed, keeping truncated inline fallback: file=%s err=%v", attachment.FileName, err)
			continue
		}
		stub := types.MessageAttachment{
			URL:                attachment.URL,
			FileName:           attachment.FileName,
			FileType:           attachment.FileType,
			FileSize:           attachment.FileSize,
			Content:            nativeAttachmentIndexedStub(attachment.FileName, attachment.LineCount, result.KnowledgeID),
			LineCount:          attachment.LineCount,
			ContentMode:        "indexed_document",
			TotalChunks:        result.ChunkCount,
			IndexedKnowledgeID: result.KnowledgeID,
		}
		rc.attachments[i] = stub
		owned = append(owned, result.KnowledgeID)
		rc.knowledgeIDs = append(rc.knowledgeIDs, result.KnowledgeID)
		logger.Infof(ctx, "oversized attachment ingested for retrieval: file=%s lines=%d chunks=%d knowledge=%s",
			attachment.FileName, attachment.LineCount, result.ChunkCount, result.KnowledgeID)
	}
	// Follow-up turns in the same conversation must keep retrieval access to
	// documents ingested by earlier turns (within their TTL), or the model is
	// forced to ask the user to re-upload the file for every question.
	h.reattachSessionTemporaryDocuments(ctx, rc, scope, tenantID, conversationID, kbID, owned)
}

func (h *Handler) reattachSessionTemporaryDocuments(
	ctx context.Context,
	rc *qaRequestContext,
	scope *types.PlatformAgentScope,
	tenantID uint64,
	sessionID, kbID string,
	ingested []string,
) {
	var messages []*types.Message
	if h.messageService != nil {
		var err error
		messages, err = h.messageService.GetRecentMessagesBySession(ctx, sessionID, 100)
		if err != nil {
			logger.Warnf(ctx, "native attachment history lookup failed: %v", err)
		}
	}
	seen := make(map[string]bool, len(ingested))
	for _, id := range ingested {
		seen[id] = true
	}
	for _, message := range messages {
		for _, attachment := range message.Attachments {
			if attachment.IndexedKnowledgeID != "" && !seen[attachment.IndexedKnowledgeID] {
				seen[attachment.IndexedKnowledgeID] = true
				ingested = append(ingested, attachment.IndexedKnowledgeID)
			}
		}
	}
	if len(ingested) == 0 {
		return
	}
	// Drop ids already reclaimed by the TTL sweeper so the prompt never pins
	// dead documents.
	if docs, err := h.nativeAttachments.ListTemporaryDocuments(ctx, tenantID, kbID); err == nil {
		alive := make(map[string]bool, len(docs))
		for _, doc := range docs {
			alive[doc.ID] = true
		}
		filtered := ingested[:0]
		for _, id := range ingested {
			if alive[id] {
				filtered = append(filtered, id)
			}
		}
		ingested = filtered
	}
	if len(ingested) == 0 {
		return
	}
	if scope.OwnedKnowledgeBases == nil {
		scope.OwnedKnowledgeBases = make(map[string][]string)
	}
	existing := scope.OwnedKnowledgeBases[kbID]
	known := make(map[string]bool, len(existing)+len(ingested))
	for _, id := range existing {
		known[id] = true
	}
	merged := existing
	for _, id := range ingested {
		if !known[id] {
			known[id] = true
			merged = append(merged, id)
		}
	}
	scope.OwnedKnowledgeBases[kbID] = merged
	for _, id := range ingested {
		rc.knowledgeIDs = append(rc.knowledgeIDs, id)
	}
	if len(scope.OwnedKnowledgeBases[kbID]) > 0 {
		rc.knowledgeBaseIDs = append(rc.knowledgeBaseIDs, kbID)
	}
	logger.Infof(ctx, "session temporary documents re-scoped: session=%s count=%d", sessionID, len(scope.OwnedKnowledgeBases[kbID]))
}

func nativeAttachmentIndexedStub(fileName string, lineCount int, knowledgeID string) string {
	return fmt.Sprintf("【%s 共 %d 行，超出对话内联上限，完整内容已索引为临时文档 %s。"+
		"禁止声称无法读取该文件；请用 list_knowledge_chunks(knowledge_id=\"%s\") 分页读取全文，"+
		"或用 grep_chunks 检索关键词，再基于全文执行任务。】", fileName, lineCount, knowledgeID, knowledgeID)
}
