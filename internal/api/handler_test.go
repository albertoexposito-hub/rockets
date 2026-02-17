package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"rockets/internal/application"
	"rockets/internal/infrastructure"
)

func setupTestServer() (*application.WorkerPool, *application.RocketApplicationService) {
	eventStore := infrastructure.NewKafkaEventStore("localhost:9092")
	repository := infrastructure.NewRocketRepository(eventStore)
	service := application.NewRocketApplicationService(repository, eventStore)

	ctx := context.Background()
	pool := application.NewWorkerPool(service, 3)
	pool.Start(ctx)

	return pool, service
}

// TestHandleMessagesValidLaunch verifies that POST /messages endpoint accepts a valid message.
func TestHandleMessagesValidLaunch(t *testing.T) {
	// Arrange
	pool, _ := setupTestServer()
	handler := HandleMessages(pool)

	payload := map[string]interface{}{
		"metadata": map[string]interface{}{
			"channel":       "rocket-test-1",
			"messageNumber": 1,
			"messageTime":   "2024-01-01T10:00:00Z",
			"messageType":   "RocketLaunched",
		},
		"message": map[string]interface{}{
			"type":        "Falcon-9",
			"launchSpeed": float64(15000),
			"mission":     "exploration",
		},
	}

	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/messages", bytes.NewReader(body))
	w := httptest.NewRecorder()

	// Act
	handler(w, req)

	// Assert
	if w.Code != http.StatusAccepted {
		t.Errorf("Expected status 202, got %d", w.Code)
	}

	// Wait for processing
	time.Sleep(100 * time.Millisecond)
}

// TestHandleMessagesInvalidMethod verifies that only POST method is accepted.
func TestHandleMessagesInvalidMethod(t *testing.T) {
	// Arrange
	pool, _ := setupTestServer()
	handler := HandleMessages(pool)

	req := httptest.NewRequest(http.MethodGet, "/messages", nil)
	w := httptest.NewRecorder()

	// Act
	handler(w, req)

	// Assert
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405, got %d", w.Code)
	}
}

// TestHandleMessagesInvalidJSON verifies that messages with invalid JSON are rejected.
func TestHandleMessagesInvalidJSON(t *testing.T) {
	// Arrange
	pool, _ := setupTestServer()
	handler := HandleMessages(pool)

	req := httptest.NewRequest(http.MethodPost, "/messages", bytes.NewReader([]byte("invalid json")))
	w := httptest.NewRecorder()

	// Act
	handler(w, req)

	// Assert
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

// TestHandleListRocketsGetAll verifies that GET /rockets returns all rockets.
// Creates 2 rockets (rocket-list-1 and rocket-list-2).
// Expected result: HTTP 200 OK, list with at least 2 rockets.
func TestHandleListRocketsGetAll(t *testing.T) {
	// Arrange
	_, service := setupTestServer()

	// Send some messages first
	msg1 := &application.ProcessMessageDTO{
		Channel:    "rocket-list-1",
		Number:     1,
		Action:     "launch",
		RocketType: "Falcon-9",
		Value:      15000,
		Param:      "mars",
		Time:       1234567890,
	}
	msg2 := &application.ProcessMessageDTO{
		Channel:    "rocket-list-2",
		Number:     1,
		Action:     "launch",
		RocketType: "Starship",
		Value:      20000,
		Param:      "moon",
		Time:       1234567890,
	}

	if err := service.ProcessMessage(msg1); err != nil {
		t.Fatalf("Expected no error message 1, got %v", err)
	}
	if err := service.ProcessMessage(msg2); err != nil {
		t.Fatalf("Expected no error message2, got %v", err)
	}

	handler := HandleListRockets(service)
	req := httptest.NewRequest(http.MethodGet, "/rockets", nil)
	w := httptest.NewRecorder()

	// Act
	handler(w, req)

	// Assert
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var rockets []application.RocketDTO
	if err := json.Unmarshal(w.Body.Bytes(), &rockets); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if len(rockets) < 2 {
		t.Errorf("Expected at least 2 rockets, got %d", len(rockets))
	}
}

// TestHandleListRocketsGetByChannel verifies that GET /rockets/{channel} returns a specific rocket.
// Expected result: HTTP 200 OK, rocket data (channel=rocket-specific, type=Falcon-9).
func TestHandleListRocketsGetByChannel(t *testing.T) {
	// Arrange
	_, service := setupTestServer()

	msg := &application.ProcessMessageDTO{
		Channel:    "rocket-specific",
		Number:     1,
		Action:     "launch",
		RocketType: "Falcon-9",
		Value:      15000,
		Param:      "exploration",
		Time:       1234567890,
	}

	if err := service.ProcessMessage(msg); err != nil {
		t.Fatalf("Expected no error processing message, got %v", err)
	}
	time.Sleep(50 * time.Millisecond)

	handler := HandleListRockets(service)
	req := httptest.NewRequest(http.MethodGet, "/rockets/rocket-specific", nil)
	w := httptest.NewRecorder()

	// Act
	handler(w, req)

	// Assert
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var rocket application.RocketDTO
	if err := json.Unmarshal(w.Body.Bytes(), &rocket); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if rocket.Channel != "rocket-specific" {
		t.Errorf("Expected channel rocket-specific, got %s", rocket.Channel)
	}
	if rocket.Type != "Falcon-9" {
		t.Errorf("Expected type Falcon-9, got %s", rocket.Type)
	}
}
