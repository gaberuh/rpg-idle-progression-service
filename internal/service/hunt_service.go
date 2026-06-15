package service

import (
	"context"

	"github.com/google/uuid"
	"github.com/gaberuh/rpg-idle-progression-service/internal/domain"
	"github.com/gaberuh/rpg-idle-progression-service/internal/repository"
)

const (
	DefaultPageSize = 20
	MaxPageSize     = 100
)

type SessionResult struct {
	Session    domain.HuntSession
	HuntName   string
	KillCounts []domain.SessionKillCount
	Loot       []domain.SessionLootEntry
}

type HuntService interface {
	// ListHunts retorna hunts paginadas. cursor nil = primeira página.
	// Retorna os itens + o próximo cursor (nil se última página).
	ListHunts(ctx context.Context, cursor *repository.HuntCursor, limit int) ([]domain.Hunt, *repository.HuntCursor, error)
	StartHunt(ctx context.Context, characterID uuid.UUID, huntID uuid.UUID, durationMinutes int, snapshot domain.HuntSession) error
	StopHunt(ctx context.Context, characterID uuid.UUID) error
	GetActiveSession(ctx context.Context, characterID uuid.UUID) (*domain.HuntSession, error)
	GetSessionResult(ctx context.Context, characterID uuid.UUID, sessionID uuid.UUID) (*SessionResult, error)
}
