package dto

import (
	"time"

	"github.com/google/uuid"
)

// SessionKillCount é o total de kills de um monstro na sessão.
type SessionKillCount struct {
	MonsterName string `json:"monster_name"`
	Kills       int    `json:"kills"`
}

// SessionLootItem é um item do loot da sessão.
type SessionLootItem struct {
	Name     string     `json:"name"`
	Rarity   string     `json:"rarity"`
	Quantity int        `json:"quantity"`
	ItemID   *uuid.UUID `json:"item_id,omitempty"` // preenchido apenas para itens únicos
}

// SessionResultResponse é o resultado completo de uma sessão encerrada.
type SessionResultResponse struct {
	SessionID       uuid.UUID          `json:"session_id"`
	HuntName        string             `json:"hunt_name"`
	Status          string             `json:"status"`
	EndedBy         *string            `json:"ended_by,omitempty"`
	StartedAt       time.Time          `json:"started_at"`
	EndedAt         *time.Time         `json:"ended_at,omitempty"`
	DurationMinutes int                `json:"duration_minutes"`
	XPGained        int64              `json:"xp_gained"`
	GoldGained      int64              `json:"gold_gained"`
	DeathCount      int                `json:"death_count"`
	KillCounts      []SessionKillCount `json:"kill_counts"`
	Loot            []SessionLootItem  `json:"loot"`
}

// StartHuntRequest é o payload para iniciar uma hunt.
// hunt_id vem do path /:hunt_id/start, não do body.
type StartHuntRequest struct {
	DurationMinutes int             `json:"duration_minutes" validate:"required,min=1,max=360"`
	Snapshot        SnapshotPayload `json:"snapshot"         validate:"required"`
}

type SnapshotPayload struct {
	Level      int                       `json:"level"`
	Vocation   string                    `json:"vocation"`
	Skills     map[string]SkillSnapshot  `json:"skills"`
	Equipment  map[string]*ItemSnapshot  `json:"equipment"`
}

type SkillSnapshot struct {
	Level int `json:"level"`
}

type ItemSnapshot struct {
	ItemID  string `json:"item_id"`
	Name    string `json:"name"`
	Attack  int    `json:"attack,omitempty"`
	Defense int    `json:"defense,omitempty"`
	Armor   int    `json:"armor,omitempty"`
}

// HuntResponse é a representação pública de uma hunt do catálogo.
type HuntResponse struct {
	ID               uuid.UUID `json:"id"`
	Name             string    `json:"name"`
	RecommendedLevel int       `json:"recommended_level"`
	Difficulty       string    `json:"difficulty"`
	XPPerHour        int64     `json:"xp_per_hour"`
	GoldPerHour      int64     `json:"gold_per_hour"`
}

// ListHuntsResponse envolve os itens e o cursor para a próxima página.
// O front envia o next_cursor de volta via query param ?cursor= na próxima requisição.
type ListHuntsResponse struct {
	Items      []HuntResponse `json:"items"`
	NextCursor *string        `json:"next_cursor"` // nil = última página
	Total      int            `json:"total"`        // total de itens nesta página
}

// ActiveSessionResponse é o estado atual da sessão de hunt.
type ActiveSessionResponse struct {
	SessionID          uuid.UUID  `json:"session_id"`
	HuntID             uuid.UUID  `json:"hunt_id"`
	Status             string     `json:"status"`
	StartedAt          time.Time  `json:"started_at"`
	ConfiguredDuration int        `json:"configured_duration_minutes"`
	XPGained           int64      `json:"xp_gained"`
	GoldGained         int64      `json:"gold_gained"`
	DeathCount         int        `json:"death_count"`
}
