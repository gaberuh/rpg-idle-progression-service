package cockroach

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	apperr "github.com/gaberuh/rpg-idle-progression-service/internal/errors"
	"github.com/gaberuh/rpg-idle-progression-service/internal/domain"
	"github.com/gaberuh/rpg-idle-progression-service/internal/repository"
)

type huntRepository struct {
	db *pgxpool.Pool
}

func NewHuntRepository(db *pgxpool.Pool) repository.HuntRepository {
	return &huntRepository{db: db}
}

func dbErr(op string, err error) error {
	slog.Error("db error", "op", op, "err", err)
	return apperr.ErrInternal
}

// GetHuntByID busca hunt do catálogo.
func (r *huntRepository) GetHuntByID(ctx context.Context, id uuid.UUID) (*domain.Hunt, error) {
	const q = `
		SELECT id, name, recommended_level, difficulty, xp_per_hour, gold_per_hour, mortality_rate
		FROM hunts WHERE id = $1`

	row := r.db.QueryRow(ctx, q, id)
	h, err := scanHunt(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, apperr.ErrHuntNotFound
	}
	if err != nil {
		return nil, dbErr("GetHuntByID", err)
	}
	return h, nil
}

// ListHunts retorna hunts com keyset pagination por (recommended_level, id).
// cursor nil = primeira página. limit máximo respeitado pelo caller (service).
func (r *huntRepository) ListHunts(ctx context.Context, cursor *repository.HuntCursor, limit int) ([]domain.Hunt, error) {
	var (
		rows pgx.Rows
		err  error
	)

	if cursor == nil {
		const q = `
			SELECT id, name, recommended_level, difficulty, xp_per_hour, gold_per_hour, mortality_rate
			FROM hunts
			ORDER BY recommended_level ASC, id ASC
			LIMIT $1`
		rows, err = r.db.Query(ctx, q, limit)
	} else {
		// Keyset: pega a próxima página depois do cursor (level, id)
		const q = `
			SELECT id, name, recommended_level, difficulty, xp_per_hour, gold_per_hour, mortality_rate
			FROM hunts
			WHERE (recommended_level, id) > ($1, $2)
			ORDER BY recommended_level ASC, id ASC
			LIMIT $3`
		rows, err = r.db.Query(ctx, q, cursor.RecommendedLevel, cursor.ID, limit)
	}

	if err != nil {
		return nil, dbErr("ListHunts", err)
	}
	defer rows.Close()

	var hunts []domain.Hunt
	for rows.Next() {
		h, err := scanHunt(rows)
		if err != nil {
			return nil, dbErr("ListHunts.scan", err)
		}
		hunts = append(hunts, *h)
	}
	return hunts, nil
}

// GetHuntMonsters retorna os monstros de uma hunt com seu spawn rate e loot table.
func (r *huntRepository) GetHuntMonsters(ctx context.Context, huntID uuid.UUID) ([]domain.MonsterWithSpawnRate, error) {
	const q = `
		SELECT
			m.id, m.name, m.xp_reward, m.gold_reward_min, m.gold_reward_max,
			m.resistances,
			COALESCE(
				json_agg(json_build_object(
					'template_id', ml.template_id,
					'drop_chance', ml.drop_chance,
					'quantity_min', ml.quantity_min,
					'quantity_max', ml.quantity_max,
					'rarity', ml.rarity
				)) FILTER (WHERE ml.template_id IS NOT NULL),
				'[]'
			) AS loot_table,
			hm.spawn_rate
		FROM hunt_monsters hm
		JOIN monsters m ON m.id = hm.monster_id
		LEFT JOIN monster_loot ml ON ml.monster_id = m.id
		WHERE hm.hunt_id = $1
		GROUP BY m.id, hm.spawn_rate`

	rows, err := r.db.Query(ctx, q, huntID)
	if err != nil {
		return nil, dbErr("GetHuntMonsters", err)
	}
	defer rows.Close()

	var result []domain.MonsterWithSpawnRate
	for rows.Next() {
		var m domain.Monster
		var lootJSON []byte
		var resistJSON []byte
		var spawnRate float64

		if err := rows.Scan(
			&m.ID, &m.Name, &m.XPReward, &m.GoldRewardMin, &m.GoldRewardMax,
			&resistJSON, &lootJSON, &spawnRate,
		); err != nil {
			return nil, dbErr("GetHuntMonsters.scan", err)
		}

		if err := json.Unmarshal(lootJSON, &m.LootTable); err != nil {
			return nil, dbErr("GetHuntMonsters.lootJSON", err)
		}
		if err := json.Unmarshal(resistJSON, &m.Resistances); err != nil {
			return nil, dbErr("GetHuntMonsters.resistJSON", err)
		}

		result = append(result, domain.MonsterWithSpawnRate{Monster: m, SpawnRate: spawnRate})
	}
	return result, nil
}

