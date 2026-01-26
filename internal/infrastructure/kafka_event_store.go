package infrastructure

import (
	"log/slog"
	"sync"

	"rockets/internal/domain"
)

// KafkaEventStore implements the event store using Kafka.
// Note: for now real Kafka is disabled; we only simulate by storing in memory.
type KafkaEventStore struct {
	brokers string //not used, it's only simulated
	mu      sync.RWMutex
	events  map[string][]domain.DomainEvent // cache ordered by insertion
}

// NewKafkaEventStore creates a new event store
func NewKafkaEventStore(brokers string) *KafkaEventStore {
	return &KafkaEventStore{
		brokers: brokers,
		events:  make(map[string][]domain.DomainEvent),
	}
}

// AppendEvent append un evento al log
func (k *KafkaEventStore) AppendEvent(event domain.DomainEvent) error {
	// Enviar a Kafka (simulado)
	slog.Debug("Event stored", "type", event.GetEventType(), "channel", event.GetChannel().Value())

	// Save to in-memory cache (arrival order)
	k.mu.Lock()
	defer k.mu.Unlock()
	channel := event.GetChannel().Value()
	k.events[channel] = append(k.events[channel], event)

	return nil
}

// GetEventsByChannel obtiene todos los eventos de un canal
func (k *KafkaEventStore) GetEventsByChannel(channel *domain.Channel) ([]domain.DomainEvent, error) {
	k.mu.RLock()
	defer k.mu.RUnlock()
	items := k.events[channel.Value()]
	// Devolver copia para no exponer el slice interno
	copySlice := make([]domain.DomainEvent, len(items))
	copy(copySlice, items)
	return copySlice, nil
}

// GetAllChannels obtiene todos los canales que tienen eventos
func (k *KafkaEventStore) GetAllChannels() []string {
	k.mu.RLock()
	defer k.mu.RUnlock()
	channels := make([]string, 0, len(k.events))
	for ch := range k.events {
		channels = append(channels, ch)
	}
	return channels
}
