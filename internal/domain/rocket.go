package domain

import (
	"fmt"
	"log/slog"
)

// Rocket is the aggregate root representing a rocket
type Rocket struct {
	channel           *Channel
	rocketType        string
	status            RocketStatus
	speed             *Speed
	mission           Mission
	lastMessageNumber *MessageNumber
	uncommittedEvents []DomainEvent
}

// NewRocket creates a new Rocket instance
func NewRocket(channel *Channel) *Rocket {
	return &Rocket{
		channel:           channel,
		rocketType:        "unknown",
		status:            StatusLaunched,
		speed:             &Speed{value: 0},
		mission:           MissionUnknown,
		lastMessageNumber: &MessageNumber{value: 0},
		uncommittedEvents: []DomainEvent{},
	}
}

// Launch launches the rocket
func (r *Rocket) Launch(msgNum *MessageNumber, rocketType string, speed *Speed, mission Mission, timestamp int64) error {
	if r.status != StatusLaunched {
		return fmt.Errorf("rocket already launched")
	}

	if msgNum.Value() <= r.lastMessageNumber.Value() {
		return fmt.Errorf("message number out of order")
	}

	event := &RocketLaunched{
		Channel:       r.channel,
		MessageNumber: msgNum,
		Type:          rocketType,
		Speed:         speed,
		Mission:       mission,
		Timestamp:     timestamp,
	}

	slog.Info("Applying RocketLaunched",
		"channel", r.channel.Value(),
		"message_number", msgNum.Value(),
		"rocket_type", rocketType,
		"speed", speed.Value(),
		"mission", mission)

	r.applyEvent(event)
	r.uncommittedEvents = append(r.uncommittedEvents, event)
	return nil
}

// IncreaseSpeed increases the rocket's speed
func (r *Rocket) IncreaseSpeed(msgNum *MessageNumber, delta int, timestamp int64) error {
	if r.status == StatusExploded {
		return fmt.Errorf("cannot change crashed rocket")
	}

	if msgNum.Value() <= r.lastMessageNumber.Value() {
		return fmt.Errorf("message number out of order")
	}

	newSpeed := r.speed.Increase(delta)

	event := &RocketSpeedIncreased{
		Channel:       r.channel,
		MessageNumber: msgNum,
		OldSpeed:      r.speed,
		NewSpeed:      newSpeed,
		Delta:         delta,
		Timestamp:     timestamp,
	}

	slog.Info("Applying SpeedIncreased",
		"channel", r.channel.Value(),
		"message_number", msgNum.Value(),
		"before", r.speed.Value(),
		"after", newSpeed.Value(),
		"increment", delta)

	r.applyEvent(event)
	r.uncommittedEvents = append(r.uncommittedEvents, event)
	return nil
}

// DecreaseSpeed decreases the rocket's speed
func (r *Rocket) DecreaseSpeed(msgNum *MessageNumber, delta int, timestamp int64) error {
	if r.status == StatusExploded {
		return fmt.Errorf("cannot change crashed rocket")
	}

	if msgNum.Value() <= r.lastMessageNumber.Value() {
		return fmt.Errorf("message number out of order")
	}

	newSpeed := r.speed.Decrease(delta)

	event := &RocketSpeedDecreased{
		Channel:       r.channel,
		MessageNumber: msgNum,
		OldSpeed:      r.speed,
		NewSpeed:      newSpeed,
		Delta:         delta,
		Timestamp:     timestamp,
	}

	slog.Info("Applying SpeedDecreased",
		"channel", r.channel.Value(),
		"message_number", msgNum.Value(),
		"before", r.speed.Value(),
		"after", newSpeed.Value(),
		"decrement", delta)

	r.applyEvent(event)
	r.uncommittedEvents = append(r.uncommittedEvents, event)
	return nil
}

// Explode explodes the rocket
func (r *Rocket) Explode(msgNum *MessageNumber, reason string, timestamp int64) error {
	if r.status == StatusExploded {
		return fmt.Errorf("rocket already exploded")
	}

	if msgNum.Value() <= r.lastMessageNumber.Value() {
		return fmt.Errorf("message number out of order")
	}

	event := &RocketExploded{
		Channel:       r.channel,
		MessageNumber: msgNum,
		Reason:        reason,
		Timestamp:     timestamp,
	}

	r.applyEvent(event)
	r.uncommittedEvents = append(r.uncommittedEvents, event)
	return nil
}

// ChangeMission changes the rocket's mission
func (r *Rocket) ChangeMission(msgNum *MessageNumber, newMission Mission, timestamp int64) error {
	if r.status == StatusExploded {
		return fmt.Errorf("cannot change crashed rocket")
	}

	if msgNum.Value() <= r.lastMessageNumber.Value() {
		return fmt.Errorf("message number out of order")
	}

	event := &RocketMissionChanged{
		Channel:       r.channel,
		MessageNumber: msgNum,
		OldMission:    r.mission,
		NewMission:    newMission,
		Timestamp:     timestamp,
	}

	r.applyEvent(event)
	r.uncommittedEvents = append(r.uncommittedEvents, event)
	return nil
}

// GetUncommittedEvents returns the uncommitted events
func (r *Rocket) GetUncommittedEvents() []DomainEvent {
	return r.uncommittedEvents
}

// MarkEventsAsCommitted marks events as committed
func (r *Rocket) MarkEventsAsCommitted() {
	r.uncommittedEvents = []DomainEvent{}
}

// LoadFromHistory reconstructs the state from the event history
func (r *Rocket) LoadFromHistory(events []DomainEvent) error {
	for _, event := range events {
		r.applyEvent(event)
	}
	return nil
}

// applyEvent applies an event to the internal state
func (r *Rocket) applyEvent(event DomainEvent) {
	switch e := event.(type) {
	case *RocketLaunched:
		r.status = StatusFlying
		r.rocketType = e.Type
		r.speed = e.Speed
		r.mission = e.Mission
		r.lastMessageNumber = e.MessageNumber

	case *RocketSpeedIncreased:
		r.speed = e.NewSpeed
		r.lastMessageNumber = e.MessageNumber

	case *RocketSpeedDecreased:
		r.speed = e.NewSpeed
		r.lastMessageNumber = e.MessageNumber

	case *RocketExploded:
		r.status = StatusExploded
		r.lastMessageNumber = e.MessageNumber

	case *RocketMissionChanged:
		r.mission = e.NewMission
		r.lastMessageNumber = e.MessageNumber
	}
}

// GetChannel returns the rocket's channel
func (r *Rocket) GetChannel() *Channel {
	return r.channel
}

// GetStatus returns the rocket's status
func (r *Rocket) GetStatus() RocketStatus {
	return r.status
}

// GetSpeed returns the rocket's speed
func (r *Rocket) GetSpeed() *Speed {
	return r.speed
}

// GetMission returns the rocket's mission
func (r *Rocket) GetMission() Mission {
	return r.mission
}

// GetRocketType returns the rocket's type
func (r *Rocket) GetRocketType() string {
	return r.rocketType
}

// GetLastMessageNumber returns the last applied messageNumber
func (r *Rocket) GetLastMessageNumber() *MessageNumber {
	return r.lastMessageNumber
}
