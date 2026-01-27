package domain

import (
	"fmt"
	"strings"
)

// Channel represents a communication channel for a rocket
type Channel struct {
	value string
}

// NewChannel creates a new channel
func NewChannel(value string) (*Channel, error) {
	if strings.TrimSpace(value) == "" {
		return nil, fmt.Errorf("channel cannot be empty")
	}
	return &Channel{value: value}, nil
}

// Value returns the channel's value
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

// Speed represents the speed of a rocket
type Speed struct {
	value int // in km/h
}

// NewSpeed creates a new speed
func NewSpeed(value int) (*Speed, error) {
	if value < 0 {
		return nil, fmt.Errorf("speed cannot be negative")
	}
	return &Speed{value: value}, nil
}

// Value returns the speed
func (s *Speed) Value() int {
	return s.value
}

// Increase increases the speed
func (s *Speed) Increase(delta int) *Speed {
	return &Speed{value: s.value + delta}
}

// Decrease decreases the speed
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

// MessageTime represents the time of a message
type MessageTime struct {
	value int64 // Unix timestamp in milliseconds
}

// NewMessageTime creates a new message time
func NewMessageTime(value int64) (*MessageTime, error) {
	if value <= 0 {
		return nil, fmt.Errorf("message time must be positive")
	}
	return &MessageTime{value: value}, nil
}

// Value returns the timestamp
func (m *MessageTime) Value() int64 {
	return m.value
}

// RocketStatus enumerates the possible states of a rocket
type RocketStatus string

const (
	StatusLaunched RocketStatus = "launched"
	StatusFlying   RocketStatus = "flying"
	StatusExploded RocketStatus = "exploded"
)
