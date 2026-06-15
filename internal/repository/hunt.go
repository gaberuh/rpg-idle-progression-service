package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/gaberuh/rpg-idle-progression-service/internal/domain"
)

// HuntCursor é o cursor de paginação para ListHunts.
// Codifica a posição da última hunt retornada como (recommended_level, id).
type HuntCursor struct {
	RecommendedLevel int
	ID               uuid.UUID
}

type HuntRepository interface {
	// Catalog
	GetHuntByID(ctx context.Context, id uuid.UUID) (*domain.Hunt, error)
	// ListHunts retorna hunts com keyset pagination.
	// cursor: último (recommended_level, id) visto — nil = primeira página.
	// limit: quantidade de itens por página (máx 100).
	ListHunts(ctx context.Context, cursor *HuntCursor, limit int) ([]domain.Hunt, error)
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

	// GetCharacterBlessings retorna a quantidade de blessings ativas do personagem.
	// Usado pelo worker para calcular penalidades no evento DeathOccurred.
	GetCharacterBlessings(ctx context.Context, characterID uuid.UUID) (int, error)

	// GetSessionKillCounts retorna kills por monstro com nome, ordenado por kill_count DESC.
	GetSessionKillCounts(ctx context.Context, sessionID uuid.UUID) ([]domain.SessionKillCount, error)

	// GetSessionLoot retorna o loot da sessão com nome do item, ordenado por raridade DESC.
	GetSessionLoot(ctx context.Context, sessionID uuid.UUID) ([]domain.SessionLootEntry, error)
}
