package dto

import (
	"time"

	"github.com/google/uuid"
)

// HuntSessionResolved é publicado após cada tick do worker resolver uma sessão.
// Consumido pelo character-service para acumular XP, gold e skill XP.
type HuntSessionResolved struct {
	SessionID   uuid.UUID         `json:"session_id"`
	CharacterID uuid.UUID         `json:"character_id"`
	HuntID      uuid.UUID         `json:"hunt_id"`
	XPGained    int64             `json:"xp_gained"`
	GoldGained  int64             `json:"gold_gained"`
	PhysicalHits int              `json:"physical_hits"`
	ManaConsumed int64            `json:"mana_consumed"`
	Kills       map[string]int    `json:"kills"` // monster_id → count
	ResolvedAt  time.Time         `json:"resolved_at"`
}

// DeathOccurred é publicado quando o simulador determina que o personagem morreu durante a hunt.
// Consumido pelo character-service para aplicar penalidade de XP e zerar bençãos.
type DeathOccurred struct {
	SessionID   uuid.UUID `json:"session_id"`
	CharacterID uuid.UUID `json:"character_id"`
	HuntID      uuid.UUID `json:"hunt_id"`
	DiedAt      time.Time `json:"died_at"`
}

// HuntSessionCompleted é publicado quando a sessão encerra (completed, stopped ou death).
// Consumido pelo character-service para colocar o personagem de volta em idle.
type HuntSessionCompleted struct {
	SessionID   uuid.UUID `json:"session_id"`
	CharacterID uuid.UUID `json:"character_id"`
	EndedBy     string    `json:"ended_by"`
	CompletedAt time.Time `json:"completed_at"`
}
