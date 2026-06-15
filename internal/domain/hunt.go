package domain

import (
	"time"

	"github.com/google/uuid"
)

type Difficulty string

const (
	DifficultyEasy    Difficulty = "easy"
	DifficultyMedium  Difficulty = "medium"
	DifficultyHard    Difficulty = "hard"
	DifficultyExtreme Difficulty = "extreme"
)

type SessionStatus string

const (
	SessionRunning      SessionStatus = "running"
	SessionPendingClaim SessionStatus = "pending_claim"
	SessionCancelled    SessionStatus = "cancelled"
)

type EndedBy string

const (
	EndedByCompleted    EndedBy = "completed"
	EndedByPlayerStopped EndedBy = "player_stopped"
	EndedByDeath        EndedBy = "death"
)

type Vocation string

const (
	VocationKnight   Vocation = "knight"
	VocationPaladin  Vocation = "paladin"
	VocationSorcerer Vocation = "sorcerer"
	VocationDruid    Vocation = "druid"
)

type Hunt struct {
	ID               uuid.UUID  `db:"id"`
	Name             string     `db:"name"`
	RecommendedLevel int        `db:"recommended_level"`
	Difficulty       Difficulty `db:"difficulty"`
	XPPerHour        int64      `db:"xp_per_hour"`
	GoldPerHour      int64      `db:"gold_per_hour"`
	MortalityRate    float64    `db:"mortality_rate"`
}

type Monster struct {
	ID           uuid.UUID `db:"id"`
	Name         string    `db:"name"`
	XPReward     int       `db:"xp_reward"`
	GoldRewardMin int      `db:"gold_reward_min"`
	GoldRewardMax int      `db:"gold_reward_max"`
	LootTable    []LootEntry
	Resistances  map[string]int
}

type LootEntry struct {
	TemplateID   uuid.UUID `json:"template_id"`
	DropChance   float64   `json:"drop_chance"`
	QuantityMin  int       `json:"quantity_min"`
	QuantityMax  int       `json:"quantity_max"`
	Rarity       string    `json:"rarity"`
}

type HuntMonster struct {
	HuntID    uuid.UUID `db:"hunt_id"`
	MonsterID uuid.UUID `db:"monster_id"`
	SpawnRate float64   `db:"spawn_rate"`
}

type SnapshotSkills map[string]SnapshotSkill

type SnapshotSkill struct {
	Level int `json:"level"`
}

type SnapshotEquipment map[string]*SnapshotItem

type SnapshotItem struct {
	ItemID  string `json:"item_id"`
	Name    string `json:"name"`
	Attack  int    `json:"attack,omitempty"`
	Defense int    `json:"defense,omitempty"`
	Armor   int    `json:"armor,omitempty"`
}

type HuntSession struct {
	ID                 uuid.UUID      `db:"id"`
	CharacterID        uuid.UUID      `db:"character_id"`
	HuntID             uuid.UUID      `db:"hunt_id"`
	Status             SessionStatus  `db:"status"`
	EndedBy            *EndedBy       `db:"ended_by"`
	StartedAt          time.Time      `db:"started_at"`
	EndedAt            *time.Time     `db:"ended_at"`
	ConfiguredDuration int            `db:"configured_duration"`
	LastResolvedAt     time.Time      `db:"last_resolved_at"`
	XPGained           int64          `db:"xp_gained"`
	GoldGained         int64          `db:"gold_gained"`
	DeathCount         int            `db:"death_count"`
	SnapshotEquipment  SnapshotEquipment `db:"snapshot_equipment"`
	SnapshotSkills     SnapshotSkills    `db:"snapshot_skills"`
	SnapshotLevel      int            `db:"snapshot_level"`
	SnapshotVocation   Vocation       `db:"snapshot_vocation"`
}

type KillCount struct {
	SessionID uuid.UUID `db:"session_id"`
	MonsterID uuid.UUID `db:"monster_id"`
	KillCount int       `db:"kill_count"`
}

type SessionLoot struct {
	SessionID  uuid.UUID   `db:"session_id"`
	TemplateID uuid.UUID   `db:"template_id"`
	Quantity   int         `db:"quantity"`
	ItemIDs    []uuid.UUID `db:"item_ids"`
	Rarity     string      `db:"rarity"`
}

// SessionKillCount é usado no resultado da sessão (join com monsters).
type SessionKillCount struct {
	MonsterName string
	KillCount   int
}

// SessionLootEntry é usado no resultado da sessão (join com item_templates).
type SessionLootEntry struct {
	ItemName string
	Rarity   string
	Quantity int
	ItemIDs  []uuid.UUID // não-nulo apenas para itens únicos
}
