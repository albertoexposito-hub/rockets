package domain

import (
	"testing"
)

// TestRocketLaunch verifica que un cohete se lance correctamente con sus propiedades iniciales.
// Resultado esperado: status=flying, speed=15000, mission=exploration, type=Falcon-9.
func TestRocketLaunch(t *testing.T) {
	// Arrange
	channel, _ := NewChannel("rocket-1")
	rocket := NewRocket(channel)
	msgNum, _ := NewMessageNumber(1)
	speed, _ := NewSpeed(15000)
	mission := NewMission("exploration")

	// Act
	err := rocket.Launch(msgNum, "Falcon-9", speed, mission, 1234567890)

	// Assert
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if rocket.GetStatus() != StatusFlying {
		t.Errorf("Expected status Flying, got %v", rocket.GetStatus())
	}
	if rocket.GetSpeed().Value() != 15000 {
		t.Errorf("Expected speed 15000, got %d", rocket.GetSpeed().Value())
	}
	if rocket.GetMission() != "exploration" {
		t.Errorf("Expected mission exploration, got %s", rocket.GetMission())
	}
	if rocket.rocketType != "Falcon-9" {
		t.Errorf("Expected type Falcon-9, got %s", rocket.rocketType)
	}
}

// TestRocketCannotLaunchTwice verifica que un cohete no pueda ser lanzado dos veces.
// Resultado esperado: primer lanzamiento OK, segundo lanzamiento retorna error.
func TestRocketCannotLaunchTwice(t *testing.T) {
	// Arrange
	channel, _ := NewChannel("rocket-1")
	rocket := NewRocket(channel)
	msgNum1, _ := NewMessageNumber(1)
	msgNum2, _ := NewMessageNumber(2)
	speed, _ := NewSpeed(15000)
	mission := NewMission("exploration")

	rocket.Launch(msgNum1, "Falcon-9", speed, mission, 1234567890)

	// Act
	err := rocket.Launch(msgNum2, "Falcon-9", speed, mission, 1234567891)

	// Assert
	if err == nil {
		t.Error("Expected error when launching twice, got nil")
	}
}

// TestRocketIncreaseSpeed verifica que la velocidad del cohete se incremente correctamente.
// Resultado esperado: velocidad inicial 15000 + incremento 5000 = 20000.
func TestRocketIncreaseSpeed(t *testing.T) {
	// Arrange
	channel, _ := NewChannel("rocket-1")
	rocket := NewRocket(channel)
	msgNum1, _ := NewMessageNumber(1)
	msgNum2, _ := NewMessageNumber(2)
	speed, _ := NewSpeed(15000)
	mission := NewMission("exploration")

	rocket.Launch(msgNum1, "Falcon-9", speed, mission, 1234567890)

	// Act
	err := rocket.IncreaseSpeed(msgNum2, 5000, 1234567891)

	// Assert
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if rocket.GetSpeed().Value() != 20000 {
		t.Errorf("Expected speed 20000, got %d", rocket.GetSpeed().Value())
	}
}

// TestRocketDecreaseSpeed verifica que la velocidad del cohete se decremente correctamente.
// Resultado esperado: velocidad inicial 15000 - decremento 3000 = 12000.
func TestRocketDecreaseSpeed(t *testing.T) {
	// Arrange
	channel, _ := NewChannel("rocket-1")
	rocket := NewRocket(channel)
	msgNum1, _ := NewMessageNumber(1)
	msgNum2, _ := NewMessageNumber(2)
	speed, _ := NewSpeed(15000)
	mission := NewMission("exploration")

	rocket.Launch(msgNum1, "Falcon-9", speed, mission, 1234567890)

	// Act
	err := rocket.DecreaseSpeed(msgNum2, 3000, 1234567891)

	// Assert
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if rocket.GetSpeed().Value() != 12000 {
		t.Errorf("Expected speed 12000, got %d", rocket.GetSpeed().Value())
	}
}

// TestRocketExplode verifica que un cohete pueda explotar y cambiar su estado.
// Resultado esperado: status cambia de flying a exploded.
func TestRocketExplode(t *testing.T) {
	// Arrange
	channel, _ := NewChannel("rocket-1")
	rocket := NewRocket(channel)
	msgNum1, _ := NewMessageNumber(1)
	msgNum2, _ := NewMessageNumber(2)
	speed, _ := NewSpeed(15000)
	mission := NewMission("exploration")

	rocket.Launch(msgNum1, "Falcon-9", speed, mission, 1234567890)

	// Act
	err := rocket.Explode(msgNum2, "fuel leak", 1234567891)

	// Assert
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if rocket.GetStatus() != StatusExploded {
		t.Errorf("Expected status Exploded, got %v", rocket.GetStatus())
	}
}

// TestRocketCannotChangeAfterExplosion verifica que un cohete explotado no pueda ser modificado.
// Resultado esperado: intentar incrementar velocidad después de explotar retorna error.
func TestRocketCannotChangeAfterExplosion(t *testing.T) {
	// Arrange
	channel, _ := NewChannel("rocket-1")
	rocket := NewRocket(channel)
	msgNum1, _ := NewMessageNumber(1)
	msgNum2, _ := NewMessageNumber(2)
	msgNum3, _ := NewMessageNumber(3)
	speed, _ := NewSpeed(15000)
	mission := NewMission("exploration")

	rocket.Launch(msgNum1, "Falcon-9", speed, mission, 1234567890)
	rocket.Explode(msgNum2, "fuel leak", 1234567891)

	// Act
	err := rocket.IncreaseSpeed(msgNum3, 1000, 1234567892)

	// Assert
	if err == nil {
		t.Error("Expected error when changing exploded rocket, got nil")
	}
}

// TestRocketLoadFromHistory verifica que un cohete se pueda reconstruir desde eventos históricos.
// Carga 2 eventos: RocketLaunched (15000) y RocketSpeedIncreased (+5000).
// Resultado esperado: velocidad final 20000, último mensaje #2 (event sourcing replay).
func TestRocketLoadFromHistory(t *testing.T) {
	// Arrange
	channel, _ := NewChannel("rocket-1")
	msgNum1, _ := NewMessageNumber(1)
	msgNum2, _ := NewMessageNumber(2)
	speed, _ := NewSpeed(15000)

	events := []DomainEvent{
		&RocketLaunched{
			Channel:       channel,
			MessageNumber: msgNum1,
			Type:          "Falcon-9",
			Speed:         speed,
			Mission:       "exploration",
			Timestamp:     1234567890,
		},
		&RocketSpeedIncreased{
			Channel:       channel,
			MessageNumber: msgNum2,
			OldSpeed:      speed,
			NewSpeed:      &Speed{value: 20000},
			Delta:         5000,
			Timestamp:     1234567891,
		},
	}

	// Act
	rocket := NewRocket(channel)
	err := rocket.LoadFromHistory(events)

	// Assert
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if rocket.GetSpeed().Value() != 20000 {
		t.Errorf("Expected speed 20000, got %d", rocket.GetSpeed().Value())
	}
	if rocket.GetLastMessageNumber().Value() != 2 {
		t.Errorf("Expected last message 2, got %d", rocket.GetLastMessageNumber().Value())
	}
}
