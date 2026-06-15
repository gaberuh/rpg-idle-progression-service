-- +goose Up

-- Renomeia hunt_session_kills → hunt_kill_counts (alinha com TSAD/PRD-03)
ALTER TABLE hunt_session_kills RENAME TO hunt_kill_counts;

-- Renomeia hunt_session_loot → session_loot (alinha com TSAD/PRD-03)
ALTER TABLE hunt_session_loot RENAME TO session_loot;

-- Adiciona item_ids UUID[] para rastrear itens únicos dropados desde o drop
-- Preenchido apenas quando item_templates.stackable = false
ALTER TABLE session_loot ADD COLUMN IF NOT EXISTS item_ids UUID[] NULL;

-- +goose Down
ALTER TABLE session_loot DROP COLUMN IF EXISTS item_ids;
ALTER TABLE session_loot RENAME TO hunt_session_loot;
ALTER TABLE hunt_kill_counts RENAME TO hunt_session_kills;
