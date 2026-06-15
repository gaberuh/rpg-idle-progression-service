package handler

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	apperr "github.com/gaberuh/rpg-idle-progression-service/internal/errors"
	httpdto "github.com/gaberuh/rpg-idle-progression-service/internal/dto"
	"github.com/gaberuh/rpg-idle-progression-service/internal/middleware"
	"github.com/gaberuh/rpg-idle-progression-service/internal/repository"
	"github.com/gaberuh/rpg-idle-progression-service/internal/service"
)

var validate = validator.New()

type HuntHandler struct {
	svc service.HuntService
}

func NewHuntHandler(svc service.HuntService) *HuntHandler {
	return &HuntHandler{svc: svc}
}

// ListHunts godoc
// @Summary     Lista hunts disponíveis com paginação por cursor
// @Tags        hunts
// @Security    BearerAuth
// @Produce     json
// @Param       cursor query string false "Cursor da página anterior (opaco, retornado em next_cursor)"
// @Param       limit  query int    false "Itens por página (padrão 20, máx 100)"
// @Success     200 {object} dto.ListHuntsResponse
// @Router      /api/v1/hunts [get]
func (h *HuntHandler) ListHunts(w http.ResponseWriter, r *http.Request) {
	playerID, ok := middleware.PlayerIDFromCtx(r.Context())
	if !ok {
		writeErr(w, apperr.ErrUnauthorized)
		return
	}

	limit := service.DefaultPageSize
	if v := r.URL.Query().Get("limit"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n <= 0 {
			writeErr(w, apperr.ErrValidation)
			return
		}
		limit = n
	}

	var cursor *repository.HuntCursor
	if v := r.URL.Query().Get("cursor"); v != "" {
		c, err := decodeCursor(v)
		if err != nil {
			writeErr(w, apperr.ErrValidation)
			return
		}
		cursor = c
	}

	hunts, nextCursor, err := h.svc.ListHunts(r.Context(), playerID, cursor, limit)
	if err != nil {
		writeErr(w, err)
		return
	}

	items := make([]httpdto.HuntResponse, len(hunts))
	for i, hunt := range hunts {
		items[i] = httpdto.HuntResponse{
			ID:               hunt.ID,
			Name:             hunt.Name,
			RecommendedLevel: hunt.RecommendedLevel,
			Difficulty:       string(hunt.Difficulty),
			XPPerHour:        hunt.XPPerHour,
			GoldPerHour:      hunt.GoldPerHour,
			Available:        hunt.Available,
		}
	}

	var nextCursorStr *string
	if nextCursor != nil {
		s := encodeCursor(nextCursor)
		nextCursorStr = &s
	}

	writeJSON(w, http.StatusOK, httpdto.ListHuntsResponse{
		Items:      items,
		NextCursor: nextCursorStr,
		Total:      len(items),
	})
}

// cursorPayload é o formato interno serializado em base64 no cursor opaco.
type cursorPayload struct {
	Level int    `json:"l"`
	ID    string `json:"i"`
}

func encodeCursor(c *repository.HuntCursor) string {
	b, _ := json.Marshal(cursorPayload{Level: c.RecommendedLevel, ID: c.ID.String()})
	return base64.RawURLEncoding.EncodeToString(b)
}

func decodeCursor(s string) (*repository.HuntCursor, error) {
	b, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		return nil, fmt.Errorf("cursor inválido")
	}
	var p cursorPayload
	if err := json.Unmarshal(b, &p); err != nil {
		return nil, fmt.Errorf("cursor inválido")
	}
	id, err := uuid.Parse(p.ID)
	if err != nil {
		return nil, fmt.Errorf("cursor inválido")
	}
	return &repository.HuntCursor{RecommendedLevel: p.Level, ID: id}, nil
}

// StartHunt godoc
// @Summary     Inicia uma sessão de hunt
// @Tags        hunts
// @Security    BearerAuth
// @Accept      json
// @Produce     json
// @Param       hunt_id path string true "ID da hunt"
// @Param       body    body dto.StartHuntRequest true "Payload"
// @Success     201 {object} dto.StartHuntResponse
// @Router      /api/v1/hunts/{hunt_id}/start [post]
func (h *HuntHandler) StartHunt(w http.ResponseWriter, r *http.Request) {
	playerID, ok := middleware.PlayerIDFromCtx(r.Context())
	if !ok {
		writeErr(w, apperr.ErrUnauthorized)
		return
	}

	huntID, err := uuid.Parse(chi.URLParam(r, "hunt_id"))
	if err != nil {
		writeErr(w, apperr.ErrValidation)
		return
	}

	var req httpdto.StartHuntRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, apperr.ErrValidation)
		return
	}
	if err := validate.Struct(req); err != nil {
		writeErr(w, apperr.ErrValidation)
		return
	}

	result, err := h.svc.StartHunt(r.Context(), playerID, huntID, req.DurationMinutes)
	if err != nil {
		writeErr(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, httpdto.StartHuntResponse{
		SessionID:              result.SessionID,
		HuntID:                 result.HuntID,
		HuntName:               result.HuntName,
		StartedAt:              result.StartedAt,
		ConfiguredDurationMins: result.ConfiguredDurationMins,
		EstimatedEndAt:         result.EstimatedEndAt,
	})
}

