// Package handlers содержит HTTP-обработчики.
// Разбирают запрос, вызывают сервисный слой, мапят доменные ошибки на HTTP-статусы.
package handlers

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"go-chi-pgx-api/internal/domain"
	"go-chi-pgx-api/internal/service"
)

type ItemHandler struct {
	svc    *service.ItemService
	logger *slog.Logger
}

func NewItemHandler(svc *service.ItemService, logger *slog.Logger) *ItemHandler {
	return &ItemHandler{svc: svc, logger: logger}
}

func (h *ItemHandler) Routes() chi.Router {
	r := chi.NewRouter()

	r.Get("/", h.List)
	r.Post("/", h.Create)
	r.Route("/{id}", func(r chi.Router) {
		r.Get("/", h.Get)
		r.Put("/", h.Update)
		r.Delete("/", h.Delete)
	})

	return r
}

// GET /items?active=true&search=foo&limit=20&offset=0
func (h *ItemHandler) List(w http.ResponseWriter, r *http.Request) {
	filter := domain.ItemFilter{
		ActiveOnly: r.URL.Query().Get("active") == "true",
		Search:     r.URL.Query().Get("search"),
		Limit:      queryInt(r, "limit", 20),
		Offset:     queryInt(r, "offset", 0),
	}

	items, err := h.svc.List(r.Context(), filter)
	if err != nil {
		h.handleError(w, err)
		return
	}

	respondJSON(w, http.StatusOK, items)
}

// GET /items/{id}
func (h *ItemHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid item ID")
		return
	}

	item, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		h.handleError(w, err)
		return
	}

	respondJSON(w, http.StatusOK, item)
}

// POST /items
func (h *ItemHandler) Create(w http.ResponseWriter, r *http.Request) {
	var input domain.CreateItemInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		respondError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	item, err := h.svc.Create(r.Context(), input)
	if err != nil {
		h.handleError(w, err)
		return
	}

	respondJSON(w, http.StatusCreated, item)
}

// PUT /items/{id}
func (h *ItemHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid item ID")
		return
	}

	var input domain.UpdateItemInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		respondError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	item, err := h.svc.Update(r.Context(), id, input)
	if err != nil {
		h.handleError(w, err)
		return
	}

	respondJSON(w, http.StatusOK, item)
}

// DELETE /items/{id}
func (h *ItemHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid item ID")
		return
	}

	if err := h.svc.Delete(r.Context(), id); err != nil {
		h.handleError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// handleError мапит доменные ошибки на HTTP-статусы.
func (h *ItemHandler) handleError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrNotFound):
		respondError(w, http.StatusNotFound, err.Error())
	case errors.Is(err, domain.ErrDuplicateTitle):
		respondError(w, http.StatusConflict, err.Error())
	case errors.Is(err, domain.ErrTitleRequired),
		errors.Is(err, domain.ErrNegativePrice):
		respondError(w, http.StatusBadRequest, err.Error())
	default:
		h.logger.Error("internal error", slog.String("error", err.Error()))
		respondError(w, http.StatusInternalServerError, "internal server error")
	}
}

type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
	Code    int    `json:"code"`
}

func respondJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func respondError(w http.ResponseWriter, status int, message string) {
	respondJSON(w, status, ErrorResponse{
		Error:   http.StatusText(status),
		Message: message,
		Code:    status,
	})
}

func parseID(r *http.Request) (int64, error) {
	return strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
}

func queryInt(r *http.Request, key string, defaultVal int) int {
	val := r.URL.Query().Get(key)
	if val == "" {
		return defaultVal
	}
	n, err := strconv.Atoi(val)
	if err != nil {
		return defaultVal
	}
	return n
}
