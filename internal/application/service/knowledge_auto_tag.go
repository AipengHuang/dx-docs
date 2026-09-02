package service

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/Tencent/WeKnora/internal/logger"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"github.com/hibiken/asynq"
)

const (
	// maximumAutoTagCandidates bounds the candidate list sent to the model.
	// Candidates are addressed by ordinal rather than UUID, so each one costs
	// roughly a tenth of what an id-based listing would and this ceiling stays
	// well inside a normal context window.
	maximumAutoTagCandidates   = 500
	maximumAutoTagContentRunes = 16000
	minimumAutoTagConfidence   = 0.75
	autoTagSpanName            = "postprocess.auto_tag"
)

// normalizeAutoTagMaxTags reuses the single source of truth for the max_tags
// bounds so the prompt, the match filter and the persisted config cannot drift.
func normalizeAutoTagMaxTags(maxTags int) int {
	config := types.AutoTagConfig{MaxTags: maxTags}
	config.Normalize()
	return config.MaxTags
}

// KnowledgeAutoTagService classifies processed documents against existing knowledge-base tags.
type KnowledgeAutoTagService struct {
	knowledgeRepo interfaces.KnowledgeRepository
	tagRepo       interfaces.KnowledgeTagRepository
	kbService     interfaces.KnowledgeBaseService
	chunkService  interfaces.ChunkService
	modelService  interfaces.ModelService
	spanTracker   SpanTracker
}

// NewKnowledgeAutoTagService creates the asynchronous automatic-tagging task handler.
func NewKnowledgeAutoTagService(
	knowledgeRepo interfaces.KnowledgeRepository,
	tagRepo interfaces.KnowledgeTagRepository,
	kbService interfaces.KnowledgeBaseService,
	chunkService interfaces.ChunkService,
	modelService interfaces.ModelService,
	spanTracker SpanTracker,
) interfaces.TaskHandler {
	return &KnowledgeAutoTagService{
		knowledgeRepo: knowledgeRepo,
		tagRepo:       tagRepo,
		kbService:     kbService,
		chunkService:  chunkService,
		modelService:  modelService,
		spanTracker:   spanTracker,
	}
}

func (s *KnowledgeAutoTagService) tracker() SpanTracker {
	if s.spanTracker == nil {
		return noopSpanTracker{}
	}
	return s.spanTracker
}

