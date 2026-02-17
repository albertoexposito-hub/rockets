package application

import (
	"fmt"
	"log/slog"
	"sync"

	"rockets/internal/domain"
)

// RocketApplicationService is the application service for rockets
type RocketApplicationService struct {
	repository domain.RocketRepository
	eventStore domain.EventStore
	// Buffer for out-of-order messages -> ordering by messageNumber per channel
	pendingMessages map[string]map[int]*ProcessMessageDTO
	bufferMutex     sync.Mutex // Mutex to protect access to pendingMessages map
}

// NewRocketApplicationService creates a new application service
func NewRocketApplicationService(repository domain.RocketRepository, eventStore domain.EventStore) *RocketApplicationService {
	return &RocketApplicationService{
		repository:      repository,
		eventStore:      eventStore,
		pendingMessages: make(map[string]map[int]*ProcessMessageDTO),
	}
}

// ProcessMessageDTO represents an incoming message
type ProcessMessageDTO struct {
	Channel    string `json:"channel"`
	Number     int    `json:"number"`
	Action     string `json:"action"`
	Param      string `json:"param,omitempty"`
	Value      int    `json:"value,omitempty"`
	Time       int64  `json:"time"`
	RocketType string `json:"rocketType,omitempty"`
}

// ProcessMessage process a message with ordering guarantees
func (s *RocketApplicationService) ProcessMessage(dto *ProcessMessageDTO) error {
	if dto == nil {
		return fmt.Errorf("message DTO cannot be nil")
	}

	s.bufferMutex.Lock()
	defer s.bufferMutex.Unlock()

	// Get the last expected messageNumber
	channel, err := domain.NewChannel(dto.Channel)
	if err != nil {
		return fmt.Errorf("invalid channel: %w", err)
	}

	rocket, err := s.repository.GetByChannel(channel)
	if err != nil {
		return fmt.Errorf("failed to get rocket: %w", err)
	}

	expected := rocket.GetLastMessageNumber().Value() + 1

	slog.Debug("Message ordering check", "channel", dto.Channel, "received", dto.Number, "expected", expected)

	// If it is the expected message, process it
	if dto.Number == expected {
		slog.Info("Processing message", "channel", dto.Channel, "number", dto.Number, "action", dto.Action)
		if err := s.processMessageDirect(dto); err != nil {
			return err
		}

		// Process consecutive messages from the buffer
		for {
			nextNum := expected + 1
			if s.pendingMessages[dto.Channel] == nil {
				break
			}
			nextDTO := s.pendingMessages[dto.Channel][nextNum]
			if nextDTO == nil {
				break
			}

			slog.Debug("Processing buffered message", "channel", dto.Channel, "number", nextNum, "action", nextDTO.Action)
			if err := s.processMessageDirect(nextDTO); err != nil {
				return err
			}
			delete(s.pendingMessages[dto.Channel], nextNum)
			expected = nextNum
		}

		return nil
	}

	// If it is a future message, store it in the buffer
	if dto.Number > expected {
		if s.pendingMessages[dto.Channel] == nil {
			s.pendingMessages[dto.Channel] = make(map[int]*ProcessMessageDTO)
		}
		s.pendingMessages[dto.Channel][dto.Number] = dto
		slog.Debug("Message stored in buffer", "channel", dto.Channel, "number", dto.Number, "waiting_for", expected)
		slog.Debug("Buffered messages", "channel", dto.Channel, "pending", s.getBufferedMessageNumbers(dto.Channel))
		return nil // Not an error, just waiting
	}

	// If it is an old or duplicate message, reject
	slog.Warn("Message rejected - already processed", "channel", dto.Channel, "number", dto.Number, "expected", expected)
	return fmt.Errorf("message %d already processed (expected %d)", dto.Number, expected)
}

// processMessageDirect processes a message directly (without buffer)
func (s *RocketApplicationService) processMessageDirect(dto *ProcessMessageDTO) error {
	if dto == nil {
		return fmt.Errorf("message DTO cannot be nil")
	}

	// Validate and create value objects
	channel, err := domain.NewChannel(dto.Channel)
	if err != nil {
		return fmt.Errorf("invalid channel: %w", err)
	}

	msgNum, err := domain.NewMessageNumber(dto.Number)
	if err != nil {
		return fmt.Errorf("invalid message number: %w", err)
	}

	// Get or create rocket
	rocket, err := s.repository.GetByChannel(channel)
	if err != nil {
		return fmt.Errorf("failed to get rocket: %w", err)
	}

	// Process action
	switch dto.Action {
	case "launch":
		speed, _ := domain.NewSpeed(dto.Value)
		mission := domain.NewMission(dto.Param)
		rocketType := dto.RocketType
		if rocketType == "" {
			rocketType = "unknown"
		}
		if err := rocket.Launch(msgNum, rocketType, speed, mission, dto.Time); err != nil {
			return err
		}

	case "increase_speed":
		if err := rocket.IncreaseSpeed(msgNum, dto.Value, dto.Time); err != nil {
			return err
		}

	case "decrease_speed":
		if err := rocket.DecreaseSpeed(msgNum, dto.Value, dto.Time); err != nil {
			return err
		}

	case "explode":
		if err := rocket.Explode(msgNum, dto.Param, dto.Time); err != nil {
			return err
		}

	case "change_mission":
		mission := domain.NewMission(dto.Param)
		if err := rocket.ChangeMission(msgNum, mission, dto.Time); err != nil {
			return err
		}

	default:
		return fmt.Errorf("unknown action: %s", dto.Action)
	}

	// Save changes
	return s.repository.Save(rocket)
}