// StopHunt godoc
// @Summary     Para a hunt em andamento
// @Tags        hunts
// @Security    BearerAuth
// @Produce     json
// @Success     200 {object} dto.StopHuntResponse
// @Router      /api/v1/hunts/current/stop [post]
func (h *HuntHandler) StopHunt(w http.ResponseWriter, r *http.Request) {
	playerID, ok := middleware.PlayerIDFromCtx(r.Context())
	if !ok {
		writeErr(w, apperr.ErrUnauthorized)
		return
	}

	result, err := h.svc.StopHunt(r.Context(), playerID)
	if err != nil {
		writeErr(w, err)
		return
	}

	writeJSON(w, http.StatusOK, httpdto.StopHuntResponse{
		SessionID:       result.SessionID,
		EndedBy:         result.EndedBy,
		XPGained:        result.XPGained,
		GoldGained:      result.GoldGained,
		DurationMinutes: result.DurationMinutes,
	})
}

// GetActiveSession godoc
// @Summary     Retorna a sessão de hunt ativa
// @Tags        hunts
// @Security    BearerAuth
// @Produce     json
// @Success     200 {object} dto.ActiveSessionResponse
// @Router      /api/v1/hunts/current [get]
func (h *HuntHandler) GetActiveSession(w http.ResponseWriter, r *http.Request) {
	playerID, ok := middleware.PlayerIDFromCtx(r.Context())
	if !ok {
		writeErr(w, apperr.ErrUnauthorized)
		return
	}

	result, err := h.svc.GetActiveSession(r.Context(), playerID)
	if err != nil {
		writeErr(w, err)
		return
	}

	s := result.Session
	writeJSON(w, http.StatusOK, httpdto.ActiveSessionResponse{
		SessionID:              s.ID,
		HuntName:               result.HuntName,
		Status:                 string(s.Status),
		StartedAt:              s.StartedAt,
		EstimatedEndAt:         result.EstimatedEndAt,
		ElapsedMinutes:         result.ElapsedMinutes,
		ConfiguredDurationMins: s.ConfiguredDuration,
		XPGained:               s.XPGained,
		GoldGained:             s.GoldGained,
	})
}

// GetSessionResult godoc
// @Summary     Retorna o resultado completo de uma sessão encerrada
// @Tags        hunts
// @Security    BearerAuth
// @Produce     json
// @Param       session_id path string true "ID da sessão"
// @Success     200 {object} dto.SessionResultResponse
// @Router      /api/v1/hunts/sessions/{session_id} [get]
func (h *HuntHandler) GetSessionResult(w http.ResponseWriter, r *http.Request) {
	playerID, ok := middleware.PlayerIDFromCtx(r.Context())
	if !ok {
		writeErr(w, apperr.ErrUnauthorized)
		return
	}

	sessionID, err := uuid.Parse(chi.URLParam(r, "session_id"))
	if err != nil {
		writeErr(w, apperr.ErrValidation)
		return
	}

	result, err := h.svc.GetSessionResult(r.Context(), playerID, sessionID)
	if err != nil {
		writeErr(w, err)
		return
	}

	s := result.Session
	durationMinutes := s.ConfiguredDuration
	if s.EndedAt != nil {
		durationMinutes = int(s.EndedAt.Sub(s.StartedAt).Minutes())
	}

	var endedByStr *string
	if s.EndedBy != nil {
		v := string(*s.EndedBy)
		endedByStr = &v
	}

	kills := make([]httpdto.SessionKillCount, len(result.KillCounts))
	for i, kc := range result.KillCounts {
		kills[i] = httpdto.SessionKillCount{MonsterName: kc.MonsterName, Kills: kc.KillCount}
	}

	loot := make([]httpdto.SessionLootItem, len(result.Loot))
	for i, l := range result.Loot {
		item := httpdto.SessionLootItem{
			Name:     l.ItemName,
			Rarity:   l.Rarity,
			Quantity: l.Quantity,
		}
		if len(l.ItemIDs) == 1 {
			item.ItemID = &l.ItemIDs[0]
		}
		loot[i] = item
	}

	writeJSON(w, http.StatusOK, httpdto.SessionResultResponse{
		SessionID:       s.ID,
		HuntName:        result.HuntName,
		Status:          string(s.Status),
		EndedBy:         endedByStr,
		StartedAt:       s.StartedAt,
		EndedAt:         s.EndedAt,
		DurationMinutes: durationMinutes,
		XPGained:        s.XPGained,
		GoldGained:      s.GoldGained,
		DeathCount:      s.DeathCount,
		KillCounts:      kills,
		Loot:            loot,
	})
}

func writeErr(w http.ResponseWriter, err error) {
	var appErr *apperr.AppError
	if e, ok := err.(*apperr.AppError); ok {
		appErr = e
	} else {
		appErr = apperr.ErrInternal
	}

	if appErr.StatusCode >= 500 {
		slog.Error("internal error", "err", err)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(appErr.StatusCode)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"code":    appErr.Code,
		"message": appErr.Message,
	})
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
