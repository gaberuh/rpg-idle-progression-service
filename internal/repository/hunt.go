package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/gaberuh/rpg-idle-progression-service/internal/domain"
)

type HuntRepository interface {
	// Catalog
	GetHuntByID(ctx context.Context, id uuid.UUID) (*domain.Hunt, error)
	ListHunts(ctx context.Context) ([]domain.Hunt, error)
	GetHuntMonsters(ctx context.Context, huntID uuid.UUID) ([]domain.MonsterWithSpawnRate, error)

	// Sessions
	CreateSession(ctx context.Context, session domain.HuntSession) error
	GetActiveSession(ctx context.Context, characterID uuid.UUID) (*domain.HuntSession, error)
	GetSessionByID(ctx context.Context, id uuid.UUID) (*domain.HuntSession, error)
	UpdateSessionProgress(ctx context.Context, id uuid.UUID, xpGained, goldGained int64, deaths int, resolvedAt time.Time) error
	EndSession(ctx context.Context, id uuid.UUID, endedBy domain.EndedBy, status domain.SessionStatus, endedAt time.Time) error
	UpsertKillCounts(ctx context.Context, sessionID uuid.UUID, kills map[uuid.UUID]int) error
	UpsertSessionLoot(ctx context.Context, sessionID uuid.UUID, loot map[uuid.UUID]domain.LootDrop) error

	// Worker: cursor-based pagination
	ListRunningSessions(ctx context.Context, after time.Time, limit int) ([]domain.HuntSession, error)
}
