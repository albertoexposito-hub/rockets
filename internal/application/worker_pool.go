package application

import (
	"context"
	"fmt"
	"log"
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
	log.Printf("[debug] %d workers started", p.workerCount)

	// Simula bucle de reentrega (stub de vigilancia)
	go func() {
		log.Printf("[debug] Message redelivery loop started")
		<-ctx.Done()
	}()

	for i := 1; i <= p.workerCount; i++ {
		p.wg.Add(1)
		go func(id int) {
			defer p.wg.Done()
			log.Printf("[WORKER-%d] Started and waiting for jobs", id)
			for {
				select {
				case <-ctx.Done():
					log.Printf("[WORKER-%d] Shutting down", id)
					return
				case job, ok := <-p.jobs:
					if !ok {
						log.Printf("[WORKER-%d] Channel closed", id)
						return
					}
					log.Printf("[WORKER-%d] ← Picked up job | Channel: %s | Msg#%d | Action: %s",
						id, job.Channel, job.Number, job.Action)

					if err := p.service.ProcessMessage(job); err != nil {
						log.Printf("[WORKER-%d] ✗ Error processing | Channel: %s | Msg#%d | Error: %v",
							id, job.Channel, job.Number, err)
					} else {
						log.Printf("[WORKER-%d] ✓ Successfully processed | Channel: %s | Msg#%d",
							id, job.Channel, job.Number)
					}
				}
			}
		}(i)
	}
}

// Enqueue agrega un mensaje a la cola.
func (p *WorkerPool) Enqueue(dto *ProcessMessageDTO) error {
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