// CreateSession insere nova sessão de hunt.
func (r *huntRepository) CreateSession(ctx context.Context, s domain.HuntSession) error {
	eqJSON, _ := json.Marshal(s.SnapshotEquipment)
	skJSON, _ := json.Marshal(s.SnapshotSkills)

	const q = `
		INSERT INTO hunt_sessions (
			id, character_id, hunt_id, status, started_at,
			configured_duration, last_resolved_at,
			snapshot_equipment, snapshot_skills, snapshot_level, snapshot_vocation
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`

	_, err := r.db.Exec(ctx, q,
		s.ID, s.CharacterID, s.HuntID, s.Status, s.StartedAt,
		s.ConfiguredDuration, s.LastResolvedAt,
		eqJSON, skJSON, s.SnapshotLevel, s.SnapshotVocation,
	)
	if err != nil {
		return dbErr("CreateSession", err)
	}
	return nil
}

// GetActiveSession retorna a sessão running de um personagem, se existir.
func (r *huntRepository) GetActiveSession(ctx context.Context, characterID uuid.UUID) (*domain.HuntSession, error) {
	const q = `
		SELECT id, character_id, hunt_id, status, ended_by, started_at, ended_at,
		       configured_duration, last_resolved_at, xp_gained, gold_gained, death_count,
		       snapshot_equipment, snapshot_skills, snapshot_level, snapshot_vocation
		FROM hunt_sessions
		WHERE character_id = $1 AND status = 'running'
		LIMIT 1`

	row := r.db.QueryRow(ctx, q, characterID)
	s, err := scanSession(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, apperr.ErrNoActiveHunt
	}
	if err != nil {
		return nil, dbErr("GetActiveSession", err)
	}
	return s, nil
}

// GetSessionByID busca sessão pelo ID.
func (r *huntRepository) GetSessionByID(ctx context.Context, id uuid.UUID) (*domain.HuntSession, error) {
	const q = `
		SELECT id, character_id, hunt_id, status, ended_by, started_at, ended_at,
		       configured_duration, last_resolved_at, xp_gained, gold_gained, death_count,
		       snapshot_equipment, snapshot_skills, snapshot_level, snapshot_vocation
		FROM hunt_sessions WHERE id = $1`

	row := r.db.QueryRow(ctx, q, id)
	s, err := scanSession(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, apperr.ErrSessionNotFound
	}
	if err != nil {
		return nil, dbErr("GetSessionByID", err)
	}
	return s, nil
}

// UpdateSessionProgress acumula XP, gold e mortes na sessão.
func (r *huntRepository) UpdateSessionProgress(ctx context.Context, id uuid.UUID, xpGained, goldGained int64, deaths int, resolvedAt time.Time) error {
	const q = `
		UPDATE hunt_sessions
		SET xp_gained        = xp_gained + $2,
		    gold_gained      = gold_gained + $3,
		    death_count      = death_count + $4,
		    last_resolved_at = $5
		WHERE id = $1`

	if _, err := r.db.Exec(ctx, q, id, xpGained, goldGained, deaths, resolvedAt); err != nil {
		return dbErr("UpdateSessionProgress", err)
	}
	return nil
}

