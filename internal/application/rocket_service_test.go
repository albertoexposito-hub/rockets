package application

import (
	"testing"

	"rockets/internal/infrastructure"
)

func setupTestService() *RocketApplicationService {
	eventStore := infrastructure.NewKafkaEventStore("localhost:9092")
	repository := infrastructure.NewRocketRepository("localhost:6379", eventStore)
	return NewRocketApplicationService(repository, eventStore)
}

// TestProcessMessageInOrder verifies that in-order messages are processed without errors.
// Expected result: message #1 launches the rocket, message #2 increases speed to 20000.
func TestProcessMessageInOrder(t *testing.T) {
	// Arrange
	service := setupTestService()

	msg1 := &ProcessMessageDTO{
		Channel:    "test-rocket-1",
		Number:     1,
		Action:     "launch",
		RocketType: "Falcon-9",
		Value:      15000,
		Param:      "exploration",
		Time:       1234567890,
	}

	msg2 := &ProcessMessageDTO{
		Channel: "test-rocket-1",
		Number:  2,
		Action:  "increase_speed",
		Value:   5000,
		Time:    1234567891,
	}

	// Act
	err1 := service.ProcessMessage(msg1)
	err2 := service.ProcessMessage(msg2)

	// Assert
	if err1 != nil {
		t.Fatalf("Expected no error for msg1, got %v", err1)
	}
	if err2 != nil {
		t.Fatalf("Expected no error for msg2, got %v", err2)
	}

	// Verify final state
	rocket, _ := service.GetRocket("test-rocket-1")
	if rocket.Speed != 20000 {
		t.Errorf("Expected speed 20000, got %d", rocket.Speed)
	}
}

// TestProcessMessageOutOfOrder verifies that out-of-order messages are buffered and processed correctly.
// Send: msg#1, msg#3, msg#2 (out of order)
// Expected result: msg#3 is buffered; when msg#2 arrives both are processed (2 and 3).
// Velocidad final: 15000 + 5000 - 2000 = 18000.
func TestProcessMessageOutOfOrder(t *testing.T) {
	// Arrange
	service := setupTestService()

	msg1 := &ProcessMessageDTO{
		Channel:    "test-rocket-2",
		Number:     1,
		Action:     "launch",
		RocketType: "Falcon-9",
		Value:      15000,
		Param:      "exploration",
		Time:       1234567890,
	}

	msg3 := &ProcessMessageDTO{
		Channel: "test-rocket-2",
		Number:  3,
		Action:  "decrease_speed",
		Value:   2000,
		Time:    1234567892,
	}

	msg2 := &ProcessMessageDTO{
		Channel: "test-rocket-2",
		Number:  2,
		Action:  "increase_speed",
		Value:   5000,
		Time:    1234567891,
	}

	// Act - Send out of order: 1, 3, 2
	err1 := service.ProcessMessage(msg1)
	err3 := service.ProcessMessage(msg3) // Should buffer
	err2 := service.ProcessMessage(msg2) // Should process 2 and then 3

	// Assert
	if err1 != nil {
		t.Fatalf("Expected no error for msg1, got %v", err1)
	}
	if err3 != nil {
		t.Fatalf("Expected no error for msg3 (buffered), got %v", err3)
	}
	if err2 != nil {
		t.Fatalf("Expected no error for msg2, got %v", err2)
	}

	// Verify final state (should have processed all 3)
	rocket, _ := service.GetRocket("test-rocket-2")
	expectedSpeed := 15000 + 5000 - 2000 // 18000
	if rocket.Speed != expectedSpeed {
		t.Errorf("Expected speed %d, got %d", expectedSpeed, rocket.Speed)
	}
}

// TestProcessDuplicateMessage verifies that duplicate messages are rejected.
// Send the same message #1 twice (at-least-once delivery guarantee).
// Expected result: first message OK, second message rejected with error.
func TestProcessDuplicateMessage(t *testing.T) {
	// Arrange
	service := setupTestService()

	msg1 := &ProcessMessageDTO{
		Channel:    "test-rocket-3",
		Number:     1,
		Action:     "launch",
		RocketType: "Falcon-9",
		Value:      15000,
		Param:      "exploration",
		Time:       1234567890,
	}

	msg1Duplicate := &ProcessMessageDTO{
		Channel:    "test-rocket-3",
		Number:     1,
		Action:     "launch",
		RocketType: "Falcon-9",
		Value:      15000,
		Param:      "exploration",
		Time:       1234567890,
	}

	// Act
	err1 := service.ProcessMessage(msg1)
	err2 := service.ProcessMessage(msg1Duplicate)

	// Assert
	if err1 != nil {
		t.Fatalf("Expected no error for first message, got %v", err1)
	}
	if err2 == nil {
		t.Error("Expected error for duplicate message, got nil")
	}
}

