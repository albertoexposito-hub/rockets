package infrastructure

import (
	"fmt"
	"log"
	"sync"

	"rockets/internal/domain"

	"github.com/redis/go-redis/v9"
)

// RocketRepository implementa el repositorio de cohetes
type RocketRepository struct {
	redisAddr   string
	redisClient *redis.Client
	eventStore  *KafkaEventStore
	cache       sync.Map
}

// NewRocketRepository crea un nuevo repositorio
func NewRocketRepository(redisAddr string, eventStore *KafkaEventStore) *RocketRepository {
	return &RocketRepository{
		redisAddr: redisAddr,
		redisClient: redis.NewClient(&redis.Options{
			Addr: redisAddr,
		}),
		eventStore: eventStore,
	}
}

// GetByChannel obtiene un cohete por canal
func (r *RocketRepository) GetByChannel(channel *domain.Channel) (*domain.Rocket, error) {
	if channel == nil {
		return nil, fmt.Errorf("channel cannot be nil")
	}

	// Try to get from in-memory cache first
	if cached, ok := r.cache.Load(channel.Value()); ok {
		return cached.(*domain.Rocket), nil
	}

	// Crear nuevo cohete si no existe
	rocket := domain.NewRocket(channel)

	// Hacer replay desde el event store (stub en memoria)
	events, err := r.eventStore.GetEventsByChannel(channel)
	if err == nil && len(events) > 0 {
		_ = rocket.LoadFromHistory(events)
	}

	// Save to cache
	r.cache.Store(channel.Value(), rocket)

	return rocket, nil
}

// Save persists a rocket
func (r *RocketRepository) Save(rocket *domain.Rocket) error {
	if rocket == nil {
		return fmt.Errorf("rocket cannot be nil")
	}

	log.Printf("[REPO] Saving rocket | Channel: %s | Events to persist: %d",
		rocket.GetChannel().Value(), len(rocket.GetUncommittedEvents()))

	// Save to cache
	r.cache.Store(rocket.GetChannel().Value(), rocket)

	// Guardar eventos en el event store
	for _, event := range rocket.GetUncommittedEvents() {
		log.Printf("[REPO] Persisting event | Channel: %s | Type: %s | Msg#%d",
			rocket.GetChannel().Value(), event.GetEventType(), event.GetMessageNumber().Value())
		if err := r.eventStore.AppendEvent(event); err != nil {
			log.Printf("[REPO] ✗ Failed to persist event | Channel: %s | Error: %v",
				rocket.GetChannel().Value(), err)
			return fmt.Errorf("failed to save event: %w", err)
		}
	}

	log.Printf("[REPO] ✓ Rocket saved successfully | Channel: %s | Total events: %d",
		rocket.GetChannel().Value(), len(rocket.GetUncommittedEvents()))

	// Marcar eventos como guardados
	rocket.MarkEventsAsCommitted()

	return nil
}

// GetAll obtiene todos los cohetes (reconstruidos desde el event store)
func (r *RocketRepository) GetAll() ([]*domain.Rocket, error) {
	// Obtener todos los canales del event store
	channels := r.eventStore.GetAllChannels()

	var rockets []*domain.Rocket
	for _, channelStr := range channels {
		channel, err := domain.NewChannel(channelStr)
		if err != nil {
			continue // Skip invalid channels
		}

		// Use GetByChannel which does automatic replay
		rocket, err := r.GetByChannel(channel)
		if err != nil {
			continue
		}

		rockets = append(rockets, rocket)
	}

	return rockets, nil
}
