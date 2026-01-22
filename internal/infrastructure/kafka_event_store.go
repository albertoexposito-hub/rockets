package infrastructure

import (
	"log"
	"sync"

	"rockets/internal/domain"
)

// KafkaEventStore implementa el almacén de eventos usando Kafka.
// Nota: por ahora Kafka real está deshabilitado; solo simulamos guardando en memoria.
type KafkaEventStore struct {
	brokers string
	mu      sync.RWMutex
	events  map[string][]domain.DomainEvent // cache ordenada por inserción
}

// NewKafkaEventStore crea un nuevo almacén de eventos
func NewKafkaEventStore(brokers string) *KafkaEventStore {
	return &KafkaEventStore{
		brokers: brokers,
		events:  make(map[string][]domain.DomainEvent),
	}
}

// AppendEvent append un evento al log
func (k *KafkaEventStore) AppendEvent(event domain.DomainEvent) error {
	// Enviar a Kafka (simulado)
	log.Printf("Event stored: %s for channel: %s", event.GetEventType(), event.GetChannel().Value())

	// Guardar en caché in-memory (orden de llegada)
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
