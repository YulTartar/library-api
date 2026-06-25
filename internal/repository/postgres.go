package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"go-chi-pgx-api/internal/domain"
)

// postgresItemRepo реализует ItemRepository для PostgreSQL через pgxpool.
// Пул потокобезопасен, один экземпляр на всё приложение.
type postgresItemRepo struct {
	pool *pgxpool.Pool
}

func NewPostgresItemRepo(pool *pgxpool.Pool) ItemRepository {
	return &postgresItemRepo{pool: pool}
}

func (r *postgresItemRepo) GetByID(ctx context.Context, id int64) (domain.Item, error) {
	var item domain.Item

	err := r.pool.QueryRow(ctx, `
		SELECT id, title, description, price, active, created_at, updated_at
		FROM items
		WHERE id = $1`, id,
	).Scan(
		&item.ID,
		&item.Title,
		&item.Description,
		&item.Price,
		&item.Active,
		&item.CreatedAt,
		&item.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Item{}, domain.ErrNotFound
		}
		return domain.Item{}, fmt.Errorf("query item by id: %w", err)
	}

	return item, nil
}

func (r *postgresItemRepo) List(ctx context.Context, filter domain.ItemFilter) ([]domain.Item, error) {
	limit := filter.Limit
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	offset := filter.Offset
	if offset < 0 {
		offset = 0
	}

	// Динамическая сборка WHERE из переданных фильтров
	query := `
		SELECT id, title, description, price, active, created_at, updated_at
		FROM items
		WHERE 1=1`
	args := []any{}
	argIdx := 1

	if filter.ActiveOnly {
		query += fmt.Sprintf(" AND active = $%d", argIdx)
		args = append(args, true)
		argIdx++
	}

	if filter.Search != "" {
		query += fmt.Sprintf(" AND title ILIKE $%d", argIdx)
		args = append(args, "%"+filter.Search+"%")
		argIdx++
	}

	query += " ORDER BY created_at DESC"
	query += fmt.Sprintf(" LIMIT $%d OFFSET $%d", argIdx, argIdx+1)
	args = append(args, limit, offset)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query items: %w", err)
	}
	defer rows.Close()

	items, err := pgx.CollectRows(rows, pgx.RowToStructByPos[domain.Item])
	if err != nil {
		return nil, fmt.Errorf("scan items: %w", err)
	}

	// Гарантируем [] вместо null в JSON
	if items == nil {
		items = []domain.Item{}
	}

	return items, nil
}

func (r *postgresItemRepo) Create(ctx context.Context, input domain.CreateItemInput) (domain.Item, error) {
	var item domain.Item

	// INSERT ... RETURNING даёт всю строку за один round-trip
	err := r.pool.QueryRow(ctx, `
		INSERT INTO items (title, description, price)
		VALUES ($1, $2, $3)
		RETURNING id, title, description, price, active, created_at, updated_at`,
		input.Title, input.Description, input.Price,
	).Scan(
		&item.ID,
		&item.Title,
		&item.Description,
		&item.Price,
		&item.Active,
		&item.CreatedAt,
		&item.UpdatedAt,
	)

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return domain.Item{}, domain.ErrDuplicateTitle
		}
		return domain.Item{}, fmt.Errorf("insert item: %w", err)
	}

	return item, nil
}

func (r *postgresItemRepo) Update(ctx context.Context, id int64, input domain.UpdateItemInput) (domain.Item, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return domain.Item{}, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	// Проверяем, что запись существует
	var exists bool
	err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM items WHERE id = $1)`, id).Scan(&exists)
	if err != nil {
		return domain.Item{}, fmt.Errorf("check item exists: %w", err)
	}
	if !exists {
		return domain.Item{}, domain.ErrNotFound
	}

	// COALESCE оставляет текущее значение, если передан nil
	var item domain.Item
	err = tx.QueryRow(ctx, `
		UPDATE items
		SET title       = COALESCE($1, title),
		    description = COALESCE($2, description),
		    price       = COALESCE($3, price),
		    active      = COALESCE($4, active),
		    updated_at  = NOW()
		WHERE id = $5
		RETURNING id, title, description, price, active, created_at, updated_at`,
		input.Title, input.Description, input.Price, input.Active, id,
	).Scan(
		&item.ID,
		&item.Title,
		&item.Description,
		&item.Price,
		&item.Active,
		&item.CreatedAt,
		&item.UpdatedAt,
	)

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return domain.Item{}, domain.ErrDuplicateTitle
		}
		return domain.Item{}, fmt.Errorf("update item: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return domain.Item{}, fmt.Errorf("commit tx: %w", err)
	}

	return item, nil
}

func (r *postgresItemRepo) Delete(ctx context.Context, id int64) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM items WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete item: %w", err)
	}

	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}

	return nil
}