// Handle processes one automatic-tagging task.
func (s *KnowledgeAutoTagService) Handle(ctx context.Context, task *asynq.Task) error {
	var payload types.KnowledgeAutoTagPayload
	if err := json.Unmarshal(task.Payload(), &payload); err != nil {
		return fmt.Errorf("unmarshal knowledge auto tag payload: %w", err)
	}
	ctx = context.WithValue(ctx, types.TenantIDContextKey, payload.TenantID)
	if payload.Language != "" {
		ctx = context.WithValue(ctx, types.LanguageContextKey, payload.Language)
	}

	parent := s.tracker().LookupStage(ctx, payload.KnowledgeID, payload.Attempt, types.StagePostProcess)
	span := s.tracker().BeginSubSpan(ctx, parent, autoTagSpanName, types.SpanKindSubSpan, types.JSONMap{
		"knowledge_base_id": payload.KnowledgeBaseID,
	})
	skip := func(reason string, extra types.JSONMap) error {
		if extra == nil {
			extra = types.JSONMap{}
		}
		extra["skipped"] = reason
		s.tracker().EndSpan(ctx, span, extra)
		return nil
	}

	knowledge, err := s.knowledgeRepo.GetKnowledgeByIDOnly(ctx, payload.KnowledgeID)
	if err != nil {
		s.tracker().FailSpan(ctx, span, "AUTO_TAG_KNOWLEDGE_LOAD_FAILED", err.Error(), err)
		return fmt.Errorf("get knowledge for auto tag: %w", err)
	}
	if knowledge == nil || knowledge.TenantID != payload.TenantID || knowledge.KnowledgeBaseID != payload.KnowledgeBaseID {
		return skip("knowledge_not_in_scope", nil)
	}
	if knowledge.ParseStatus == types.ParseStatusCancelled ||
		knowledge.ParseStatus == types.ParseStatusDeleting ||
		knowledge.ParseStatus == types.ParseStatusFailed {
		return skip("knowledge_not_active", types.JSONMap{"parse_status": knowledge.ParseStatus})
	}

	kb, err := s.kbService.GetKnowledgeBaseByIDOnly(ctx, payload.KnowledgeBaseID)
	if err != nil {
		s.tracker().FailSpan(ctx, span, "AUTO_TAG_KB_LOAD_FAILED", err.Error(), err)
		return fmt.Errorf("get knowledge base for auto tag: %w", err)
	}
	if kb == nil || kb.TenantID != payload.TenantID || kb.Type != types.KnowledgeBaseTypeDocument {
		return skip("knowledge_base_not_eligible", nil)
	}
	config := kb.AutoTagConfig
	if config == nil || !config.Enabled {
		return skip("auto_tag_disabled", nil)
	}
	config.Normalize()

	tags, totalTags, err := s.tagRepo.ListByKB(
		ctx,
		payload.TenantID,
		payload.KnowledgeBaseID,
		&types.Pagination{Page: 1, PageSize: maximumAutoTagCandidates},
		"",
	)
	if err != nil {
		s.tracker().FailSpan(ctx, span, "AUTO_TAG_LIST_TAGS_FAILED", err.Error(), err)
		return fmt.Errorf("list auto tag candidates: %w", err)
	}
	if len(tags) == 0 {
		return skip("no_candidate_tags", nil)
	}
	// ListByKB orders by (sort_order, created_at, seq_id), so an oversized tag
	// set yields a stable prefix. Classifying that prefix keeps the feature
	// working on large knowledge bases instead of disabling it outright, which
	// is what dropping the task did before.
	candidatesTruncated := totalTags > int64(len(tags))
	if candidatesTruncated {
		logger.Warnf(ctx,
			"[KnowledgeAutoTag] Knowledge base %s has %d tags; classifying the first %d only",
			payload.KnowledgeBaseID, totalTags, len(tags),
		)
	}

	// Existing tags are loaded before the model call so a document that is
	// already classified — manually or by an earlier run — costs no LLM
	// round-trip at all when skip_if_tagged is on.
	existing, err := s.knowledgeRepo.GetKnowledgeTags(ctx, []string{payload.KnowledgeID})
	if err != nil {
		s.tracker().FailSpan(ctx, span, "AUTO_TAG_LOAD_EXISTING_FAILED", err.Error(), err)
		return fmt.Errorf("get existing knowledge tags: %w", err)
	}
	existingTags := existing[payload.KnowledgeID]
	if len(existingTags) > 0 && config.ShouldSkipIfTagged() {
		return skip("knowledge_already_tagged", types.JSONMap{
			"existing_tag_count": len(existingTags),
		})
	}
	existingIDs := make(map[string]struct{}, len(existingTags))
	for _, tag := range existingTags {
		existingIDs[tag.ID] = struct{}{}
	}

	modelID := strings.TrimSpace(config.ModelID)
	if modelID == "" {
		modelID = strings.TrimSpace(kb.SummaryModelID)
	}
	if modelID == "" {
		return skip("model_not_configured", nil)
	}

	chunks, err := s.chunkService.ListChunksByKnowledgeID(ctx, payload.KnowledgeID)
	if err != nil {
		s.tracker().FailSpan(ctx, span, "AUTO_TAG_LIST_CHUNKS_FAILED", err.Error(), err)
		return fmt.Errorf("list chunks for auto tag: %w", err)
	}
	content := buildAutoTagDocumentContent(knowledge, chunks)
	if config.MaxTags == 1 {
		content = buildCategoryDocumentContent(chunks)
	}
	if strings.TrimSpace(content) == "" {
		if config.MaxTags == 1 {
			decision := documentClassificationResult{
				Status: classificationStatusPending, PendingReason: classificationReasonUnsupported,
				Candidates: []documentClassificationCandidate{}, ModelID: modelID,
				RuleVersion: classificationRuleVersion(tags),
			}
			if err := s.persistCategoryDecision(ctx, knowledge, decision); err != nil {
				return err
			}
		}
		return skip("no_text_chunks", nil)
	}

	chatModel, err := s.modelService.GetChatModel(ctx, modelID)
	if err != nil {
		s.tracker().FailSpan(ctx, span, "AUTO_TAG_MODEL_LOAD_FAILED", err.Error(), err)
		return fmt.Errorf("get auto tag model: %w", err)
	}
	response, err := classifyExistingTags(ctx, chatModel, tags, content, config)
	if err != nil {
		s.tracker().FailSpan(ctx, span, "AUTO_TAG_MODEL_CALL_FAILED", err.Error(), err)
		return err
	}

	validIDs := validateAutoTagMatches(tags, response.Matches, config.MaxTags)
	var categoryDecision *documentClassificationResult
	if config.MaxTags == 1 {
		decision := buildCategoryDecision(tags, response.Matches, modelID, classificationRuleVersion(tags))
		categoryDecision = &decision
		staged := decision
		if staged.Status == classificationStatusConfirmed {
			staged.Status = classificationStatusPending
			staged.PendingReason = classificationReasonAttachmentPending
		}
		if err := s.persistCategoryDecision(ctx, knowledge, staged); err != nil {
			return err
		}
		validIDs = decision.SelectedTagIDs()
	}
	if len(validIDs) == 0 {
		return skip("no_matching_tags", types.JSONMap{
			"candidate_tag_count":   len(tags),
			"candidate_tags_capped": candidatesTruncated,
			"model_id":              modelID,
		})
	}

	added := make([]string, 0, len(validIDs))
	for _, id := range validIDs {
		if _, ok := existingIDs[id]; !ok {
			added = append(added, id)
		}
	}
	if len(added) == 0 {
		if categoryDecision != nil {
			if err := s.persistCategoryDecision(ctx, knowledge, *categoryDecision); err != nil {
				return err
			}
		}
		return skip("all_matches_already_attached", types.JSONMap{"matched_tag_count": len(validIDs)})
	}
	if err := s.knowledgeRepo.AddKnowledgeTagRelations(
		ctx,
		payload.TenantID,
		payload.KnowledgeBaseID,
		payload.KnowledgeID,
		added,
	); err != nil {
		s.tracker().FailSpan(ctx, span, "AUTO_TAG_PERSIST_FAILED", err.Error(), err)
		return fmt.Errorf("add automatic knowledge tags: %w", err)
	}
	if categoryDecision != nil {
		if err := s.persistCategoryDecision(ctx, knowledge, *categoryDecision); err != nil {
			return err
		}
	}

	logger.Infof(ctx, "[KnowledgeAutoTag] Added %d tag(s) to knowledge %s", len(added), payload.KnowledgeID)
	s.tracker().EndSpan(ctx, span, types.JSONMap{
		"candidate_tag_count":   len(tags),
		"candidate_tags_capped": candidatesTruncated,
		"matched_tag_count":     len(validIDs),
		"added_tag_count":       len(added),
		"added_tag_ids":         added,
		"model_id":              modelID,
	})
	return nil
}

