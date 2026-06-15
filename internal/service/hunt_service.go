package service

import (
	"context"

	"github.com/google/uuid"
	"github.com/gaberuh/rpg-idle-progression-service/internal/domain"
)

type HuntService interface {
	ListHunts(ctx context.Context) ([]domain.Hunt, error)
	StartHunt(ctx context.Context, characterID uuid.UUID, huntID uuid.UUID, durationMinutes int, snapshot domain.HuntSession) error
	StopHunt(ctx context.Context, characterID uuid.UUID) error
	GetActiveSession(ctx context.Context, characterID uuid.UUID) (*domain.HuntSession, error)
}
