package domain

// DomainEvent is the interface for all domain events
type DomainEvent interface {
	GetEventType() string
	GetChannel() *Channel
	GetMessageNumber() *MessageNumber
	GetTimestamp() int64
}

// RocketLaunched event when a rocket is launched
type RocketLaunched struct {
	Channel       *Channel
	MessageNumber *MessageNumber
	Type          string
	Speed         *Speed
	Mission       Mission
	Timestamp     int64
}

func (e *RocketLaunched) GetEventType() string             { return "rocket_launched" }
func (e *RocketLaunched) GetChannel() *Channel             { return e.Channel }
func (e *RocketLaunched) GetMessageNumber() *MessageNumber { return e.MessageNumber }
func (e *RocketLaunched) GetTimestamp() int64              { return e.Timestamp }

// RocketSpeedIncreased event when speed increases
type RocketSpeedIncreased struct {
	Channel       *Channel
	MessageNumber *MessageNumber
	OldSpeed      *Speed
	NewSpeed      *Speed
	Delta         int
	Timestamp     int64
}

func (e *RocketSpeedIncreased) GetEventType() string             { return "rocket_speed_increased" }
func (e *RocketSpeedIncreased) GetChannel() *Channel             { return e.Channel }
func (e *RocketSpeedIncreased) GetMessageNumber() *MessageNumber { return e.MessageNumber }
func (e *RocketSpeedIncreased) GetTimestamp() int64              { return e.Timestamp }

// RocketSpeedDecreased event when speed decreases
type RocketSpeedDecreased struct {
	Channel       *Channel
	MessageNumber *MessageNumber
	OldSpeed      *Speed
	NewSpeed      *Speed
	Delta         int
	Timestamp     int64
}

func (e *RocketSpeedDecreased) GetEventType() string             { return "rocket_speed_decreased" }
func (e *RocketSpeedDecreased) GetChannel() *Channel             { return e.Channel }
func (e *RocketSpeedDecreased) GetMessageNumber() *MessageNumber { return e.MessageNumber }
func (e *RocketSpeedDecreased) GetTimestamp() int64              { return e.Timestamp }

// RocketExploded event when rocket explodes
type RocketExploded struct {
	Channel       *Channel
	MessageNumber *MessageNumber
	Reason        string
	Timestamp     int64
}

func (e *RocketExploded) GetEventType() string             { return "rocket_exploded" }
func (e *RocketExploded) GetChannel() *Channel             { return e.Channel }
func (e *RocketExploded) GetMessageNumber() *MessageNumber { return e.MessageNumber }
func (e *RocketExploded) GetTimestamp() int64              { return e.Timestamp }

// RocketMissionChanged event when mission changes
type RocketMissionChanged struct {
	Channel       *Channel
	MessageNumber *MessageNumber
	OldMission    Mission
	NewMission    Mission
	Timestamp     int64
}

func (e *RocketMissionChanged) GetEventType() string             { return "rocket_mission_changed" }
func (e *RocketMissionChanged) GetChannel() *Channel             { return e.Channel }
func (e *RocketMissionChanged) GetMessageNumber() *MessageNumber { return e.MessageNumber }
func (e *RocketMissionChanged) GetTimestamp() int64              { return e.Timestamp }
