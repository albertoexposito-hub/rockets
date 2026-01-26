package application

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
)

// WorkerPool procesa mensajes en paralelo usando una cola interna.
type WorkerPool struct {
	service     *RocketApplicationService
	jobs        chan *ProcessMessageDTO
	wg          sync.WaitGroup
	workerCount int
	ctx         context.Context
}

// NewWorkerPool crea un pool con un numero de workers fijo.
func NewWorkerPool(service *RocketApplicationService, workerCount int) *WorkerPool {
	if service == nil {
		panic("service cannot be nil")
	}
	if workerCount <= 0 {
		workerCount = 1
	}
	return &WorkerPool{
		service:     service,
		jobs:        make(chan *ProcessMessageDTO, 100),
		workerCount: workerCount,
	}
}

// Start lanza los workers y registra logs de inicio.
func (p *WorkerPool) Start(ctx context.Context) {
	p.ctx = ctx
	slog.Debug("Workers started", "count", p.workerCount)

	// Simula bucle de reentrega (stub de vigilancia)
	go func() {
		slog.Debug("Message redelivery loop started")
		<-ctx.Done()
	}()

	for i := 1; i <= p.workerCount; i++ {
		p.wg.Add(1)
		go func(id int) {
			defer p.wg.Done()
			slog.Debug("Worker started and waiting for jobs", "worker_id", id)
			for {
				select {
				case <-ctx.Done():
					slog.Debug("Worker shutting down", "worker_id", id)
					return
				case job, ok := <-p.jobs:
					if !ok {
						slog.Debug("Job channel closed", "worker_id", id)
						return
					}
					slog.Debug("Picked up job",
						"worker_id", id,
						"channel", job.Channel,
						"number", job.Number,
						"action", job.Action)

					if err := p.service.ProcessMessage(job); err != nil {
						slog.Error("Error processing message",
							"worker_id", id,
							"channel", job.Channel,
							"number", job.Number,
							"err", err)
					} else {
						slog.Info("Message processed successfully",
							"worker_id", id,
							"channel", job.Channel,
							"number", job.Number)
					}
				}
			}
		}(i)
	}
}

// Enqueue agrega un mensaje a la cola.
func (p *WorkerPool) Enqueue(dto *ProcessMessageDTO) error {
	if dto == nil {
		return fmt.Errorf("message DTO cannot be nil")
	}

	select {
	case <-p.ctx.Done():
		return fmt.Errorf("worker pool stopped")
	default:
	}

	select {
	case p.jobs <- dto:
		return nil
	case <-p.ctx.Done():
		return fmt.Errorf("worker pool stopped")
	}
}

// Wait espera a que todos los workers terminen.
func (p *WorkerPool) Wait() {
	p.wg.Wait()
}
