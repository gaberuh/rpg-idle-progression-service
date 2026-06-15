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
// item_id é sempre presente: preenchido para itens únicos, null para stackable.
type SessionLootItem struct {
	Name     string     `json:"name"`
	Rarity   string     `json:"rarity"`
	Quantity int        `json:"quantity"`
	ItemID   *uuid.UUID `json:"item_id"`
}

// SessionResultResponse é o resultado completo de uma sessão encerrada.
type SessionResultResponse struct {
	SessionID       uuid.UUID          `json:"session_id"`
	HuntName        string             `json:"hunt_name"`
	Status          string             `json:"status"`
	EndedBy         *string            `json:"ended_by"`
	StartedAt       time.Time          `json:"started_at"`
	EndedAt         *time.Time         `json:"ended_at"`
	DurationMinutes int                `json:"duration_minutes"`
	XPGained        int64              `json:"xp_gained"`
	GoldGained      int64              `json:"gold_gained"`
	DeathCount      int                `json:"death_count"`
	KillCounts      []SessionKillCount `json:"kill_counts"`
	Loot            []SessionLootItem  `json:"loot"`
}

// StartHuntRequest é o payload de POST /api/v1/hunts/:hunt_id/start.
// O snapshot do personagem é capturado pelo servidor — não enviado pelo cliente.
type StartHuntRequest struct {
	DurationMinutes int `json:"duration_minutes" validate:"required,min=1,max=360"`
}

// StartHuntResponse é o payload de resposta 201 de POST /api/v1/hunts/:hunt_id/start.
type StartHuntResponse struct {
	SessionID              uuid.UUID `json:"session_id"`
	HuntID                 uuid.UUID `json:"hunt_id"`
	HuntName               string    `json:"hunt_name"`
	StartedAt              time.Time `json:"started_at"`
	ConfiguredDurationMins int       `json:"configured_duration_minutes"`
	EstimatedEndAt         time.Time `json:"estimated_end_at"`
}

// StopHuntResponse é o payload de resposta 200 de POST /api/v1/hunts/current/stop.
type StopHuntResponse struct {
	SessionID       uuid.UUID `json:"session_id"`
	EndedBy         string    `json:"ended_by"`
	XPGained        int64     `json:"xp_gained"`
	GoldGained      int64     `json:"gold_gained"`
	DurationMinutes int       `json:"duration_minutes"`
}

// HuntResponse é a representação pública de uma hunt do catálogo.
type HuntResponse struct {
	ID               uuid.UUID `json:"id"`
	Name             string    `json:"name"`
	RecommendedLevel int       `json:"recommended_level"`
	Difficulty       string    `json:"difficulty"`
	XPPerHour        int64     `json:"xp_per_hour"`
	GoldPerHour      int64     `json:"gold_per_hour"`
	Available        bool      `json:"available"`
}

// ListHuntsResponse envolve os itens e o cursor para a próxima página.
type ListHuntsResponse struct {
	Items      []HuntResponse `json:"items"`
	NextCursor *string        `json:"next_cursor"`
	Total      int            `json:"total"`
}

// ActiveSessionResponse é o estado atual da sessão de hunt em andamento.
type ActiveSessionResponse struct {
	SessionID              uuid.UUID `json:"session_id"`
	HuntName               string    `json:"hunt_name"`
	Status                 string    `json:"status"`
	StartedAt              time.Time `json:"started_at"`
	EstimatedEndAt         time.Time `json:"estimated_end_at"`
	ElapsedMinutes         int       `json:"elapsed_minutes"`
	ConfiguredDurationMins int       `json:"configured_duration_minutes"`
	XPGained               int64     `json:"xp_gained_so_far"`
	GoldGained             int64     `json:"gold_gained_so_far"`
}
