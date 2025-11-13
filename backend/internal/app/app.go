package app

import (
	"context"
	"database/sql"
	"github.com/go-chi/chi/v5"
	_ "github.com/jackc/pgx/v5/stdlib"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"test_task_avito/backend/config"
	"test_task_avito/backend/internal/adapter/repository/postgres"
	"test_task_avito/backend/internal/input/http/gen"
	"test_task_avito/backend/internal/input/http/handler"
	usecase2 "test_task_avito/backend/internal/usecase"
	"test_task_avito/backend/pkg/migration"
	"time"

	"go.uber.org/zap"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func Start(cfg config.Config, logger *zap.Logger) error {

	db, err := sql.Open("pgx", cfg.PgConnStr)
	if err != nil {
		logger.Fatal("failed to connect to postgres", zap.Error(err))
	}
	defer db.Close()

	// Миграции применяются автоматически при старте
	if err := migrations.Migrate(db); err != nil {
		logger.Fatal("error with create migrations", zap.Error(err))
	}

	logger.Info("Migrations applied successfully")

	// Проверяем подключение
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		logger.Fatal("Failed to ping database", zap.Error(err))
	}

	// Настраиваем пул соединений
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	logger.Info("Database connection established")

	// Создаем репозитории
	repo := postgres.NewPostgresRepository(db)

	// Создаем use cases
	teamUseCase := usecase2.NewTeamUseCase(repo, repo, repo)
	userUseCase := usecase2.NewUserUseCase(repo, repo)
	prUseCase := usecase2.NewPullRequestUseCase(repo, repo, repo)

	// Создаем handler
	h := handler.NewHandler(teamUseCase, userUseCase, prUseCase)

	// Создаем strict handler
	strictHandler := gen.NewStrictHandler(h, nil)

	// Настраиваем роутер
	r := chi.NewRouter()
	r.Use(loggingMiddleware)
	r.Use(corsMiddleware)

	// Регистрируем routes
	gen.HandlerFromMux(strictHandler, r)

	// Добавляем health check
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Запускаем сервер
	srv := &http.Server{
		Addr:    cfg.HTTPPort,
		Handler: r,
	}

	// Graceful shutdown
	go func() {
		logger.Info("Starting server", zap.String("port", cfg.HTTPPort))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("Server failed to start", zap.Error(err))
		}
	}()

	// Ожидаем сигнал для graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")

	ctx, cancel = context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Warn("Server forced to shutdown", zap.Error(err))
		return err
	}

	logger.Info("Server exited")
	return nil
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %v", r.Method, r.URL.Path, time.Since(start))
	})
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}
