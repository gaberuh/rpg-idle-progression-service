package service

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/gaberuh/rpg-idle-progression-service/internal/domain"
	"github.com/gaberuh/rpg-idle-progression-service/internal/repository"
)

const (
	DefaultPageSize = 20
	MaxPageSize     = 100
)

// HuntWithAvailability é uma hunt do catálogo com o campo available calculado para o personagem.
type HuntWithAvailability struct {
	domain.Hunt
	Available bool
}

// StartHuntResult é o payload de resposta 201 de POST /api/v1/hunts/:hunt_id/start.
type StartHuntResult struct {
	SessionID              uuid.UUID
	HuntID                 uuid.UUID
	HuntName               string
	StartedAt              time.Time
	ConfiguredDurationMins int
	EstimatedEndAt         time.Time
}

// StopHuntResult é o payload de resposta 200 de POST /api/v1/hunts/current/stop.
type StopHuntResult struct {
	SessionID       uuid.UUID
	EndedBy         string
	XPGained        int64
	GoldGained      int64
	DurationMinutes int
}

// ActiveSessionResult é o payload de resposta 200 de GET /api/v1/hunts/current.
type ActiveSessionResult struct {
	Session        domain.HuntSession
	HuntName       string
	ElapsedMinutes int
	EstimatedEndAt time.Time
}

// SessionResult é o payload de resposta 200 de GET /api/v1/hunts/sessions/:session_id.
type SessionResult struct {
	Session    domain.HuntSession
	HuntName   string
	KillCounts []domain.SessionKillCount
	Loot       []domain.SessionLootEntry
}

type HuntService interface {
	// ListHunts retorna hunts paginadas com o campo available calculado para o personagem.
	ListHunts(ctx context.Context, characterID uuid.UUID, cursor *repository.HuntCursor, limit int) ([]HuntWithAvailability, *repository.HuntCursor, error)

	// StartHunt inicia uma hunt capturando o snapshot do personagem no servidor.
	StartHunt(ctx context.Context, characterID uuid.UUID, huntID uuid.UUID, durationMinutes int) (*StartHuntResult, error)

	// StopHunt para a hunt em andamento e retorna o resumo.
	StopHunt(ctx context.Context, characterID uuid.UUID) (*StopHuntResult, error)

	// GetActiveSession retorna a sessão em andamento com dados calculados (elapsed, estimatedEnd).
	GetActiveSession(ctx context.Context, characterID uuid.UUID) (*ActiveSessionResult, error)

	// GetSessionResult retorna o resultado completo de uma sessão encerrada.
	GetSessionResult(ctx context.Context, characterID uuid.UUID, sessionID uuid.UUID) (*SessionResult, error)
}
