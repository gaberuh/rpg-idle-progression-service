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

func (s *huntServiceImpl) ListHunts(ctx context.Context, cursor *repository.HuntCursor, limit int) ([]domain.Hunt, *repository.HuntCursor, error) {
	if limit <= 0 {
		limit = DefaultPageSize
	}
	if limit > MaxPageSize {
		limit = MaxPageSize
	}

	hunts, err := s.repo.ListHunts(ctx, cursor, limit+1)
	if err != nil {
		return nil, nil, err
	}

	var nextCursor *repository.HuntCursor
	if len(hunts) > limit {
		last := hunts[limit-1]
		nextCursor = &repository.HuntCursor{
			RecommendedLevel: last.RecommendedLevel,
			ID:               last.ID,
		}
		hunts = hunts[:limit]
	}

	return hunts, nextCursor, nil
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

	if err := s.repo.CreateSession(ctx, session); err != nil {
		return err
	}

	_ = s.producer.PublishHuntStarted(ctx, dto.HuntSessionStarted{
		SessionID:   session.ID,
		CharacterID: characterID,
		HuntID:      huntID,
		StartedAt:   now,
	})

	return nil
}

func (s *huntServiceImpl) StopHunt(ctx context.Context, characterID uuid.UUID) error {
	session, err := s.repo.GetActiveSession(ctx, characterID)
	if err != nil {
		return err
	}

	now := time.Now().UTC()
	endedBy := domain.EndedByPlayerStopped
	if err := s.repo.EndSession(ctx, session.ID, endedBy, domain.SessionPendingClaim, now); err != nil {
		return err
	}

	durationMinutes := int(now.Sub(session.StartedAt).Minutes())

	_ = s.producer.PublishHuntResolved(ctx, dto.HuntSessionResolved{
		SessionID:       session.ID,
		CharacterID:     characterID,
		HuntID:          session.HuntID,
		EndedBy:         string(endedBy),
		XPGained:        session.XPGained,
		GoldGained:      session.GoldGained,
		DurationMinutes: durationMinutes,
		Vocation:        string(session.SnapshotVocation),
		ResolvedAt:      now,
	})

	return nil
}

func (s *huntServiceImpl) GetActiveSession(ctx context.Context, characterID uuid.UUID) (*domain.HuntSession, error) {
	return s.repo.GetActiveSession(ctx, characterID)
}
