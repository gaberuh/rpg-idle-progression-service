package worker

import (
	"context"
	"log/slog"
	"time"

	"golang.org/x/sync/errgroup"
	"golang.org/x/sync/semaphore"
	apperr "github.com/gaberuh/rpg-idle-progression-service/internal/errors"
	"github.com/gaberuh/rpg-idle-progression-service/internal/domain"
	"github.com/gaberuh/rpg-idle-progression-service/internal/event"
	"github.com/gaberuh/rpg-idle-progression-service/internal/event/dto"
	"github.com/gaberuh/rpg-idle-progression-service/internal/repository"
)

const sessionBatchSize = 100

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
	// Cursor começa no início do tempo para pegar todas as sessões running.
	// O campo last_resolved_at > cursor garante que cada sessão seja processada pelo menos uma vez por tick.
	// Usamos um cursor fixo (epoch) por tick — todas as sessões com any last_resolved_at são elegíveis.
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
			s := s // capture
			if err := w.sem.Acquire(gctx, 1); err != nil {
				// ctx cancelado
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

	// Persiste progresso incremental
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

	// Publica HuntSessionResolved para character-service acumular XP/skills
	kills := make(map[string]int, len(result.KillCounts))
	for k, v := range result.KillCounts {
		kills[k.String()] = v
	}
	_ = w.producer.PublishHuntResolved(ctx, dto.HuntSessionResolved{
		SessionID:    session.ID,
		CharacterID:  session.CharacterID,
		HuntID:       session.HuntID,
		XPGained:     result.XP,
		GoldGained:   result.Gold,
		PhysicalHits: result.PhysicalHits,
		ManaConsumed: result.ManaConsumed,
		Kills:        kills,
		ResolvedAt:   now,
	})

	if result.Died {
		_ = w.producer.PublishDeathOccurred(ctx, dto.DeathOccurred{
			SessionID:   session.ID,
			CharacterID: session.CharacterID,
			HuntID:      session.HuntID,
			DiedAt:      session.LastResolvedAt.Add(result.TimeUntilDeath),
		})

		// Morte encerra a sessão
		endedBy := domain.EndedByDeath
		if err := w.repo.EndSession(ctx, session.ID, endedBy, domain.SessionPendingClaim, now); err != nil {
			slog.Error("worker: EndSession (death) failed", "session_id", session.ID, "err", err)
			return
		}

		_ = w.producer.PublishHuntCompleted(ctx, dto.HuntSessionCompleted{
			SessionID:   session.ID,
			CharacterID: session.CharacterID,
			EndedBy:     string(endedBy),
			CompletedAt: now,
		})
		return
	}

	// Verifica se a sessão completou o tempo configurado
	if result.Completed {
		endedBy := domain.EndedByCompleted
		if err := w.repo.EndSession(ctx, session.ID, endedBy, domain.SessionPendingClaim, now); err != nil {
			slog.Error("worker: EndSession (completed) failed", "session_id", session.ID, "err", err)
			return
		}

		_ = w.producer.PublishHuntCompleted(ctx, dto.HuntSessionCompleted{
			SessionID:   session.ID,
			CharacterID: session.CharacterID,
			EndedBy:     string(endedBy),
			CompletedAt: now,
		})
	}

	_ = apperr.ErrInternal // satisfaz lint de import não usado
}
