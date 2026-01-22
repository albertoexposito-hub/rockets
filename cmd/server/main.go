package main

import (
	"context"
	"fmt"
	"log"
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
	// Inicializar Kafka
	kafkaEventStore := infrastructure.NewKafkaEventStore("localhost:9092")
	// Inicializar Redis
	redisRepository := infrastructure.NewRocketRepository("localhost:6379", kafkaEventStore)
	// Inicializar servicio de aplicación
	rocketService := application.NewRocketApplicationService(redisRepository, kafkaEventStore)

	// Configurar worker pool
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

	// Configurar manejadores HTTP
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"status":"ok"}`)
	})
	http.HandleFunc("/messages", api.HandleMessages(workerPool))
	// Registrar rutas para listar y obtener por canal
	http.HandleFunc("/rockets", api.HandleListRockets(rocketService))
	http.HandleFunc("/rockets/", api.HandleListRockets(rocketService))
	// Debug endpoint para ver el estado del buffer
	http.HandleFunc("/debug/buffer", api.HandleDebugBuffer(rocketService))

	// Iniciar servidor HTTP
	server := &http.Server{
		Addr:         ":8088",
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Iniciar en goroutine
	go func() {
		log.Printf("Server starting on %s", server.Addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// Esperar la señal de finalización
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt)
	<-sigChan

	// Graceful shutdown -> para no romper conexiones activas
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Printf("Server shutdown error: %v", err)
	}

	// Detener workers tras apagar el servidor
	workerCancel()
	workerPool.Wait()

	log.Println("Server stopped")
}