func buildAutoTagDocumentContent(knowledge *types.Knowledge, chunks []*types.Chunk) string {
	parts := make([]string, 0, len(chunks)+2)
	if name := strings.TrimSpace(knowledge.FileName); name != "" {
		parts = append(parts, "Document name: "+name)
	}
	if summary := strings.TrimSpace(knowledge.Description); summary != "" {
		parts = append(parts, "Existing summary: "+summary)
	}
	orderedChunks := make([]*types.Chunk, 0, len(chunks))
	for _, chunk := range chunks {
		if chunk != nil {
			orderedChunks = append(orderedChunks, chunk)
		}
	}
	sort.SliceStable(orderedChunks, func(i, j int) bool { return orderedChunks[i].StartAt < orderedChunks[j].StartAt })
	for _, chunk := range orderedChunks {
		if chunk.ChunkType != types.ChunkTypeText &&
			chunk.ChunkType != types.ChunkTypeImageOCR &&
			chunk.ChunkType != types.ChunkTypeImageCaption {
			continue
		}
		if text := strings.TrimSpace(chunk.Content); text != "" {
			parts = append(parts, text)
		}
	}
	return sampleLongContent(strings.Join(parts, "\n\n"), maximumAutoTagContentRunes)
}

func buildCategoryDocumentContent(chunks []*types.Chunk) string {
	return buildAutoTagDocumentContent(&types.Knowledge{}, chunks)
}
