package application

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
)

// WorkerPool process messages concurrently with a fixed number of workers.
type WorkerPool struct {
	service     *RocketApplicationService
	jobs        chan *ProcessMessageDTO
	wg          sync.WaitGroup
	workerCount int
	ctx         context.Context // Context to manage shutdown
}

// NewWorkerPool creates a pool with a fixed number of workers.
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

// Start launches the workers and logs their start.
func (p *WorkerPool) Start(ctx context.Context) {
	p.ctx = ctx
	slog.Debug("Workers started", "count", p.workerCount)

	// Simulate redelivery loop (watchdog stub)
	go func() {
		slog.Debug("Message redelivery loop started")
		<-ctx.Done()
	}()

	// Start workers
	for i := 1; i <= p.workerCount; i++ {
		// Launch worker goroutine
		p.wg.Add(1)
		go func(id int) {
			defer p.wg.Done()
			slog.Debug("Worker started and waiting for jobs", "worker_id", id)
			for {
				select {
				case <-ctx.Done(): // Shutdown signal
					slog.Debug("Worker shutting down", "worker_id", id)
					return
				case job, ok := <-p.jobs: // Receive job
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

// Enqueue adds a message to the queue.
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

// Wait waits for all workers to finish.
// This should be called after cancelling the context to ensure graceful shutdown.
func (p *WorkerPool) Wait() {
	p.wg.Wait()
}
