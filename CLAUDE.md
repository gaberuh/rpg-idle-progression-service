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

```bash
goose -dir ./migrations postgres "$DB_URL" up
```

## Variáveis de ambiente obrigatórias
- `SERVER_PORT` — porta HTTP (ex: 8082)
- `DB_URL` — CockroachDB (banco `rpg_idle`)
- `REDIS_CLUSTER_NODES` — nodes Redis separados por vírgula
- `KAFKA_BROKERS` — brokers Kafka separados por vírgula
- `JWT_SECRET` — segredo JWT compartilhado com auth-service
- `LOG_LEVEL` — DEBUG | INFO | WARN | ERROR (padrão: INFO)

## Kafka topics produzidos
| Topic | Evento |
|-------|--------|
| `hunt.session.resolved` | HuntSessionResolved — character-service consome para XP/gold/skills |
| `hunt.death.occurred` | DeathOccurred — character-service consome para penalidade de morte |
| `hunt.session.completed` | HuntSessionCompleted — character-service consome para retornar personagem a idle |

## Rotas HTTP
```
GET  /health
GET  /swagger/*          (SWAGGER_ENABLED=true)
GET  /api/v1/hunts       → listar hunts do catálogo
GET  /api/v1/hunts/active → sessão ativa do personagem
POST /api/v1/hunts/start  → iniciar hunt (body: StartHuntRequest)
POST /api/v1/hunts/stop   → parar hunt em andamento
```

Todas as rotas `/api/v1/*` requerem `Authorization: Bearer <JWT>`.
O `player_id` é extraído do JWT — **não passar no body**.

## Hunt Worker
- Tick de 1 minuto (configurável via `HUNT_WORKER_INTERVAL`)
- Fan-out com semaphore: máximo de `WORKER_MAX_CONCURRENT` sessões concorrentes (padrão: 30)
- Cursor-based pagination por `last_resolved_at` (sem OFFSET)
- `CombatSimulator.Resolve()` é função pura — sem IO, testável unitariamente

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
