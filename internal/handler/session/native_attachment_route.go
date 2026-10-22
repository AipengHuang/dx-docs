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
			URL:         attachment.URL,
			FileName:    attachment.FileName,
			FileType:    attachment.FileType,
			FileSize:    attachment.FileSize,
			Content:     nativeAttachmentIndexedStub(attachment.FileName, attachment.LineCount, result.KnowledgeID),
			LineCount:   attachment.LineCount,
			ContentMode: "indexed_document",
			TotalChunks: result.ChunkCount,
		}
		rc.attachments[i] = stub
		owned = append(owned, result.KnowledgeID)
		rc.knowledgeIDs = append(rc.knowledgeIDs, result.KnowledgeID)
		logger.Infof(ctx, "oversized attachment ingested for retrieval: file=%s lines=%d chunks=%d knowledge=%s",
			attachment.FileName, attachment.LineCount, result.ChunkCount, result.KnowledgeID)
	}
	if len(owned) == 0 {
		return
	}
	rc.knowledgeBaseIDs = append(rc.knowledgeBaseIDs, kbID)
	if scope.OwnedKnowledgeBases == nil {
		scope.OwnedKnowledgeBases = make(map[string][]string)
	}
	scope.OwnedKnowledgeBases[kbID] = owned
}

func nativeAttachmentIndexedStub(fileName string, lineCount int, knowledgeID string) string {
	return fmt.Sprintf("【%s 共 %d 行，超出对话内联上限，完整内容已索引为临时文档 %s。"+
		"禁止声称无法读取该文件；请用 list_knowledge_chunks(knowledge_id=\"%s\") 分页读取全文，"+
		"或用 grep_chunks 检索关键词，再基于全文执行任务。】", fileName, lineCount, knowledgeID, knowledgeID)
}
