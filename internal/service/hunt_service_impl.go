package service

import (
	"context"
	"time"

	"github.com/google/uuid"
	apperr "github.com/gaberuh/rpg-idle-progression-service/internal/errors"
	"github.com/gaberuh/rpg-idle-progression-service/internal/domain"
	"github.com/gaberuh/rpg-idle-progression-service/internal/event"
	"github.com/gaberuh/rpg-idle-progression-service/internal/event/dto"
	"github.com/gaberuh/rpg-idle-progression-service/internal/repository"
)

const (
	maxDurationMinutes = 360
	minDurationMinutes = 1
)

type huntServiceImpl struct {
	repo     repository.HuntRepository
	producer event.HuntProducer
}

func NewHuntService(repo repository.HuntRepository, producer event.HuntProducer) HuntService {
	return &huntServiceImpl{repo: repo, producer: producer}
}

func (s *huntServiceImpl) ListHunts(ctx context.Context) ([]domain.Hunt, error) {
	return s.repo.ListHunts(ctx)
}

func (s *huntServiceImpl) StartHunt(
	ctx context.Context,
	characterID uuid.UUID,
	huntID uuid.UUID,
	durationMinutes int,
	snapshot domain.HuntSession,
) error {
	if durationMinutes < minDurationMinutes || durationMinutes > maxDurationMinutes {
		return apperr.ErrInvalidDuration
	}

	hunt, err := s.repo.GetHuntByID(ctx, huntID)
	if err != nil {
		return err
	}

	if snapshot.SnapshotLevel < hunt.RecommendedLevel/2 {
		return apperr.ErrLevelTooLow
	}

	// Verifica se já existe sessão ativa
	existing, err := s.repo.GetActiveSession(ctx, characterID)
	if err == nil && existing != nil {
		return apperr.ErrHuntAlreadyActive
	}

	now := time.Now().UTC()
	session := domain.HuntSession{
		ID:                 uuid.New(),
		CharacterID:        characterID,
		HuntID:             huntID,
		Status:             domain.SessionRunning,
		StartedAt:          now,
		ConfiguredDuration: durationMinutes,
		LastResolvedAt:     now,
		SnapshotEquipment:  snapshot.SnapshotEquipment,
		SnapshotSkills:     snapshot.SnapshotSkills,
		SnapshotLevel:      snapshot.SnapshotLevel,
		SnapshotVocation:   snapshot.SnapshotVocation,
	}

	return s.repo.CreateSession(ctx, session)
}

func (s *huntServiceImpl) StopHunt(ctx context.Context, characterID uuid.UUID) error {
	session, err := s.repo.GetActiveSession(ctx, characterID)
	if err != nil {
		return err
	}

	endedBy := domain.EndedByPlayerStopped
	return s.repo.EndSession(ctx, session.ID, endedBy, domain.SessionPendingClaim, time.Now().UTC())
}

func (s *huntServiceImpl) GetActiveSession(ctx context.Context, characterID uuid.UUID) (*domain.HuntSession, error) {
	return s.repo.GetActiveSession(ctx, characterID)
}

// completeSession é chamado pelo worker quando a sessão chega ao fim do tempo configurado.
func (s *huntServiceImpl) completeSession(ctx context.Context, session domain.HuntSession) error {
	endedBy := domain.EndedByCompleted
	if err := s.repo.EndSession(ctx, session.ID, endedBy, domain.SessionPendingClaim, time.Now().UTC()); err != nil {
		return err
	}

	return s.producer.PublishHuntCompleted(ctx, dto.HuntSessionCompleted{
		SessionID:   session.ID,
		CharacterID: session.CharacterID,
		EndedBy:     string(endedBy),
		CompletedAt: time.Now().UTC(),
	})
}
