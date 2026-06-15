-- +goose Up
-- item_templates estava faltando stackable e base_stats definidos no PRD-05
ALTER TABLE item_templates
    ADD COLUMN IF NOT EXISTS stackable  BOOLEAN NOT NULL DEFAULT false,
    ADD COLUMN IF NOT EXISTS base_stats JSONB   NOT NULL DEFAULT '{}';

-- +goose Down
ALTER TABLE item_templates
    DROP COLUMN IF EXISTS stackable,
    DROP COLUMN IF EXISTS base_stats;
