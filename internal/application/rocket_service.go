package application

import (
	"fmt"
	"log"
	"sync"

	"rockets/internal/domain"
)

// RocketApplicationService implementa los casos de uso
type RocketApplicationService struct {
	repository domain.RocketRepository
	eventStore domain.EventStore
	// Buffer para mensajes fuera de orden
	pendingMessages map[string]map[int]*ProcessMessageDTO
	bufferMutex     sync.Mutex
}

// NewRocketApplicationService crea un nuevo servicio de aplicación
func NewRocketApplicationService(repository domain.RocketRepository, eventStore domain.EventStore) *RocketApplicationService {
	return &RocketApplicationService{
		repository:      repository,
		eventStore:      eventStore,
		pendingMessages: make(map[string]map[int]*ProcessMessageDTO),
	}
}

// ProcessMessageDTO representa un mensaje entrante
type ProcessMessageDTO struct {
	Channel    string `json:"channel"`
	Number     int    `json:"number"`
	Action     string `json:"action"`
	Param      string `json:"param,omitempty"`
	Value      int    `json:"value,omitempty"`
	Time       int64  `json:"time"`
	RocketType string `json:"rocketType,omitempty"`
}

// ProcessMessage procesa un mensaje y actualiza el cohete con reordenamiento
func (s *RocketApplicationService) ProcessMessage(dto *ProcessMessageDTO) error {
	s.bufferMutex.Lock()
	defer s.bufferMutex.Unlock()

	// Obtener el último messageNumber esperado
	channel, err := domain.NewChannel(dto.Channel)
	if err != nil {
		return fmt.Errorf("invalid channel: %w", err)
	}

	rocket, err := s.repository.GetByChannel(channel)
	if err != nil {
		return fmt.Errorf("failed to get rocket: %w", err)
	}

	expected := rocket.GetLastMessageNumber().Value() + 1

	log.Printf("[DEBUG] Channel: %s | Received msg#%d | Expected msg#%d", dto.Channel, dto.Number, expected)

	// Si es el mensaje esperado, procesarlo
	if dto.Number == expected {
		log.Printf("[PROCESS] Channel: %s | Processing msg#%d (action: %s)", dto.Channel, dto.Number, dto.Action)
		if err := s.processMessageDirect(dto); err != nil {
			return err
		}

		// Procesar mensajes consecutivos del buffer
		for {
			nextNum := expected + 1
			if s.pendingMessages[dto.Channel] == nil {
				break
			}
			nextDTO := s.pendingMessages[dto.Channel][nextNum]
			if nextDTO == nil {
				break
			}

			log.Printf("[BUFFER] Channel: %s | Processing buffered msg#%d (action: %s)", dto.Channel, nextNum, nextDTO.Action)
			if err := s.processMessageDirect(nextDTO); err != nil {
				return err
			}
			delete(s.pendingMessages[dto.Channel], nextNum)
			expected = nextNum
		}

		return nil
	}

	// Si es un mensaje futuro, guardarlo en el buffer
	if dto.Number > expected {
		if s.pendingMessages[dto.Channel] == nil {
			s.pendingMessages[dto.Channel] = make(map[int]*ProcessMessageDTO)
		}
		s.pendingMessages[dto.Channel][dto.Number] = dto
		log.Printf("[BUFFER] Channel: %s | Message #%d stored in buffer (waiting for #%d)", dto.Channel, dto.Number, expected)
		log.Printf("[BUFFER] Channel: %s | Buffered messages: %v", dto.Channel, s.getBufferedMessageNumbers(dto.Channel))
		return nil // No es error, solo está esperando
	}

	// Si es un mensaje viejo o duplicado, rechazar
	log.Printf("[REJECT] Channel: %s | Message #%d rejected (already processed, expected #%d)", dto.Channel, dto.Number, expected)
	return fmt.Errorf("message %d already processed (expected %d)", dto.Number, expected)
}

// processMessageDirect procesa un mensaje directamente (sin buffer)
func (s *RocketApplicationService) processMessageDirect(dto *ProcessMessageDTO) error {
	// Validar y crear value objects
	channel, err := domain.NewChannel(dto.Channel)
	if err != nil {
		return fmt.Errorf("invalid channel: %w", err)
	}

	msgNum, err := domain.NewMessageNumber(dto.Number)
	if err != nil {
		return fmt.Errorf("invalid message number: %w", err)
	}

	// Obtener o crear cohete
	rocket, err := s.repository.GetByChannel(channel)
	if err != nil {
		return fmt.Errorf("failed to get rocket: %w", err)
	}

	// Procesar acción
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

	// Guardar cambios
	return s.repository.Save(rocket)
}

// getBufferedMessageNumbers devuelve los números de mensajes en el buffer
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

// RocketDTO representa el estado de un cohete
type RocketDTO struct {
	Channel string `json:"channel"`
	Type    string `json:"type"`
	Status  string `json:"status"`
	Speed   int    `json:"speed"`
	Mission string `json:"mission"`
}

// EventDTO representa un evento para exponer por API
type EventDTO struct {
	Type          string `json:"type"`
	MessageNumber int    `json:"messageNumber"`
	Timestamp     int64  `json:"timestamp"`
	Details       string `json:"details,omitempty"`
}

// GetRocket obtiene el estado actual de un cohete
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

// ListRockets obtiene todos los cohetes
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

// ListEvents obtiene los eventos de un canal (ordenados por llegada en el store)
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

// BufferStatusDTO representa el estado del buffer para debug
type BufferStatusDTO struct {
	Channel          string `json:"channel"`
	ExpectedNext     int    `json:"expectedNext"`
	BufferedMessages []int  `json:"bufferedMessages"`
}

// GetBufferStatus devuelve el estado del buffer para todas las channels
func (s *RocketApplicationService) GetBufferStatus() []*BufferStatusDTO {
	s.bufferMutex.Lock()
	defer s.bufferMutex.Unlock()

	var status []*BufferStatusDTO
	for channel, messages := range s.pendingMessages {
		// Obtener último mensaje procesado
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
