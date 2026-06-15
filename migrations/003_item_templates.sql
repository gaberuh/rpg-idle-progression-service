-- +goose Up
-- item_templates: catálogo compartilhado de itens (owned by inventory-service)
-- Criado aqui temporariamente pois progression-service precisa referenciar template_id no loot.
-- Quando inventory-service for implementado, este arquivo será migrado para lá.

CREATE TABLE IF NOT EXISTS item_templates (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name        STRING NOT NULL,
    description STRING,
    weight      DECIMAL(8, 2) NOT NULL DEFAULT 0,
    value       INT NOT NULL DEFAULT 0,
    rarity      STRING NOT NULL DEFAULT 'common',
    item_type   STRING NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_item_templates_rarity ON item_templates(rarity);
CREATE INDEX IF NOT EXISTS idx_item_templates_type ON item_templates(item_type);

-- +goose Down
DROP TABLE IF EXISTS item_templates;
