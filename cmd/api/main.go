package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	httpapi "streamingplatform/internal/http"
	"streamingplatform/internal/platform/database"
	postgresrepo "streamingplatform/internal/repository/postgres"
	"streamingplatform/internal/service"
	"streamingplatform/internal/worker"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	if err := run(logger); err != nil {
		logger.Error("application stopped", "error", err)
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	databaseURL := getenv(
		"DATABASE_URL",
		"postgres://abdyrakhman:12345@localhost:5432/go_test?sslmode=disable&search_path=public",
	)

	address := getenv("HTTP_ADDR", ":8080")

	startupCtx, startupCancel := context.WithTimeout(context.Background(), 5*time.Second)
	pool, err := database.Open(startupCtx, databaseURL)
	startupCancel()
	if err != nil {
		return err
	}

	users := postgresrepo.NewUserRepository(pool)
	subscriptions := postgresrepo.NewSubscriptionRepository(pool)
	favorites := postgresrepo.NewFavoriteRepository(pool)
	history := postgresrepo.NewWatchHistoryRepository(pool)

	notifications := service.NewNotificationService([]service.Notifier{
		service.EmailNotifier{}, service.PushNotifier{},
	})
	profiles := service.NewProfileService(users, service.NewID)
	subscriptionService := service.NewSubscriptionService(
		users, subscriptions, service.FakePaymentProvider{}, notifications, service.NewID,
	)
	favoriteService := service.NewFavoriteService(favorites, service.NewID)
	historyService := service.NewWatchHistoryService(history, service.NewID)

	// Background workers have an explicit lifecycle independent of individual requests.
	workerCtx, cancelWorkers := context.WithCancel(context.Background())
	analytics := worker.NewAnalyticsProcessor(workerCtx, 4, 128)
	transcoding := worker.NewVideoTranscodingService(3, 10*time.Second, 100*time.Millisecond)
	taste := service.NewUserTasteService(service.SlowTasteSource)

	api := httpapi.NewAPI(httpapi.Dependencies{
		Profiles: profiles, Subscriptions: subscriptionService, Favorites: favoriteService,
		WatchHistory: historyService, Moderation: service.ModerationService{},
		Analytics: analytics, Transcoding: transcoding, Taste: taste,
	}, logger)
	server := &http.Server{
		Addr:              address,
		Handler:           api.Handler(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	serverErrors := make(chan error, 1)
	go func() {
		logger.Info("HTTP server started", "address", address)
		serverErrors <- server.ListenAndServe()
	}()

	// Go note: NotifyContext turns Unix signals into normal context cancellation.
	signalCtx, stopSignals := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stopSignals()

	select {
	case <-signalCtx.Done():
		logger.Info("shutdown signal received")
	case serverErr := <-serverErrors:
		if !errors.Is(serverErr, http.ErrServerClosed) {
			cancelWorkers()
			analytics.Shutdown()
			pool.Close()
			return fmt.Errorf("serve HTTP: %w", serverErr)
		}
	}

	// Shutdown order matters: first stop accepting requests and wait for in-flight
	// handlers; then stop workers; finally close the DB they may still need.
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("HTTP shutdown failed", "error", err)
	}
	cancelWorkers()
	analytics.Shutdown()
	pool.Close()
	logger.Info("graceful shutdown completed")
	return nil
}

func getenv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
