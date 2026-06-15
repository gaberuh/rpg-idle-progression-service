package dto

import (
	"time"

	"github.com/google/uuid"
)

// HuntSessionStarted é publicado quando o jogador inicia uma hunt.
// Consumido pelo leaderboard-service e analytics.
type HuntSessionStarted struct {
	SessionID   uuid.UUID `json:"session_id"`
	CharacterID uuid.UUID `json:"character_id"`
	HuntID      uuid.UUID `json:"hunt_id"`
	StartedAt   time.Time `json:"started_at"`
}

// HuntTickResolved é publicado a cada tick do worker para sessões em andamento.
// Consumido pelo character-service para acumular XP de skills de forma incremental.
type HuntTickResolved struct {
	SessionID    uuid.UUID      `json:"session_id"`
	CharacterID  uuid.UUID      `json:"character_id"`
	HuntID       uuid.UUID      `json:"hunt_id"`
	XPGained     int64          `json:"xp_gained"`
	GoldGained   int64          `json:"gold_gained"`
	PhysicalHits int            `json:"physical_hits"`
	ManaConsumed int64          `json:"mana_consumed"`
	Vocation     string         `json:"vocation"`
	Kills        map[string]int `json:"kills"` // monster_id → count do intervalo
	ResolvedAt   time.Time      `json:"resolved_at"`
}

// HuntSessionResolved é publicado quando a sessão encerra (por qualquer motivo).
// Consumido pelo character-service para aplicar XP total, gold e atualizar status.
type HuntSessionResolved struct {
	SessionID       uuid.UUID `json:"session_id"`
	CharacterID     uuid.UUID `json:"character_id"`
	HuntID          uuid.UUID `json:"hunt_id"`
	EndedBy         string    `json:"ended_by"`
	XPGained        int64     `json:"xp_gained"`
	GoldGained      int64     `json:"gold_gained"`
	PhysicalHits    int       `json:"physical_hits"`
	ManaConsumed    int64     `json:"mana_consumed"`
	DurationMinutes int       `json:"duration_minutes"`
	Vocation        string    `json:"vocation"`
	ResolvedAt      time.Time `json:"resolved_at"`
}

// DeathOccurred é publicado quando o simulador detecta morte durante a hunt.
// Consumido pelo character-service para aplicar penalidades e pelo notifications-service.
// As penalidades já vêm calculadas — o character-service apenas aplica.
type DeathOccurred struct {
	SessionID           uuid.UUID `json:"session_id"`
	CharacterID         uuid.UUID `json:"character_id"`
	HuntID              uuid.UUID `json:"hunt_id"`
	BlessingsAtDeath    int       `json:"blessings_at_death"`
	XPPenaltyPercent    int       `json:"xp_penalty_percent"`
	SkillPenaltyPercent int       `json:"skill_penalty_percent"`
	DiedAt              time.Time `json:"died_at"`
}
