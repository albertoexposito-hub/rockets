package api

import (
	"encoding/json"
	"fmt"
	"log/slog"
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

// from message to internal ProcessMessageDTO
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

// HandleMessages  POST /messages
func HandleMessages(pool *application.WorkerPool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Try to parse as official challenge format
		var lunarMsg LunarMessage
		if err := json.NewDecoder(r.Body).Decode(&lunarMsg); err != nil {
			http.Error(w, "Invalid request format", http.StatusBadRequest)
			return
		}

		slog.Info("Received message",
			"channel", lunarMsg.Metadata.Channel,
			"number", lunarMsg.Metadata.MessageNumber,
			"type", lunarMsg.Metadata.MessageType)

		// Convert to internal format
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
		// (should not happen in official format)
		// it can break ordering otherwise
		if dto.Number <= 0 {
			dto.Number = int(getNextMessageNumber())
		}

		// If time is invalid, use current time
		// never in production, only for tests
		if dto.Time <= 0 {
			dto.Time = time.Now().UnixMilli()
		}

		slog.Debug("Enqueueing to worker pool",
			"channel", dto.Channel,
			"number", dto.Number,
			"action", dto.Action)

		if err := pool.Enqueue(dto); err != nil {
			slog.Error("Failed to enqueue",
				"channel", dto.Channel,
				"number", dto.Number,
				"err", err)
			http.Error(w, err.Error(), http.StatusServiceUnavailable)
			return
		}

		slog.Info("Message queued successfully",
			"channel", dto.Channel,
			"number", dto.Number)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		if err := json.NewEncoder(w).Encode(map[string]string{"status": "queued"}); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}

// HandleListRockets  GET /rockets
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

		// List all rockets
		rockets, err := service.ListRockets()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Sort by channel
		sort.Slice(rockets, func(i, j int) bool {
			return rockets[i].Channel < rockets[j].Channel
		})

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(rockets); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}

// HandleDebugBuffer shows the messages in the buffer
// not mandatory but good to have for debugging
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
