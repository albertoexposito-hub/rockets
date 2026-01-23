package domain

import (
	"fmt"
	"strings"
)

// Channel representa el canal o identificador de un cohete
type Channel struct {
	value string
}

// NewChannel crea un nuevo canal
func NewChannel(value string) (*Channel, error) {
	if strings.TrimSpace(value) == "" {
		return nil, fmt.Errorf("channel cannot be empty")
	}
	return &Channel{value: value}, nil
}

// Value retorna el valor del canal
func (c *Channel) Value() string {
	return c.value
}

// MessageNumber represents the sequential number of a message
type MessageNumber struct {
	value int
}

// NewMessageNumber creates a new message number
func NewMessageNumber(value int) (*MessageNumber, error) {
	if value <= 0 {
		return nil, fmt.Errorf("message number must be positive")
	}
	return &MessageNumber{value: value}, nil
}

// Value returns the number
func (m *MessageNumber) Value() int {
	return m.value
}

// Speed representa la velocidad de un cohete
type Speed struct {
	value int // en km/h
}

// NewSpeed crea una nueva velocidad
func NewSpeed(value int) (*Speed, error) {
	if value < 0 {
		return nil, fmt.Errorf("speed cannot be negative")
	}
	return &Speed{value: value}, nil
}

// Value retorna la velocidad
func (s *Speed) Value() int {
	return s.value
}

// Increase incrementa la velocidad
func (s *Speed) Increase(delta int) *Speed {
	return &Speed{value: s.value + delta}
}

// Decrease disminuye la velocidad
func (s *Speed) Decrease(delta int) *Speed {
	if delta >= s.value {
		return &Speed{value: 0}
	}
	return &Speed{value: s.value - delta}
}

// Mission represents the rocket's mission
type Mission string

const (
	MissionExploration Mission = "exploration"
	MissionSatellite   Mission = "satellite"
	MissionResupply    Mission = "resupply"
	MissionUnknown     Mission = "unknown"
)

// NewMission creates a new mission
func NewMission(value string) Mission {
	m := Mission(strings.ToLower(value))
	switch m {
	case MissionExploration, MissionSatellite, MissionResupply:
		return m
	default:
		return MissionUnknown
	}
}

// MessageTime representa el tiempo de un mensaje
type MessageTime struct {
	value int64 // Unix timestamp in milliseconds
}

// NewMessageTime crea un nuevo tiempo de mensaje
func NewMessageTime(value int64) (*MessageTime, error) {
	if value <= 0 {
		return nil, fmt.Errorf("message time must be positive")
	}
	return &MessageTime{value: value}, nil
}

// Value retorna el timestamp
func (m *MessageTime) Value() int64 {
	return m.value
}

// RocketStatus enumera los posibles estados de un cohete
type RocketStatus string

const (
	StatusLaunched RocketStatus = "launched"
	StatusFlying   RocketStatus = "flying"
	StatusExploded RocketStatus = "exploded"
)