// TestProcessMessageWithLargeGap verifies that messages with large gaps are buffered correctly.
// Send msg#1 and then msg#100 (gap of 98 messages).
// Expected result: msg#100 remains in buffer waiting for messages 2-99.
// The rocket should NOT be exploded because msg#100 has not been applied yet.
func TestProcessMessageWithLargeGap(t *testing.T) {
	// Arrange
	service := setupTestService()

	msg1 := &ProcessMessageDTO{
		Channel:    "test-rocket-4",
		Number:     1,
		Action:     "launch",
		RocketType: "Falcon-9",
		Value:      15000,
		Param:      "exploration",
		Time:       1234567890,
	}

	msg100 := &ProcessMessageDTO{
		Channel: "test-rocket-4",
		Number:  100,
		Action:  "explode",
		Param:   "alien attack",
		Time:    1234567990,
	}

	// Act
	err1 := service.ProcessMessage(msg1)
	err100 := service.ProcessMessage(msg100)

	// Assert
	if err1 != nil {
		t.Fatalf("Expected no error for msg1, got %v", err1)
	}
	if err100 != nil {
		t.Fatalf("Expected no error for msg100 (buffered), got %v", err100)
	}

	// Verify msg100 is buffered (not applied yet)
	rocket, _ := service.GetRocket("test-rocket-4")
	if rocket.Status == "exploded" {
		t.Error("msg100 should be buffered, not applied yet")
	}
}

// TestMultipleRocketsSimultaneously verifies that multiple rockets are processed independently.
// Launch rocket-A (Falcon-9, 15000) and rocket-B (Starship, 20000) simultaneously.
// Resultado esperado: ambos cohetes existen con sus velocidades correctas.
// Demonstrates that the buffer is independent per channel.
func TestMultipleRocketsSimultaneously(t *testing.T) {
	// Arrange
	service := setupTestService()

	rocketAMsg1 := &ProcessMessageDTO{
		Channel:    "rocket-A",
		Number:     1,
		Action:     "launch",
		RocketType: "Falcon-9",
		Value:      15000,
		Param:      "mars",
		Time:       1234567890,
	}

	rocketBMsg1 := &ProcessMessageDTO{
		Channel:    "rocket-B",
		Number:     1,
		Action:     "launch",
		RocketType: "Starship",
		Value:      20000,
		Param:      "moon",
		Time:       1234567890,
	}

	// Act
	err1 := service.ProcessMessage(rocketAMsg1)
	err2 := service.ProcessMessage(rocketBMsg1)

	// Assert
	if err1 != nil {
		t.Fatalf("Expected no error for rocket-A, got %v", err1)
	}
	if err2 != nil {
		t.Fatalf("Expected no error for rocket-B, got %v", err2)
	}

	// Verify both rockets exist
	rocketA, _ := service.GetRocket("rocket-A")
	rocketB, _ := service.GetRocket("rocket-B")

	if rocketA.Speed != 15000 {
		t.Errorf("Expected rocket-A speed 15000, got %d", rocketA.Speed)
	}
	if rocketB.Speed != 20000 {
		t.Errorf("Expected rocket-B speed 20000, got %d", rocketB.Speed)
	}
}

// TestBufferReprocessing verifies that the buffer reorders and processes messages correctly.
// Send messages: 1, 4, 2, 3 (out of order)
// Expected result: processed in order: 1→2→3→4
// Estado final: launch(10000) + increase(2000) - decrease(1000) + explode = 11000, exploded.
func TestBufferReprocessing(t *testing.T) {
	// Arrange
	service := setupTestService()

	// Send messages: 1, 4, 2, 3
	msgs := []*ProcessMessageDTO{
		{Channel: "rocket-buffer", Number: 1, Action: "launch", RocketType: "Falcon-9", Value: 10000, Param: "test", Time: 100},
		{Channel: "rocket-buffer", Number: 4, Action: "explode", Param: "test", Time: 400},
		{Channel: "rocket-buffer", Number: 2, Action: "increase_speed", Value: 2000, Time: 200},
		{Channel: "rocket-buffer", Number: 3, Action: "decrease_speed", Value: 1000, Time: 300},
	}

	// Act
	for _, msg := range msgs {
		if err := service.ProcessMessage(msg); err != nil {
			t.Fatalf("Expected no error processing message, got %v", err)
		}
	}

	// Assert - All should be processed in correct order
	rocket, _ := service.GetRocket("rocket-buffer")

	// Final state: launched(10000) + increase(2000) - decrease(1000) + explode
	expectedSpeed := 10000 + 2000 - 1000 // 11000
	if rocket.Speed != expectedSpeed {
		t.Errorf("Expected speed %d, got %d", expectedSpeed, rocket.Speed)
	}
	if rocket.Status != "exploded" {
		t.Errorf("Expected status exploded, got %s", rocket.Status)
	}
}
