package service

import (
	"context"
	"errors"
	"strings"
	"time"

	werrors "github.com/Tencent/WeKnora/internal/errors"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// CreatePlatformTag 使用 Platform 提供的稳定 UUID 创建项目分类标签。
func (s *knowledgeTagService) CreatePlatformTag(
	ctx context.Context,
	kbID string,
	id string,
	name string,
	sortOrder int,
) (*types.KnowledgeTag, error) {
	name = strings.TrimSpace(name)
	if kbID == "" || name == "" {
		return nil, werrors.NewBadRequestError("Knowledge base and category name are required")
	}
	if _, err := uuid.Parse(id); err != nil {
		return nil, werrors.NewBadRequestError("Category tag id must be a UUID")
	}
	kb, err := s.kbService.GetKnowledgeBaseByID(ctx, kbID)
	if err != nil {
		return nil, err
	}
	existing, err := s.repo.GetByID(ctx, kb.TenantID, id)
	if err == nil && existing != nil {
		if existing.KnowledgeBaseID == kb.ID && existing.Name == name {
			return existing, nil
		}
		return nil, werrors.NewConflictError("Category tag id is already in use")
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	byName, err := s.repo.GetByName(ctx, kb.TenantID, kb.ID, name)
	if err == nil && byName != nil {
		return nil, werrors.NewConflictError("Category name is already in use")
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	now := time.Now()
	tag := &types.KnowledgeTag{
		ID: id, TenantID: kb.TenantID, KnowledgeBaseID: kb.ID,
		Name: name, SortOrder: sortOrder, CreatedAt: now, UpdatedAt: now,
	}
	if err := s.repo.Create(ctx, tag); err != nil {
		return nil, err
	}
	return tag, nil
}
