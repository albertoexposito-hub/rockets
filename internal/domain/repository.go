package domain

// RocketRepository defines the contract for rocket persistence
type RocketRepository interface {
	GetByChannel(channel *Channel) (*Rocket, error)
	Save(rocket *Rocket) error
	GetAll() ([]*Rocket, error)
}

// EventStore defines the contract for event storage
type EventStore interface {
	AppendEvent(event DomainEvent) error
	GetEventsByChannel(channel *Channel) ([]DomainEvent, error)
}
