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

func (s *huntServiceImpl) ListHunts(ctx context.Context, characterID uuid.UUID, cursor *repository.HuntCursor, limit int) ([]HuntWithAvailability, *repository.HuntCursor, error) {
	if limit <= 0 {
		limit = DefaultPageSize
	}
	if limit > MaxPageSize {
		limit = MaxPageSize
	}

	characterLevel := 0
	if level, _, err := s.repo.GetCharacterLevel(ctx, characterID); err == nil {
		characterLevel = level
	}
	// Se não há personagem ainda, characterLevel = 0 → todas as hunts retornam available: false.

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

	result := make([]HuntWithAvailability, len(hunts))
	for i, h := range hunts {
		result[i] = HuntWithAvailability{
			Hunt:      h,
			Available: characterLevel >= h.RecommendedLevel,
		}
	}

	return result, nextCursor, nil
}

func (s *huntServiceImpl) StartHunt(
	ctx context.Context,
	characterID uuid.UUID,
	huntID uuid.UUID,
	durationMinutes int,
) (*StartHuntResult, error) {
	if durationMinutes < minDurationMinutes || durationMinutes > maxDurationMinutes {
		return nil, apperr.ErrInvalidDuration
	}

	// Captura snapshot do personagem no servidor — valida status idle e level mínimo.
	snapshot, err := s.repo.GetCharacterSnapshot(ctx, characterID)
	if err != nil {
		return nil, err
	}

	hunt, err := s.repo.GetHuntByID(ctx, huntID)
	if err != nil {
		return nil, err
	}

	if snapshot.Level < hunt.RecommendedLevel {
		return nil, apperr.ErrLevelTooLow
	}

	existing, err := s.repo.GetActiveSession(ctx, characterID)
	if err == nil && existing != nil {
		return nil, apperr.ErrHuntAlreadyActive
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
		SnapshotEquipment:  snapshot.Equipment,
		SnapshotSkills:     snapshot.Skills,
		SnapshotLevel:      snapshot.Level,
		SnapshotVocation:   snapshot.Vocation,
	}

	if err := s.repo.CreateSession(ctx, session); err != nil {
		return nil, err
	}

	// Atualiza status do personagem para 'hunting' — guard para impedir ações concorrentes
	if err := s.repo.UpdateCharacterStatus(ctx, characterID, "hunting"); err != nil {
		return nil, err
	}

	_ = s.producer.PublishHuntStarted(ctx, dto.HuntSessionStarted{
		SessionID:   session.ID,
		CharacterID: characterID,
		HuntID:      huntID,
		StartedAt:   now,
	})

	return &StartHuntResult{
		SessionID:              session.ID,
		HuntID:                 huntID,
		HuntName:               hunt.Name,
		StartedAt:              now,
		ConfiguredDurationMins: durationMinutes,
		EstimatedEndAt:         now.Add(time.Duration(durationMinutes) * time.Minute),
	}, nil
}

func (s *huntServiceImpl) StopHunt(ctx context.Context, characterID uuid.UUID) (*StopHuntResult, error) {
	session, err := s.repo.GetActiveSession(ctx, characterID)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	endedBy := domain.EndedByPlayerStopped
	if err := s.repo.EndSession(ctx, session.ID, endedBy, domain.SessionPendingClaim, now); err != nil {
		return nil, err
	}

	// Atualiza status do personagem para 'pending_claim' — bloqueia novas ações até resgate do loot
	if err := s.repo.UpdateCharacterStatus(ctx, characterID, "pending_claim"); err != nil {
		return nil, err
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

	return &StopHuntResult{
		SessionID:       session.ID,
		EndedBy:         string(endedBy),
		XPGained:        session.XPGained,
		GoldGained:      session.GoldGained,
		DurationMinutes: durationMinutes,
	}, nil
}

func (s *huntServiceImpl) GetActiveSession(ctx context.Context, characterID uuid.UUID) (*ActiveSessionResult, error) {
	session, err := s.repo.GetActiveSession(ctx, characterID)
	if err != nil {
		return nil, err
	}

	hunt, err := s.repo.GetHuntByID(ctx, session.HuntID)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	elapsedMinutes := int(now.Sub(session.StartedAt).Minutes())
	estimatedEndAt := session.StartedAt.Add(time.Duration(session.ConfiguredDuration) * time.Minute)

	return &ActiveSessionResult{
		Session:        *session,
		HuntName:       hunt.Name,
		ElapsedMinutes: elapsedMinutes,
		EstimatedEndAt: estimatedEndAt,
	}, nil
}

func (s *huntServiceImpl) GetSessionResult(ctx context.Context, characterID uuid.UUID, sessionID uuid.UUID) (*SessionResult, error) {
	session, err := s.repo.GetSessionByID(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	if session.CharacterID != characterID {
		return nil, apperr.ErrSessionNotFound
	}

	hunt, err := s.repo.GetHuntByID(ctx, session.HuntID)
	if err != nil {
		return nil, err
	}

	kills, err := s.repo.GetSessionKillCounts(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	loot, err := s.repo.GetSessionLoot(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	return &SessionResult{
		Session:    *session,
		HuntName:   hunt.Name,
		KillCounts: kills,
		Loot:       loot,
	}, nil
}
