package service

import (
	"context"
	"testing"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

type platformTagKBService struct {
	interfaces.KnowledgeBaseService
}

func (platformTagKBService) GetKnowledgeBaseByID(
	context.Context, string,
) (*types.KnowledgeBase, error) {
	return &types.KnowledgeBase{ID: "kb", TenantID: 7}, nil
}

type platformTagRepo struct {
	interfaces.KnowledgeTagRepository
	tags map[string]*types.KnowledgeTag
}

func (r *platformTagRepo) GetByID(
	_ context.Context, _ uint64, id string,
) (*types.KnowledgeTag, error) {
	if tag := r.tags[id]; tag != nil {
		return tag, nil
	}
	return nil, gorm.ErrRecordNotFound
}

func (r *platformTagRepo) GetByName(
	_ context.Context, _ uint64, kbID string, name string,
) (*types.KnowledgeTag, error) {
	for _, tag := range r.tags {
		if tag.KnowledgeBaseID == kbID && tag.Name == name {
			return tag, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func (r *platformTagRepo) Create(_ context.Context, tag *types.KnowledgeTag) error {
	r.tags[tag.ID] = tag
	return nil
}

func TestCreatePlatformTagUsesStableSuppliedID(t *testing.T) {
	repo := &platformTagRepo{tags: map[string]*types.KnowledgeTag{}}
	service := &knowledgeTagService{kbService: platformTagKBService{}, repo: repo}
	id := "20b3507a-48bf-5a67-a3ea-e5749b49b899"

	first, err := service.CreatePlatformTag(context.Background(), "kb", id, "Specification", 1)
	require.NoError(t, err)
	second, err := service.CreatePlatformTag(context.Background(), "kb", id, "Specification", 1)
	require.NoError(t, err)

	assert.Equal(t, id, first.ID)
	assert.Same(t, first, second)
	assert.Len(t, repo.tags, 1)
}
