package worker

import (
	"context"
	"log/slog"
	"time"

	"golang.org/x/sync/errgroup"
	"golang.org/x/sync/semaphore"
	"github.com/gaberuh/rpg-idle-progression-service/internal/domain"
	"github.com/gaberuh/rpg-idle-progression-service/internal/event"
	"github.com/gaberuh/rpg-idle-progression-service/internal/event/dto"
	"github.com/gaberuh/rpg-idle-progression-service/internal/repository"
)

const sessionBatchSize = 1000

type HuntWorker struct {
	repo      repository.HuntRepository
	simulator domain.CombatSimulator
	producer  event.HuntProducer
	interval  time.Duration
	sem       *semaphore.Weighted
}

func NewHuntWorker(
	repo repository.HuntRepository,
	simulator domain.CombatSimulator,
	producer event.HuntProducer,
	interval time.Duration,
	maxConcurrent int64,
) *HuntWorker {
	return &HuntWorker{
		repo:      repo,
		simulator: simulator,
		producer:  producer,
		interval:  interval,
		sem:       semaphore.NewWeighted(maxConcurrent),
	}
}

// Run inicia o loop do worker. Bloqueia até ctx ser cancelado.
func (w *HuntWorker) Run(ctx context.Context) {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	slog.Info("hunt worker started", "interval", w.interval)

	for {
		select {
		case <-ctx.Done():
			slog.Info("hunt worker stopping")
			return
		case <-ticker.C:
			w.tick(ctx)
		}
	}
}

// tick processa TODAS as sessões running usando cursor-based pagination + fan-out.
func (w *HuntWorker) tick(ctx context.Context) {
	cursor := time.Time{}
	now := time.Now().UTC()

	for {
		sessions, err := w.repo.ListRunningSessions(ctx, cursor, sessionBatchSize)
		if err != nil {
			slog.Error("hunt worker: list sessions failed", "err", err)
			return
		}
		if len(sessions) == 0 {
			break
		}

		g, gctx := errgroup.WithContext(ctx)

		for _, s := range sessions {
			s := s
			if err := w.sem.Acquire(gctx, 1); err != nil {
				break
			}
			g.Go(func() error {
				defer w.sem.Release(1)
				w.resolveSession(gctx, s, now)
				return nil
			})
		}

		_ = g.Wait()

		if len(sessions) < sessionBatchSize {
			break
		}
		cursor = sessions[len(sessions)-1].LastResolvedAt
	}
}

// resolveSession simula o intervalo desde last_resolved_at até now.
func (w *HuntWorker) resolveSession(ctx context.Context, session domain.HuntSession, now time.Time) {
	delta := now.Sub(session.LastResolvedAt)
	if delta <= 0 {
		return
	}

	totalElapsed := now.Sub(session.StartedAt)

	hunt, err := w.repo.GetHuntByID(ctx, session.HuntID)
	if err != nil {
		slog.Error("worker: GetHuntByID failed", "session_id", session.ID, "err", err)
		return
	}

	monsters, err := w.repo.GetHuntMonsters(ctx, session.HuntID)
	if err != nil {
		slog.Error("worker: GetHuntMonsters failed", "session_id", session.ID, "err", err)
		return
	}

	result := w.simulator.Resolve(session, *hunt, monsters, delta, totalElapsed)

	deaths := 0
	if result.Died {
		deaths = 1
	}

	if err := w.repo.UpdateSessionProgress(ctx, session.ID, result.XP, result.Gold, deaths, now); err != nil {
		slog.Error("worker: UpdateSessionProgress failed", "session_id", session.ID, "err", err)
		return
	}

	if len(result.KillCounts) > 0 {
		if err := w.repo.UpsertKillCounts(ctx, session.ID, result.KillCounts); err != nil {
			slog.Error("worker: UpsertKillCounts failed", "session_id", session.ID, "err", err)
		}
	}

	if len(result.LootDrops) > 0 {
		if err := w.repo.UpsertSessionLoot(ctx, session.ID, result.LootDrops); err != nil {
			slog.Error("worker: UpsertSessionLoot failed", "session_id", session.ID, "err", err)
		}
	}

	// Publica tick incremental para o character-service acumular skill XP
	kills := make(map[string]int, len(result.KillCounts))
	for k, v := range result.KillCounts {
		kills[k.String()] = v
	}
	_ = w.producer.PublishHuntTickResolved(ctx, dto.HuntTickResolved{
		SessionID:    session.ID,
		CharacterID:  session.CharacterID,
		HuntID:       session.HuntID,
		XPGained:     result.XP,
		GoldGained:   result.Gold,
		PhysicalHits: result.PhysicalHits,
		ManaConsumed: result.ManaConsumed,
		Vocation:     string(session.SnapshotVocation),
		Kills:        kills,
		ResolvedAt:   now,
	})

	if result.Died {
		w.handleDeath(ctx, session, result, now)
		return
	}

	if result.Completed {
		w.handleSessionEnd(ctx, session, domain.EndedByCompleted, now)
	}
}