// EndSession fecha a sessão de hunt.
func (r *huntRepository) EndSession(ctx context.Context, id uuid.UUID, endedBy domain.EndedBy, status domain.SessionStatus, endedAt time.Time) error {
	const q = `
		UPDATE hunt_sessions
		SET status = $2, ended_by = $3, ended_at = $4
		WHERE id = $1`

	if _, err := r.db.Exec(ctx, q, id, status, endedBy, endedAt); err != nil {
		return dbErr("EndSession", err)
	}
	return nil
}

// UpsertKillCounts incrementa contagem de kills por monstro na sessão.
func (r *huntRepository) UpsertKillCounts(ctx context.Context, sessionID uuid.UUID, kills map[uuid.UUID]int) error {
	for monsterID, count := range kills {
		const q = `
			INSERT INTO hunt_session_kills (session_id, monster_id, kill_count)
			VALUES ($1, $2, $3)
			ON CONFLICT (session_id, monster_id)
			DO UPDATE SET kill_count = hunt_session_kills.kill_count + EXCLUDED.kill_count`

		if _, err := r.db.Exec(ctx, q, sessionID, monsterID, count); err != nil {
			return dbErr("UpsertKillCounts", err)
		}
	}
	return nil
}

// UpsertSessionLoot acumula loot dropado na sessão.
func (r *huntRepository) UpsertSessionLoot(ctx context.Context, sessionID uuid.UUID, loot map[uuid.UUID]domain.LootDrop) error {
	for templateID, drop := range loot {
		const q = `
			INSERT INTO hunt_session_loot (session_id, template_id, quantity, rarity)
			VALUES ($1, $2, $3, $4)
			ON CONFLICT (session_id, template_id)
			DO UPDATE SET quantity = hunt_session_loot.quantity + EXCLUDED.quantity`

		if _, err := r.db.Exec(ctx, q, sessionID, templateID, drop.Quantity, drop.Rarity); err != nil {
			return dbErr("UpsertSessionLoot", err)
		}
	}
	return nil
}

// ListRunningSessions retorna sessões running com cursor-based pagination por last_resolved_at.
func (r *huntRepository) ListRunningSessions(ctx context.Context, after time.Time, limit int) ([]domain.HuntSession, error) {
	const q = `
		SELECT id, character_id, hunt_id, status, ended_by, started_at, ended_at,
		       configured_duration, last_resolved_at, xp_gained, gold_gained, death_count,
		       snapshot_equipment, snapshot_skills, snapshot_level, snapshot_vocation
		FROM hunt_sessions
		WHERE status = 'running' AND last_resolved_at > $1
		ORDER BY last_resolved_at ASC
		LIMIT $2`

	rows, err := r.db.Query(ctx, q, after, limit)
	if err != nil {
		return nil, dbErr("ListRunningSessions", err)
	}
	defer rows.Close()

	var sessions []domain.HuntSession
	for rows.Next() {
		s, err := scanSession(rows)
		if err != nil {
			return nil, dbErr("ListRunningSessions.scan", err)
		}
		sessions = append(sessions, *s)
	}
	return sessions, nil
}

// scanner comum para pgx.Row e pgx.Rows
type scanner interface {
	Scan(dest ...any) error
}

func scanHunt(row scanner) (*domain.Hunt, error) {
	var h domain.Hunt
	err := row.Scan(
		&h.ID, &h.Name, &h.RecommendedLevel, &h.Difficulty,
		&h.XPPerHour, &h.GoldPerHour, &h.MortalityRate,
	)
	return &h, err
}

func scanSession(row scanner) (*domain.HuntSession, error) {
	var s domain.HuntSession
	var eqJSON, skJSON []byte
	err := row.Scan(
		&s.ID, &s.CharacterID, &s.HuntID, &s.Status, &s.EndedBy,
		&s.StartedAt, &s.EndedAt, &s.ConfiguredDuration, &s.LastResolvedAt,
		&s.XPGained, &s.GoldGained, &s.DeathCount,
		&eqJSON, &skJSON, &s.SnapshotLevel, &s.SnapshotVocation,
	)
	if err != nil {
		return nil, err
	}
	_ = json.Unmarshal(eqJSON, &s.SnapshotEquipment)
	_ = json.Unmarshal(skJSON, &s.SnapshotSkills)
	return &s, nil
}
