package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	httpswagger "github.com/swaggo/http-swagger"

	_ "github.com/gaberuh/rpg-idle-progression-service/docs"
	"github.com/gaberuh/rpg-idle-progression-service/internal/config"
	"github.com/gaberuh/rpg-idle-progression-service/internal/domain"
	"github.com/gaberuh/rpg-idle-progression-service/internal/event"
	"github.com/gaberuh/rpg-idle-progression-service/internal/handler"
	"github.com/gaberuh/rpg-idle-progression-service/internal/middleware"
	"github.com/gaberuh/rpg-idle-progression-service/internal/repository/cockroach"
	"github.com/gaberuh/rpg-idle-progression-service/internal/service"
	"github.com/gaberuh/rpg-idle-progression-service/internal/worker"
)

// @title           RPG Idle — Progression Service
// @version         1.0
// @description     Gerencia sessões de hunt, simulação de combate e eventos de progressão.
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
func main() {
	_ = godotenv.Load()

	cfg, err := config.Load()
	if err != nil {
		slog.Error("config load failed", "err", err)
		os.Exit(1)
	}

	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: cfg.Log.Level,
	})))

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	// DB
	poolCfg, err := pgxpool.ParseConfig(cfg.DB.URL)
	if err != nil {
		slog.Error("pgxpool config failed", "err", err)
		os.Exit(1)
	}
	poolCfg.MaxConns = cfg.DB.MaxConns

	db, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		slog.Error("pgxpool connect failed", "err", err)
		os.Exit(1)
	}
	defer db.Close()

	// Kafka producer
	producer, err := event.NewHuntProducer(cfg.Kafka.Brokers)
	if err != nil {
		slog.Error("kafka producer failed", "err", err)
		os.Exit(1)
	}

	// Wiring
	repo := cockroach.NewHuntRepository(db)
	simulator := domain.NewCombatSimulator()
	svc := service.NewHuntService(repo, producer)
	huntHandler := handler.NewHuntHandler(svc)

	// Hunt worker
	hw := worker.NewHuntWorker(repo, simulator, producer, cfg.Worker.HuntInterval, cfg.Worker.MaxConcurrent)
	go hw.Run(ctx)

	// Router
	r := chi.NewRouter()
	r.Use(chimiddleware.RequestID)
	r.Use(chimiddleware.RealIP)
	r.Use(chimiddleware.Recoverer)

	r.Get("/health", handler.HealthHandler)

	if cfg.Swagger.Enabled {
		r.Get("/swagger/*", httpswagger.WrapHandler)
	}

	r.Route("/api/v1/hunts", func(r chi.Router) {
		r.Use(middleware.Auth(cfg.JWT.Secret))
		r.Get("/", huntHandler.ListHunts)
		r.Get("/active", huntHandler.GetActiveSession)
		r.Post("/start", huntHandler.StartHunt)
		r.Post("/stop", huntHandler.StopHunt)
	})

	srv := &http.Server{
		Addr:         ":" + cfg.Server.Port,
		Handler:      r,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	go func() {
		slog.Info("server started", "port", cfg.Server.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "err", err)
		}
	}()

	<-ctx.Done()
	slog.Info("shutting down...")

	shutCtx, shutCancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer shutCancel()
	_ = srv.Shutdown(shutCtx)
}
