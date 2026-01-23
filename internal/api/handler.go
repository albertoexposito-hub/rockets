package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"rockets/internal/application"
)

type LunarMessage struct {
	Metadata struct {
		Channel       string `json:"channel"`
		MessageNumber int    `json:"messageNumber"`
		MessageTime   string `json:"messageTime"`
		MessageType   string `json:"messageType"`
	} `json:"metadata"`
	Message map[string]interface{} `json:"message"`
}

// convertLunarMessageToDTO convierte el formato oficial al internal ProcessMessageDTO
func convertLunarMessageToDTO(msg *LunarMessage) (*application.ProcessMessageDTO, error) {
	// Parse messageTime (ISO8601) to Unix milliseconds
	t, err := time.Parse(time.RFC3339Nano, msg.Metadata.MessageTime)
	if err != nil {
		// Try parsing with simpler format
		t, err = time.Parse(time.RFC3339, msg.Metadata.MessageTime)
		if err != nil {
			return nil, fmt.Errorf("invalid messageTime: %w", err)
		}
	}
	timestamp := t.UnixMilli()

	dto := &application.ProcessMessageDTO{
		Channel: msg.Metadata.Channel,
		Number:  msg.Metadata.MessageNumber,
		Time:    timestamp,
	}

	// Map messageType to action and extract parameters
	switch msg.Metadata.MessageType {
	case "RocketLaunched":
		dto.Action = "launch"
		if rocketType, ok := msg.Message["type"]; ok {
			if v, ok := rocketType.(string); ok {
				dto.RocketType = v
			}
		}
		if mission, ok := msg.Message["mission"]; ok {
			dto.Param = mission.(string)
		}
		if speed, ok := msg.Message["launchSpeed"]; ok {
			if v, ok := speed.(float64); ok {
				dto.Value = int(v)
			}
		}

	case "RocketSpeedIncreased":
		dto.Action = "increase_speed"
		if by, ok := msg.Message["by"]; ok {
			if v, ok := by.(float64); ok {
				dto.Value = int(v)
			}
		}

	case "RocketSpeedDecreased":
		dto.Action = "decrease_speed"
		if by, ok := msg.Message["by"]; ok {
			if v, ok := by.(float64); ok {
				dto.Value = int(v)
			}
		}

	case "RocketExploded":
		dto.Action = "explode"
		if reason, ok := msg.Message["reason"]; ok {
			dto.Param = reason.(string)
		}

	case "RocketMissionChanged":
		dto.Action = "change_mission"
		if mission, ok := msg.Message["newMission"]; ok {
			dto.Param = mission.(string)
		}

	default:
		return nil, fmt.Errorf("unknown messageType: %s", msg.Metadata.MessageType)
	}

	return dto, nil
}

// Global counter to generate unique message numbers
var (
	messageCounter int64
	counterMutex   sync.Mutex
)

// getNextMessageNumber generates the next message number
func getNextMessageNumber() int64 {
	counterMutex.Lock()
	defer counterMutex.Unlock()
	messageCounter++
	return messageCounter
}

// HandleMessages maneja las solicitudes POST /messages
func HandleMessages(pool *application.WorkerPool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Intentar parsear como formato oficial del challenge
		var lunarMsg LunarMessage
		if err := json.NewDecoder(r.Body).Decode(&lunarMsg); err != nil {
			http.Error(w, "Invalid request format", http.StatusBadRequest)
			return
		}

		log.Printf("[HTTP] ← Received message | Channel: %s | Msg#%d | Type: %s",
			lunarMsg.Metadata.Channel, lunarMsg.Metadata.MessageNumber, lunarMsg.Metadata.MessageType)

		// Convertir al formato interno
		dto, err := convertLunarMessageToDTO(&lunarMsg)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// If channel is empty, generate one automatically
		if dto.Channel == "" {
			dto.Channel = fmt.Sprintf("rocket-%d", time.Now().UnixNano())
		}

		// If message number is invalid, generate one automatically
		if dto.Number <= 0 {
			dto.Number = int(getNextMessageNumber())
		}

		// If time is invalid, use current time
		if dto.Time <= 0 {
			dto.Time = time.Now().UnixMilli()
		}

		log.Printf("[HTTP] → Enqueueing to worker pool | Channel: %s | Msg#%d | Action: %s",
			dto.Channel, dto.Number, dto.Action)

		if err := pool.Enqueue(dto); err != nil {
			log.Printf("[HTTP] ✗ Failed to enqueue | Channel: %s | Msg#%d | Error: %v",
				dto.Channel, dto.Number, err)
			http.Error(w, err.Error(), http.StatusServiceUnavailable)
			return
		}

		log.Printf("[HTTP] ✓ Message queued successfully | Channel: %s | Msg#%d", dto.Channel, dto.Number)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		if err := json.NewEncoder(w).Encode(map[string]string{"status": "queued"}); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}

// HandleListRockets maneja las solicitudes GET /rockets
func HandleListRockets(service *application.RocketApplicationService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		if strings.HasPrefix(r.URL.Path, "/rockets/") {
			trimmed := strings.TrimPrefix(r.URL.Path, "/rockets/")

			if strings.HasSuffix(trimmed, "/events") {
				channel := strings.TrimSuffix(trimmed, "/events")
				channel = strings.TrimSuffix(channel, "/")
				if channel == "" {
					http.Error(w, "Not found", http.StatusNotFound)
					return
				}

				events, err := service.ListEvents(channel)
				if err != nil {
					http.Error(w, "Not found", http.StatusNotFound)
					return
				}

				w.Header().Set("Content-Type", "application/json")
				if err := json.NewEncoder(w).Encode(events); err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
				}
				return
			}

			channel := strings.TrimSuffix(trimmed, "/")
			if channel == "" {
				http.Error(w, "Not found", http.StatusNotFound)
				return
			}

			rocket, err := service.GetRocket(channel)
			if err != nil {
				http.Error(w, "Not found", http.StatusNotFound)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			if err := json.NewEncoder(w).Encode(rocket); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
			return
		}

		// Listar todos los cohetes
		rockets, err := service.ListRockets()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Ordenar por canal
		sort.Slice(rockets, func(i, j int) bool {
			return rockets[i].Channel < rockets[j].Channel
		})

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(rockets); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}

// HandleDebugBuffer muestra los mensajes en el buffer
func HandleDebugBuffer(service *application.RocketApplicationService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		buffer := service.GetBufferStatus()
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(buffer); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}
