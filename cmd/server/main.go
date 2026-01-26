package main

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"time"

	"rockets/internal/api"
	"rockets/internal/application"
	"rockets/internal/infrastructure"
)

func main() {
	// Initialize structured logging with JSON handler
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})))

	// Initialize Kafka
	kafkaEventStore := infrastructure.NewKafkaEventStore("localhost:9092")
	// Initialize Redis
	redisRepository := infrastructure.NewRocketRepository("localhost:6379", kafkaEventStore)
	// Initialize application service
	rocketService := application.NewRocketApplicationService(redisRepository, kafkaEventStore)

	// Configure worker pool
	workerCount := 3
	if value := os.Getenv("WORKER_COUNT"); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil && parsed > 0 {
			workerCount = parsed
		}
	}
	workerPool := application.NewWorkerPool(rocketService, workerCount)
	workerCtx, workerCancel := context.WithCancel(context.Background())
	defer workerCancel()
	workerPool.Start(workerCtx)

	// Configure HTTP handlers
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]string{"status": "ok"}); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})
	http.HandleFunc("/messages", api.HandleMessages(workerPool))
	// Register routes to list and get by channel
	http.HandleFunc("/rockets", api.HandleListRockets(rocketService))
	http.HandleFunc("/rockets/", api.HandleListRockets(rocketService))
	// Debug endpoint to see buffer state
	http.HandleFunc("/debug/buffer", api.HandleDebugBuffer(rocketService))

	// Start HTTP server
	server := &http.Server{
		Addr:         ":8088",
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Iniciar en goroutine
	go func() {
		slog.Info("Server starting", "addr", server.Addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("Server error", "err", err)
			os.Exit(1)
		}
	}()

	// Wait for termination signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt)
	<-sigChan

	// Graceful shutdown -> don't break active connections
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		slog.Error("Server shutdown error", "err", err)
	}

	// Stop workers after shutting down server
	workerCancel()
	workerPool.Wait()

	slog.Info("Server stopped")
}
