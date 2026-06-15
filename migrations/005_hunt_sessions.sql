-- +goose Up
CREATE TYPE IF NOT EXISTS hunt_session_status AS ENUM ('running', 'pending_claim', 'cancelled');
CREATE TYPE IF NOT EXISTS hunt_ended_by AS ENUM ('completed', 'player_stopped', 'death');

CREATE TABLE IF NOT EXISTS hunt_sessions (
    id                  UUID               PRIMARY KEY DEFAULT gen_random_uuid(),
    character_id        UUID               NOT NULL,
    hunt_id             UUID               NOT NULL REFERENCES hunts(id),
    status              hunt_session_status NOT NULL DEFAULT 'running',
    ended_by            hunt_ended_by,
    started_at          TIMESTAMPTZ        NOT NULL DEFAULT now(),
    ended_at            TIMESTAMPTZ,
    configured_duration INT                NOT NULL,      -- em minutos
    last_resolved_at    TIMESTAMPTZ        NOT NULL DEFAULT now(),
    xp_gained           BIGINT             NOT NULL DEFAULT 0,
    gold_gained         BIGINT             NOT NULL DEFAULT 0,
    death_count         INT                NOT NULL DEFAULT 0,
    snapshot_equipment  JSONB              NOT NULL DEFAULT '{}',
    snapshot_skills     JSONB              NOT NULL DEFAULT '{}',
    snapshot_level      INT                NOT NULL DEFAULT 1,
    snapshot_vocation   STRING             NOT NULL DEFAULT 'knight'
);

-- Índice principal para o worker: busca sessões running ordenadas por last_resolved_at
CREATE INDEX IF NOT EXISTS idx_hunt_sessions_running ON hunt_sessions(last_resolved_at ASC)
    WHERE status = 'running';

-- Índice para buscar sessão ativa de um personagem
CREATE INDEX IF NOT EXISTS idx_hunt_sessions_character ON hunt_sessions(character_id)
    WHERE status = 'running';

CREATE TABLE IF NOT EXISTS hunt_session_kills (
    session_id UUID NOT NULL REFERENCES hunt_sessions(id) ON DELETE CASCADE,
    monster_id UUID NOT NULL REFERENCES monsters(id),
    kill_count  INT  NOT NULL DEFAULT 0,
    PRIMARY KEY (session_id, monster_id)
);

CREATE TABLE IF NOT EXISTS hunt_session_loot (
    session_id  UUID   NOT NULL REFERENCES hunt_sessions(id) ON DELETE CASCADE,
    template_id UUID   NOT NULL REFERENCES item_templates(id),
    quantity    INT    NOT NULL DEFAULT 0,
    rarity      STRING NOT NULL DEFAULT 'common',
    PRIMARY KEY (session_id, template_id)
);

-- +goose Down
DROP TABLE IF EXISTS hunt_session_loot;
DROP TABLE IF EXISTS hunt_session_kills;
DROP TABLE IF EXISTS hunt_sessions;
DROP TYPE IF EXISTS hunt_ended_by;
DROP TYPE IF EXISTS hunt_session_status;
