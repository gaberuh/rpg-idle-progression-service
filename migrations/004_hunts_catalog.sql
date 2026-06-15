-- +goose Up
CREATE TYPE IF NOT EXISTS hunt_difficulty AS ENUM ('easy', 'medium', 'hard', 'extreme');

CREATE TABLE IF NOT EXISTS hunts (
    id                UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    name              STRING      NOT NULL UNIQUE,
    recommended_level INT         NOT NULL DEFAULT 1,
    difficulty        hunt_difficulty NOT NULL DEFAULT 'easy',
    xp_per_hour       BIGINT      NOT NULL DEFAULT 0,
    gold_per_hour     BIGINT      NOT NULL DEFAULT 0,
    mortality_rate    DECIMAL(5, 4) NOT NULL DEFAULT 0.0,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS monsters (
    id              UUID    PRIMARY KEY DEFAULT gen_random_uuid(),
    name            STRING  NOT NULL UNIQUE,
    xp_reward       INT     NOT NULL DEFAULT 0,
    gold_reward_min INT     NOT NULL DEFAULT 0,
    gold_reward_max INT     NOT NULL DEFAULT 0,
    resistances     JSONB   NOT NULL DEFAULT '{}',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS hunt_monsters (
    hunt_id    UUID       NOT NULL REFERENCES hunts(id) ON DELETE CASCADE,
    monster_id UUID       NOT NULL REFERENCES monsters(id) ON DELETE CASCADE,
    spawn_rate DECIMAL(5, 4) NOT NULL DEFAULT 1.0,
    PRIMARY KEY (hunt_id, monster_id)
);

CREATE TABLE IF NOT EXISTS monster_loot (
    id          UUID    PRIMARY KEY DEFAULT gen_random_uuid(),
    monster_id  UUID    NOT NULL REFERENCES monsters(id) ON DELETE CASCADE,
    template_id UUID    NOT NULL REFERENCES item_templates(id) ON DELETE CASCADE,
    drop_chance DECIMAL(5, 4) NOT NULL,
    quantity_min INT    NOT NULL DEFAULT 1,
    quantity_max INT    NOT NULL DEFAULT 1,
    rarity      STRING  NOT NULL DEFAULT 'common'
);

CREATE INDEX IF NOT EXISTS idx_monster_loot_monster ON monster_loot(monster_id);

-- Seed: algumas hunts de exemplo para dev
INSERT INTO hunts (name, recommended_level, difficulty, xp_per_hour, gold_per_hour, mortality_rate)
VALUES
    ('Cyclops Plains',   20, 'easy',    50000,  3000,  0.01),
    ('Dwarf Mines',      30, 'easy',    80000,  5000,  0.02),
    ('Hellgate',         60, 'medium', 200000, 12000,  0.05),
    ('Demon Oak Quest', 100, 'hard',   500000, 30000,  0.15),
    ('Roshamuul',       200, 'extreme',1200000,80000,  0.30)
ON CONFLICT (name) DO NOTHING;

-- +goose Down
DROP TABLE IF EXISTS monster_loot;
DROP TABLE IF EXISTS hunt_monsters;
DROP TABLE IF EXISTS monsters;
DROP TABLE IF EXISTS hunts;
DROP TYPE IF EXISTS hunt_difficulty;
