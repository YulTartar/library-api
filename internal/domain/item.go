// Package domain содержит бизнес-модели и доменные ошибки.
// Никаких зависимостей от HTTP, БД или внешних библиотек.
package domain

import (
	"errors"
	"time"
)

type Item struct {
	ID          int64     `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description,omitempty"`
	Price       float64   `json:"price"`
	Active      bool      `json:"active"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type CreateItemInput struct {
	Title       string  `json:"title"`
	Description string  `json:"description,omitempty"`
	Price       float64 `json:"price"`
}

func (c CreateItemInput) Validate() error {
	if c.Title == "" {
		return ErrTitleRequired
	}
	if c.Price < 0 {
		return ErrNegativePrice
	}
	return nil
}

// UpdateItemInput использует указатели, чтобы отличить непереданное поле (nil) от явно заданного пустого значения.
type UpdateItemInput struct {
	Title       *string  `json:"title,omitempty"`
	Description *string  `json:"description,omitempty"`
	Price       *float64 `json:"price,omitempty"`
	Active      *bool    `json:"active,omitempty"`
}

func (u UpdateItemInput) Validate() error {
	if u.Price != nil && *u.Price < 0 {
		return ErrNegativePrice
	}
	return nil
}

// ItemFilter задаёт фильтрацию и пагинацию для списка товаров.
type ItemFilter struct {
	ActiveOnly bool
	Search     string
	Limit      int
	Offset     int
}

// Доменные ошибки. Handler'ы мапят их на HTTP-коды.
var (
	ErrNotFound       = errors.New("item not found")
	ErrDuplicateTitle = errors.New("item with this title already exists")
	ErrTitleRequired  = errors.New("title is required")
	ErrNegativePrice  = errors.New("price must be non-negative")
)
