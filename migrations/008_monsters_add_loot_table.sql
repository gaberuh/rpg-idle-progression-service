-- +goose Up
-- monsters estava faltando loot_table definido no PRD-03
ALTER TABLE monsters
    ADD COLUMN IF NOT EXISTS loot_table JSONB NOT NULL DEFAULT '[]';

-- +goose Down
ALTER TABLE monsters
    DROP COLUMN IF EXISTS loot_table;
