// Package service содержит бизнес-логику - валидация, оркестрация репозитория, логирование.
package service

import (
	"context"
	"fmt"
	"log/slog"

	"go-chi-pgx-api/internal/domain"
	"go-chi-pgx-api/internal/repository"
)

type ItemService struct {
	repo   repository.ItemRepository
	logger *slog.Logger
}

func NewItemService(repo repository.ItemRepository, logger *slog.Logger) *ItemService {
	return &ItemService{repo: repo, logger: logger}
}

func (s *ItemService) GetByID(ctx context.Context, id int64) (domain.Item, error) {
	item, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return domain.Item{}, fmt.Errorf("service get item: %w", err)
	}
	return item, nil
}

func (s *ItemService) List(ctx context.Context, filter domain.ItemFilter) ([]domain.Item, error) {
	items, err := s.repo.List(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("service list items: %w", err)
	}
	return items, nil
}

func (s *ItemService) Create(ctx context.Context, input domain.CreateItemInput) (domain.Item, error) {
	if err := input.Validate(); err != nil {
		return domain.Item{}, err
	}

	item, err := s.repo.Create(ctx, input)
	if err != nil {
		return domain.Item{}, fmt.Errorf("service create item: %w", err)
	}

	s.logger.Info("item created",
		slog.Int64("id", item.ID),
		slog.String("title", item.Title),
	)
	return item, nil
}

func (s *ItemService) Update(ctx context.Context, id int64, input domain.UpdateItemInput) (domain.Item, error) {
	if err := input.Validate(); err != nil {
		return domain.Item{}, err
	}

	item, err := s.repo.Update(ctx, id, input)
	if err != nil {
		return domain.Item{}, fmt.Errorf("service update item: %w", err)
	}

	s.logger.Info("item updated", slog.Int64("id", item.ID))
	return item, nil
}

func (s *ItemService) Delete(ctx context.Context, id int64) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		return fmt.Errorf("service delete item: %w", err)
	}
	s.logger.Info("item deleted", slog.Int64("id", id))
	return nil
}
