# rpg-idle-progression-service

Microserviço responsável por sessões de hunt, simulação de combate idle e eventos de progressão de personagens.

## Stack
- Go 1.23, chi router, pgx/v5, franz-go (Kafka producer), slog (JSON), swaggo/swag
- Banco compartilhado: `rpg_idle` (CockroachDB) — **não criar banco separado**

## Migrations (goose)
| Arquivo | Versão | Owner |
|---------|--------|-------|
| 001_auth_schema.sql | 001 | auth-service |
| 002_character_schema.sql | 002 | character-service |
| 003_item_templates.sql | 003 | **progression-service** (temporário, migra para inventory-service) |
| 004_hunts_catalog.sql | 004 | **progression-service** |
| 005_hunt_sessions.sql | 005 | **progression-service** |
| 006_rename_tables_add_item_ids.sql | 006 | **progression-service** |

```bash
goose -dir ./migrations postgres "$DB_URL" up
```

## Variáveis de ambiente obrigatórias
- `SERVER_PORT` — porta HTTP (ex: 8082)
- `DB_URL` — CockroachDB (banco `rpg_idle`)
- `KAFKA_BROKERS` — brokers Kafka separados por vírgula
- `JWT_SECRET` — segredo JWT compartilhado com auth-service
- `LOG_LEVEL` — DEBUG | INFO | WARN | ERROR (padrão: INFO)
- `SWAGGER_ENABLED` — true | false (padrão: false)
- `HUNT_WORKER_INTERVAL` — intervalo do worker (padrão: 1m)
- `WORKER_MAX_CONCURRENT` — sessões processadas em paralelo pelo worker (padrão: 30)

## Kafka topics produzidos
Todos os eventos são publicados no topic único `progression-events`.

| Evento | Trigger |
|--------|---------|
| `HuntSessionStarted` | Jogador inicia uma hunt |
| `HuntTickResolved` | Worker resolve um tick incremental (a cada ~1 min por sessão) |
| `HuntSessionResolved` | Hunt encerrada por qualquer motivo (completed, player_stopped, death) |
| `DeathOccurred` | Simulador detecta morte — inclui penalidades calculadas |

**`HuntTickResolved`** é consumido pelo character-service para acumular XP de skills de forma incremental (sem esperar a sessão encerrar). Contém `physical_hits` e `mana_consumed` do intervalo.

## Rotas HTTP
```
GET  /health
GET  /swagger/*                              (SWAGGER_ENABLED=true)
GET  /api/v1/hunts                           → listar hunts com cursor pagination + campo available
POST /api/v1/hunts/:hunt_id/start           → iniciar hunt (body: {duration_minutes})
GET  /api/v1/hunts/current                  → sessão ativa do personagem
POST /api/v1/hunts/current/stop             → parar hunt em andamento
GET  /api/v1/hunts/sessions/:session_id     → resultado completo de sessão encerrada
```

Todas as rotas `/api/v1/*` requerem `Authorization: Bearer <JWT>`.
O `character_id` é extraído do JWT — **não passar no body**.

## Snapshot da build
O snapshot do personagem (level, vocação, skills, equipment) é **capturado pelo servidor** no momento de `StartHunt`. O cliente envia apenas `duration_minutes`. Isso impede que o cliente envie dados falsos para inflar resultados.

O `GetCharacterSnapshot` valida que o personagem está `idle` e lê das tabelas `characters`, `character_skills` e `character_equipment` (banco compartilhado).

## Hunt Worker
- Tick de 1 minuto (configurável via `HUNT_WORKER_INTERVAL`)
- Fan-out com semaphore: máximo de `WORKER_MAX_CONCURRENT` sessões concorrentes (padrão: 30)
- Cursor-based pagination por `last_resolved_at` (sem OFFSET)
- `CombatSimulator.Resolve()` é função pura — sem IO, testável unitariamente

## Paginação
`GET /api/v1/hunts` usa cursor-based pagination (keyset) por `(recommended_level, id)`.
Este é o padrão de paginação de todo o projeto para endpoints de listagem — sem OFFSET/LIMIT estilo página. O cursor é opaco (base64 JSON), retornado em `next_cursor`.

## Padrões de código
- `dbErr(op, err)` no repository: loga o erro real antes de retornar `ErrInternal`
- `writeErr(w, err)` no handler: loga `slog.Error` para status >= 500
- LOG_LEVEL via `slog.Level.UnmarshalText` no config.go

## Gerar Swagger
```bash
~/go/bin/swag init -g cmd/server/main.go
```

## Run local
```bash
cp .env.example .env
# editar .env com suas credenciais
go run ./cmd/server
```
