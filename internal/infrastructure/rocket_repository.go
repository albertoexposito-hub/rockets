package infrastructure

import (
	"fmt"
	"log/slog"
	"sync"

	"rockets/internal/domain"

	"github.com/redis/go-redis/v9"
)

// RocketRepository implements the RocketRepository using Redis and KafkaEventStore SIMULATION
type RocketRepository struct {
	redisAddr   string
	redisClient *redis.Client
	eventStore  *KafkaEventStore
	cache       sync.Map
}

// NewRocketRepository creates a new RocketRepository
func NewRocketRepository(redisAddr string, eventStore *KafkaEventStore) *RocketRepository {
	return &RocketRepository{
		redisAddr: redisAddr,
		redisClient: redis.NewClient(&redis.Options{
			Addr: redisAddr,
		}),
		eventStore: eventStore,
	}
}

// GetByChannel get a rocket by channel - FAKECONSUMER
func (r *RocketRepository) GetByChannel(channel *domain.Channel) (*domain.Rocket, error) {
	if channel == nil {
		return nil, fmt.Errorf("channel cannot be nil")
	}

	// Try to get from in-memory cache first
	if cached, ok := r.cache.Load(channel.Value()); ok {
		return cached.(*domain.Rocket), nil
	}

	// Create new rocket if it doesn't exist
	rocket := domain.NewRocket(channel)

	// Replay from the event store (in-memory stub)
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

	slog.Debug("Saving rocket", "channel", rocket.GetChannel().Value(), "pending_events", len(rocket.GetUncommittedEvents()))

	// Save to cache
	r.cache.Store(rocket.GetChannel().Value(), rocket)

	// Save events to the event store
	for _, event := range rocket.GetUncommittedEvents() {
		slog.Debug("Persisting event",
			"channel", rocket.GetChannel().Value(),
			"type", event.GetEventType(),
			"message_number", event.GetMessageNumber().Value())
		if err := r.eventStore.AppendEvent(event); err != nil {
			slog.Error("Failed to persist event",
				"channel", rocket.GetChannel().Value(),
				"err", err)
			return fmt.Errorf("failed to save event: %w", err)
		}
	}

	slog.Info("Rocket saved successfully", "channel", rocket.GetChannel().Value(), "total_events", len(rocket.GetUncommittedEvents()))

	// Mark events as committed
	rocket.MarkEventsAsCommitted()

	return nil
}

// GetAll gets all rockets (reconstructed from the event store)
func (r *RocketRepository) GetAll() ([]*domain.Rocket, error) {
	// Get all channels from the event store
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