// getBufferedMessageNumbers returns the message numbers in the buffer
func (s *RocketApplicationService) getBufferedMessageNumbers(channel string) []int {
	numbers := []int{}
	if s.pendingMessages[channel] == nil {
		return numbers
	}
	for num := range s.pendingMessages[channel] {
		numbers = append(numbers, num)
	}
	return numbers
}

// RocketDTO represents a rocket to be exposed via API
type RocketDTO struct {
	Channel string `json:"channel"`
	Type    string `json:"type"`
	Status  string `json:"status"`
	Speed   int    `json:"speed"`
	Mission string `json:"mission"`
}

// EventDTO represents an event to be exposed via API
type EventDTO struct {
	Type          string `json:"type"`
	MessageNumber int    `json:"messageNumber"`
	Timestamp     int64  `json:"timestamp"`
	Details       string `json:"details,omitempty"`
}

// GetRocket gets the current state of a rocket
func (s *RocketApplicationService) GetRocket(channelStr string) (*RocketDTO, error) {
	channel, err := domain.NewChannel(channelStr)
	if err != nil {
		return nil, err
	}

	rocket, err := s.repository.GetByChannel(channel)
	if err != nil {
		return nil, err
	}

	return &RocketDTO{
		Channel: channel.Value(),
		Type:    rocket.GetRocketType(),
		Status:  string(rocket.GetStatus()),
		Speed:   rocket.GetSpeed().Value(),
		Mission: string(rocket.GetMission()),
	}, nil
}

// ListRockets gets all rockets
func (s *RocketApplicationService) ListRockets() ([]*RocketDTO, error) {
	rockets, err := s.repository.GetAll()
	if err != nil {
		return nil, err
	}

	var dtos []*RocketDTO
	for _, rocket := range rockets {
		dtos = append(dtos, &RocketDTO{
			Channel: rocket.GetChannel().Value(),
			Type:    rocket.GetRocketType(),
			Status:  string(rocket.GetStatus()),
			Speed:   rocket.GetSpeed().Value(),
			Mission: string(rocket.GetMission()),
		})
	}

	return dtos, nil
}

// ListEvents gets the events of a channel (ordered by arrival in the store)
func (s *RocketApplicationService) ListEvents(channelStr string) ([]*EventDTO, error) {
	channel, err := domain.NewChannel(channelStr)
	if err != nil {
		return nil, err
	}

	events, err := s.eventStore.GetEventsByChannel(channel)
	if err != nil {
		return nil, err
	}

	var dtos []*EventDTO
	for _, ev := range events {
		e := &EventDTO{
			Type:          ev.GetEventType(),
			MessageNumber: ev.GetMessageNumber().Value(),
			Timestamp:     ev.GetTimestamp(),
		}
		switch v := ev.(type) {
		case *domain.RocketLaunched:
			e.Details = fmt.Sprintf("mission=%s speed=%d", v.Mission, v.Speed.Value())
		case *domain.RocketSpeedIncreased:
			e.Details = fmt.Sprintf("delta=%d newSpeed=%d", v.Delta, v.NewSpeed.Value())
		case *domain.RocketSpeedDecreased:
			e.Details = fmt.Sprintf("delta=%d newSpeed=%d", v.Delta, v.NewSpeed.Value())
		case *domain.RocketMissionChanged:
			e.Details = fmt.Sprintf("mission=%s", v.NewMission)
		case *domain.RocketExploded:
			e.Details = fmt.Sprintf("reason=%s", v.Reason)
		}
		dtos = append(dtos, e)
	}

	return dtos, nil
}

// BufferStatusDTO represents the buffer status for debugging
type BufferStatusDTO struct {
	Channel          string `json:"channel"`
	ExpectedNext     int    `json:"expectedNext"`
	BufferedMessages []int  `json:"bufferedMessages"`
}

// GetBufferStatus returns the buffer status for all channels
func (s *RocketApplicationService) GetBufferStatus() []*BufferStatusDTO {
	s.bufferMutex.Lock()
	defer s.bufferMutex.Unlock()

	var status []*BufferStatusDTO
	for channel, messages := range s.pendingMessages {
		// Get last processed message
		ch, _ := domain.NewChannel(channel)
		rocket, _ := s.repository.GetByChannel(ch)
		expectedNext := rocket.GetLastMessageNumber().Value() + 1

		numbers := []int{}
		for num := range messages {
			numbers = append(numbers, num)
		}

		status = append(status, &BufferStatusDTO{
			Channel:          channel,
			ExpectedNext:     expectedNext,
			BufferedMessages: numbers,
		})
	}

	return status
}
