// Package repository определяет интерфейс доступа к данным.
// Service-слой зависит от интерфейса, а не от конкретной реализации (PostgreSQL).
package repository

import (
	"context"

	"go-chi-pgx-api/internal/domain"
)

type ItemRepository interface {
	GetByID(ctx context.Context, id int64) (domain.Item, error)
	List(ctx context.Context, filter domain.ItemFilter) ([]domain.Item, error)
	Create(ctx context.Context, input domain.CreateItemInput) (domain.Item, error)
	Update(ctx context.Context, id int64, input domain.UpdateItemInput) (domain.Item, error)
	Delete(ctx context.Context, id int64) error
}
