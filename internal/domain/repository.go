package domain

// RocketRepository define el contrato para persistir cohetes
type RocketRepository interface {
	GetByChannel(channel *Channel) (*Rocket, error)
	Save(rocket *Rocket) error
	GetAll() ([]*Rocket, error)
}

// EventStore define el contrato para almacenar eventos
type EventStore interface {
	AppendEvent(event DomainEvent) error
	GetEventsByChannel(channel *Channel) ([]DomainEvent, error)
}
