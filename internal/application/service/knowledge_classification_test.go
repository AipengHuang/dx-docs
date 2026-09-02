package service

import (
	"encoding/json"
	"testing"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildCategoryDecisionConfirmsOneSupportedCandidate(t *testing.T) {
	tags := []*types.KnowledgeTag{{ID: "tag-a", Name: "Specification"}}
	decision := buildCategoryDecision(tags, []autoTagModelMatch{{
		Index: 1, Confidence: confidencePtr(0.91), Evidence: "The document defines product requirements.",
	}}, "deepseek", "baseline-3")

	assert.Equal(t, classificationStatusConfirmed, decision.Status)
	assert.Equal(t, []string{"tag-a"}, decision.SelectedTagIDs())
	require.Len(t, decision.Candidates, 1)
	assert.Equal(t, "baseline-3", decision.RuleVersion)
}

func TestBuildCategoryDecisionLeavesUncertainResultsPending(t *testing.T) {
	tags := []*types.KnowledgeTag{{ID: "tag-a", Name: "Specification"}, {ID: "tag-b", Name: "Report"}}
	tests := []struct {
		name    string
		matches []autoTagModelMatch
		reason  classificationPendingReason
	}{
		{name: "multiple", matches: []autoTagModelMatch{
			{Index: 1, Confidence: confidencePtr(0.9), Evidence: "requirements"},
			{Index: 2, Confidence: confidencePtr(0.8), Evidence: "results"},
		}, reason: classificationReasonMultipleCandidates},
		{name: "low confidence", matches: []autoTagModelMatch{
			{Index: 1, Confidence: confidencePtr(0.4), Evidence: "requirements"},
		}, reason: classificationReasonLowConfidence},
		{name: "missing evidence", matches: []autoTagModelMatch{
			{Index: 1, Confidence: confidencePtr(0.9)},
		}, reason: classificationReasonMissingEvidence},
		{name: "missing confidence", matches: []autoTagModelMatch{
			{Index: 1, Evidence: "requirements"},
		}, reason: classificationReasonMissingConfidence},
		{name: "unsupported", matches: []autoTagModelMatch{
			{Index: 99, Confidence: confidencePtr(0.9), Evidence: "unknown"},
		}, reason: classificationReasonUnsupported},
		{name: "partly unsupported", matches: []autoTagModelMatch{
			{Index: 1, Confidence: confidencePtr(0.9), Evidence: "requirements"},
			{Index: 99, Confidence: confidencePtr(0.9), Evidence: "unknown"},
		}, reason: classificationReasonUnsupported},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			decision := buildCategoryDecision(tags, test.matches, "deepseek", "baseline-3")
			assert.Equal(t, classificationStatusPending, decision.Status)
			assert.Equal(t, test.reason, decision.PendingReason)
			assert.Empty(t, decision.SelectedTagIDs())
		})
	}
}

func TestSetClassificationMetadataPreservesExistingMetadata(t *testing.T) {
	metadata, err := setClassificationMetadata(types.JSON(`{"pages":"2"}`), documentClassificationResult{
		Status: classificationStatusPending, ModelID: "deepseek", RuleVersion: "baseline-3",
	})
	require.NoError(t, err)
	var value map[string]any
	require.NoError(t, json.Unmarshal(metadata, &value))
	assert.Equal(t, "2", value["pages"])
	assert.NotNil(t, value[classificationMetadataKey])
}

func TestCategoryModePersistsDecisionAndOnlyAttachesConfirmedTag(t *testing.T) {
	confirmed := newAutoTagFixture(t, `{"matches":[{"index":1,"confidence":0.91,"evidence":"requirements"}]}`)
	confirmed.service.kbService.(*autoTagKBService).kb.AutoTagConfig.MaxTags = 1
	require.NoError(t, confirmed.handle(t))
	assert.Equal(t, []string{"tag-a"}, confirmed.repo.added)
	assert.Contains(t, string(confirmed.repo.metadata), `"status":"confirmed"`)

	pending := newAutoTagFixture(t, `{"matches":[{"index":1,"confidence":0.91}]}`)
	pending.service.kbService.(*autoTagKBService).kb.AutoTagConfig.MaxTags = 1
	require.NoError(t, pending.handle(t))
	assert.Empty(t, pending.repo.added)
	assert.Contains(t, string(pending.repo.metadata), `"status":"pending"`)
}

func TestCategoryModeNeverClassifiesFromFileNameAlone(t *testing.T) {
	fixture := newAutoTagFixture(t, `{"matches":[{"index":1,"confidence":0.99,"evidence":"policy.pdf"}]}`)
	fixture.service.kbService.(*autoTagKBService).kb.AutoTagConfig.MaxTags = 1
	fixture.service.chunkService = &autoTagChunkService{}

	require.NoError(t, fixture.handle(t))
	assert.Empty(t, fixture.chatModel.messages)
	assert.Empty(t, fixture.repo.added)
	assert.Contains(t, string(fixture.repo.metadata), `"status":"pending"`)
	assert.Contains(t, string(fixture.repo.metadata), `"pending_reason":"unsupported"`)
}

func TestCategoryModePromptUsesParsedContentWithoutFileName(t *testing.T) {
	fixture := newAutoTagFixture(t, `{"matches":[{"index":1,"confidence":0.91,"evidence":"quarterly HR policy"}]}`)
	fixture.service.kbService.(*autoTagKBService).kb.AutoTagConfig.MaxTags = 1

	require.NoError(t, fixture.handle(t))
	require.Len(t, fixture.chatModel.messages, 2)
	assert.Contains(t, fixture.chatModel.messages[1].Content, "quarterly HR policy")
	assert.NotContains(t, fixture.chatModel.messages[1].Content, "policy.pdf")
}
