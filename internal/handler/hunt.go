package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	apperr "github.com/gaberuh/rpg-idle-progression-service/internal/errors"
	"github.com/gaberuh/rpg-idle-progression-service/internal/domain"
	httpdto "github.com/gaberuh/rpg-idle-progression-service/internal/dto"
	"github.com/gaberuh/rpg-idle-progression-service/internal/middleware"
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
// @Summary     Lista hunts disponíveis
// @Tags        hunts
// @Security    BearerAuth
// @Produce     json
// @Success     200 {array} dto.HuntResponse
// @Router      /api/v1/hunts [get]
func (h *HuntHandler) ListHunts(w http.ResponseWriter, r *http.Request) {
	hunts, err := h.svc.ListHunts(r.Context())
	if err != nil {
		writeErr(w, err)
		return
	}

	resp := make([]httpdto.HuntResponse, len(hunts))
	for i, hunt := range hunts {
		resp[i] = toHuntResponse(hunt)
	}
	writeJSON(w, http.StatusOK, resp)
}

// StartHunt godoc
// @Summary     Inicia uma sessão de hunt
// @Tags        hunts
// @Security    BearerAuth
// @Accept      json
// @Produce     json
// @Param       body body dto.StartHuntRequest true "Payload"
// @Success     201
// @Router      /api/v1/hunts/start [post]
func (h *HuntHandler) StartHunt(w http.ResponseWriter, r *http.Request) {
	playerID, ok := middleware.PlayerIDFromCtx(r.Context())
	if !ok {
		writeErr(w, apperr.ErrUnauthorized)
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

	// Monta snapshot a partir do payload
	snapshot := buildSnapshot(req.Snapshot)

	if err := h.svc.StartHunt(r.Context(), playerID, req.HuntID, req.DurationMinutes, snapshot); err != nil {
		writeErr(w, err)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

// StopHunt godoc
// @Summary     Para a hunt em andamento
// @Tags        hunts
// @Security    BearerAuth
// @Produce     json
// @Success     200
// @Router      /api/v1/hunts/stop [post]
func (h *HuntHandler) StopHunt(w http.ResponseWriter, r *http.Request) {
	playerID, ok := middleware.PlayerIDFromCtx(r.Context())
	if !ok {
		writeErr(w, apperr.ErrUnauthorized)
		return
	}

	if err := h.svc.StopHunt(r.Context(), playerID); err != nil {
		writeErr(w, err)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// GetActiveSession godoc
// @Summary     Retorna a sessão de hunt ativa
// @Tags        hunts
// @Security    BearerAuth
// @Produce     json
// @Success     200 {object} dto.ActiveSessionResponse
// @Router      /api/v1/hunts/active [get]
func (h *HuntHandler) GetActiveSession(w http.ResponseWriter, r *http.Request) {
	playerID, ok := middleware.PlayerIDFromCtx(r.Context())
	if !ok {
		writeErr(w, apperr.ErrUnauthorized)
		return
	}

	session, err := h.svc.GetActiveSession(r.Context(), playerID)
	if err != nil {
		writeErr(w, err)
		return
	}

	writeJSON(w, http.StatusOK, httpdto.ActiveSessionResponse{
		SessionID:          session.ID,
		HuntID:             session.HuntID,
		Status:             string(session.Status),
		StartedAt:          session.StartedAt,
		ConfiguredDuration: session.ConfiguredDuration,
		XPGained:           session.XPGained,
		GoldGained:         session.GoldGained,
		DeathCount:         session.DeathCount,
	})
}

func toHuntResponse(h domain.Hunt) httpdto.HuntResponse {
	return httpdto.HuntResponse{
		ID:               h.ID,
		Name:             h.Name,
		RecommendedLevel: h.RecommendedLevel,
		Difficulty:       string(h.Difficulty),
		XPPerHour:        h.XPPerHour,
		GoldPerHour:      h.GoldPerHour,
	}
}

func buildSnapshot(p httpdto.SnapshotPayload) domain.HuntSession {
	skills := make(domain.SnapshotSkills, len(p.Skills))
	for k, v := range p.Skills {
		skills[k] = domain.SnapshotSkill{Level: v.Level}
	}

	equipment := make(domain.SnapshotEquipment, len(p.Equipment))
	for k, v := range p.Equipment {
		if v == nil {
			equipment[k] = nil
			continue
		}
		equipment[k] = &domain.SnapshotItem{
			ItemID:  v.ItemID,
			Name:    v.Name,
			Attack:  v.Attack,
			Defense: v.Defense,
			Armor:   v.Armor,
		}
	}

	return domain.HuntSession{
		SnapshotLevel:     p.Level,
		SnapshotVocation:  domain.Vocation(p.Vocation),
		SnapshotSkills:    skills,
		SnapshotEquipment: equipment,
	}
}

// writeErr escreve um erro HTTP padronizado.
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

// Satisfaz o compilador — uuid é usado indiretamente via middleware
var _ = uuid.Nil
