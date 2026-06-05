package main

import (
	"context"
	"fmt"
	"github.com/go-chi/chi/v5"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"github.com/pmkrish02/payflow-ai/internal/handler"
	"github.com/pmkrish02/payflow-ai/internal/middleware"
	"github.com/pmkrish02/payflow-ai/internal/repository"
	"github.com/pmkrish02/payflow-ai/internal/service"
	"github.com/pmkrish02/payflow-ai/internal/worker"
	"github.com/redis/go-redis/v9"
	"google.golang.org/genai"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	godotenv.Load("../../.env")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	pool, err := pgxpool.New(context.Background(), os.Getenv("POSTGRES_URL"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to connect to database: %v\n", err)
		os.Exit(1)
	}
	defer pool.Close()

	slog.Info("Connected to database")

	rdb := redis.NewClient(&redis.Options{
		Addr:     os.Getenv("REDIS_ADDR"),
		Password: os.Getenv("REDIS_PASSWORD"),
		DB:       0,
	})

	_, err = rdb.Ping(context.Background()).Result()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to connect to Redis: %v\n", err)
		os.Exit(1)
	}

	slog.Info("Connected to Redis")

	migrationsPath := os.Getenv("MIGRATIONS_PATH")
	if migrationsPath == "" {
		migrationsPath = "file://../../migrations"
	}

	m, err := migrate.New(
		migrationsPath,
		os.Getenv("POSTGRES_URL")+"?sslmode=disable",
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Migration failed to initialize: %v\n", err)
		os.Exit(1)
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		fmt.Fprintf(os.Stderr, "Migration failed: %v\n", err)
		os.Exit(1)
	}

	slog.Info("Migrations ran successfully")

	auditRepo := &repository.AuditRepository{DB: pool}
	wp := worker.NewWorkerPool(5, auditRepo)
	wp.Start(ctx)

	geminiClient, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  os.Getenv("GEMINI_API_KEY"),
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to create Gemini client: %v\n", err)
		os.Exit(1)
	}
	agentHandler := &handler.AgentHandler{DB: pool, GeminiClient: geminiClient}

	authRepo := &repository.AuthRepository{DB: pool}
	authService := &service.AuthService{AuthRepo: authRepo}
	authHandler := &handler.AuthHandler{AuthService: authService}

	accountRepo := &repository.AccountRepository{DB: pool, Redis: rdb}
	accountService := &service.AccountService{AccountRepo: accountRepo}
	accountHandler := &handler.AccountHandler{AccountService: accountService}

	transferRepo := &repository.TransferRepository{DB: pool, Redis: rdb}
	transferService := &service.TransferService{TransferRepo: transferRepo}
	transferHandler := &handler.TransferHandler{TransferService: transferService, WorkerPool: wp}

	r := chi.NewRouter()

	r.Use(middleware.RequestLogger)
	r.Post("/auth/register", authHandler.RegisterHandler)
	r.Post("/auth/login", authHandler.Login)
	r.Get("/auth/me", middleware.AuthMiddleware(authHandler.Me))
	r.Post("/accounts", middleware.AuthMiddleware(accountHandler.CreateAccountHandler))

	r.Get("/accounts", middleware.AuthMiddleware(middleware.RateLimitMiddleware(rdb)(accountHandler.GetAccounts)))
	r.Get("/accounts/{id}", middleware.AuthMiddleware(accountHandler.GetAccountByID))
	r.Post("/transfers", middleware.AuthMiddleware(middleware.RateLimitMiddleware(rdb)(transferHandler.TransferHandler)))

	r.Post("/agent/query", middleware.AuthMiddleware(agentHandler.QueryHandler))
	r.Post("/agent/reconcile", middleware.AuthMiddleware(agentHandler.ReconcileHandler))
	r.Post("/agent/anomaly-scan", middleware.AuthMiddleware(agentHandler.AnomalyHandler))

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "ok"}`))
	})

	go func() {
		slog.Info("Server starting", "port", 8080)
		if err := http.ListenAndServe(":8080", r); err != nil {
			slog.Error("Server failed", "error", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit
	slog.Info("Shutting down server...")
	cancel()
	pool.Close()
	slog.Info("Server stopped")

}