// handleDeath encerra a sessão por morte, calcula penalidades e publica os eventos.
func (w *HuntWorker) handleDeath(ctx context.Context, session domain.HuntSession, result domain.SimulationResult, now time.Time) {
	blessings, err := w.repo.GetCharacterBlessings(ctx, session.CharacterID)
	if err != nil {
		slog.Error("worker: GetCharacterBlessings failed", "session_id", session.ID, "err", err)
		blessings = 0
	}

	xpPenalty, skillPenalty := deathPenalties(blessings)

	diedAt := session.LastResolvedAt.Add(result.TimeUntilDeath)
	_ = w.producer.PublishDeathOccurred(ctx, dto.DeathOccurred{
		SessionID:           session.ID,
		CharacterID:         session.CharacterID,
		HuntID:              session.HuntID,
		BlessingsAtDeath:    blessings,
		XPPenaltyPercent:    xpPenalty,
		SkillPenaltyPercent: skillPenalty,
		DiedAt:              diedAt,
	})

	w.handleSessionEnd(ctx, session, domain.EndedByDeath, now)
}

// handleSessionEnd encerra a sessão no banco e publica HuntSessionResolved.
func (w *HuntWorker) handleSessionEnd(ctx context.Context, session domain.HuntSession, endedBy domain.EndedBy, now time.Time) {
	if err := w.repo.EndSession(ctx, session.ID, endedBy, domain.SessionPendingClaim, now); err != nil {
		slog.Error("worker: EndSession failed", "session_id", session.ID, "ended_by", endedBy, "err", err)
		return
	}

	// session.CharacterID armazena player_id — atualiza characters.status para pending_claim
	if err := w.repo.UpdateCharacterStatus(ctx, session.CharacterID, "pending_claim"); err != nil {
		slog.Error("worker: UpdateCharacterStatus failed", "session_id", session.ID, "err", err)
	}

	durationMinutes := int(now.Sub(session.StartedAt).Minutes())

	_ = w.producer.PublishHuntResolved(ctx, dto.HuntSessionResolved{
		SessionID:       session.ID,
		CharacterID:     session.CharacterID,
		HuntID:          session.HuntID,
		EndedBy:         string(endedBy),
		XPGained:        session.XPGained,
		GoldGained:      session.GoldGained,
		DurationMinutes: durationMinutes,
		Vocation:        string(session.SnapshotVocation),
		ResolvedAt:      now,
	})
}

// deathPenalties retorna os percentuais de penalidade de XP e skill baseado nas blessings.
// Tabela definida no TSAD: 0 blessings = 10%, 1 = 3%, 2 = 1%.
func deathPenalties(blessings int) (xpPercent, skillPercent int) {
	switch blessings {
	case 1:
		return 3, 3
	case 2:
		return 1, 1
	default:
		return 10, 10
	}
}
