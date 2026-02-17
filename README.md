# Rockets Event Sourcing Server

Event Sourcing backend for rocket state management built with Go. This is a **single‑process, in‑memory** implementation with no external dependencies.

## Requirements

- Go 1.25+

## Quick Start

```bash
# Using Makefile
make run

# Or directly
go run cmd/server/main.go

# Custom worker count
WORKER_COUNT=5 make run
```

Server starts on http://localhost:8088

## Features

- In‑memory event store (no database, no Kafka, no Redis)
- Out‑of‑order buffering per channel
- Worker pool for parallel processing (default: 3)
- Event replay for current rocket state

## API

### POST /messages

Accepted format (RFC3339/RFC3339Nano `messageTime`):

```bash
curl -X POST http://localhost:8088/messages \
  -H 'Content-Type: application/json' \
  -d '{
    "metadata": {
      "channel": "rocket-alpha",
      "messageNumber": 1,
      "messageTime": "2026-01-22T12:00:00Z",
      "messageType": "RocketLaunched"
    },
    "message": {
      "type": "Falcon-9",
      "mission": "Mars Mission",
      "launchSpeed": 25000
    }
  }'
```

Supported `messageType` and payload fields:

- `RocketLaunched`: `type`, `mission`, `launchSpeed`
- `RocketSpeedIncreased`: `by`
- `RocketSpeedDecreased`: `by`
- `RocketMissionChanged`: `newMission`
- `RocketExploded`: `reason`

Notes:

- If `channel` is empty, it is auto‑generated.
- If `messageNumber` ≤ 0, it is auto‑generated.
- If `messageTime` is invalid/missing, current time is used.

### GET /rockets

```bash
curl http://localhost:8088/rockets
```

### GET /rockets/{channel}

```bash
curl http://localhost:8088/rockets/rocket-alpha
```

### GET /rockets/{channel}/events

```bash
curl http://localhost:8088/rockets/rocket-alpha/events
```

### GET /health

```bash
curl http://localhost:8088/health
```

## How it Works

```
POST /messages → Worker Pool → Reordering Buffer → Aggregate.Apply(event) → Event Store
GET /rockets   → Replay events → Current state
```

Out‑of‑order messages are buffered per channel and applied when gaps are filled. This is in‑memory and **not** safe for multiple instances.

## Testing

```bash
make test
```

## Project Structure

```
rockets/
├── cmd/server/          # Application entry point
│   └── main.go
├── internal/
│   ├── api/            # HTTP handlers
│   ├── application/    # Use cases & DTOs
│   ├── domain/         # Aggregates, events, value objects
│   └── infrastructure/ # Event store, repository
├── Makefile            # Build commands
├── go.mod
└── README.md
```

## License

MIT
