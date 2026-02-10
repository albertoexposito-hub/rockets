# Rockets Event Sourcing Server

An Event Sourcing backend for rocket state management built with Go. Implements Domain-Driven Design principles with automatic out-of-order message buffering and reordering.

## ⚠️ Important Note: In-Memory Implementation

**This implementation uses 100% in-memory storage. There is NO real database (not PostgreSQL), NO real Kafka, NO Redis.** Everything is simulated in RAM using Go maps and goroutines.

### Current Limitations:
- ❌ **NOT horizontally scalable** - Cannot run multiple instances (no shared state)
- ❌ **NOT persistent** - All data lost on server restart
- ❌ **Single-pod only** - Works perfectly for one process, breaks with multiple pods
- ❌ **Memory growth** - Buffered messages stay in RAM indefinitely

### Why This Approach:
- ✅ Meets all challenge requirements
- ✅ Zero external dependencies
- ✅ Fast and simple
- ✅ Perfect for learning/interviews

### For Production:
See **Scaling Considerations** section below for the PostgreSQL + Redis + Kafka architecture needed for real deployments.

## Features

- ✅ **Event Sourcing**: All state changes are persisted as immutable events
- ✅ **Message Reordering**: Automatic buffering and reordering of out-of-order messages
- ✅ **Worker Pool**: Configurable parallel message processing (default: 3 workers)
- ✅ **Event Replay**: Complete state reconstruction from event history
- ✅ **In-Memory Store**: Zero external dependencies required

## Quick Start

### Prerequisites

- Go 1.25

### Run Server

```bash
# Using Makefile (recommended)
make run

# Or directly with go run
go run cmd/server/main.go

# With custom worker count
WORKER_COUNT=5 make run
```

Server starts on `http://localhost:8088`

### API Endpoints

#### POST /messages
Submit rocket events in official challenge format:

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

**Supported messageTypes:**
- `RocketLaunched` - Launch rocket with type, mission, and speed
- `RocketSpeedIncreased` - Increase speed by delta
- `RocketSpeedDecreased` - Decrease speed by delta
- `RocketMissionChanged` - Change mission
- `RocketExploded` - Mark rocket as exploded

#### GET /rockets
List all rockets with current state:

```bash
curl http://localhost:8088/rockets
```

Response:
```json
[
  {
    "channel": "rocket-alpha",
    "type": "Falcon-9",
    "status": "flying",
    "speed": 25000,
    "mission": "Mars Mission"
  }
]
```

#### GET /rockets/{channel}
Get specific rocket state:

```bash
curl http://localhost:8088/rockets/rocket-alpha
```

#### GET /rockets/{channel}/events
View event history for a rocket:

```bash
curl http://localhost:8088/rockets/rocket-alpha/events
```

#### GET /health
Health check endpoint:

```bash
curl http://localhost:8088/health
```

## Architecture

### Event Sourcing Pattern
- **Aggregate**: Rocket entity that enforces business rules
- **Events**: RocketLaunched, RocketSpeedIncreased, RocketSpeedDecreased, RocketExploded, etc.
- **Event Store**: In-memory append-only log
- **Replay**: Complete state reconstruction from event history

### Message Flow

```
POST /messages → Job Queue → Worker Pool (3 workers)
                                  ↓
                          Reordering Buffer
                                  ↓
                          Aggregate.Apply(event)
                                  ↓
                          Event Store (persist)
                                  ↓
            GET /rockets → Replay events → Return current state
```

### Out-of-Order Message Handling

Messages are automatically buffered when they arrive out of sequence:

```
Received order:  1, 4, 2, 3
Buffer behavior: [msg#4 buffered] → [msg#2 arrives → apply #2] 
                 → [msg#3 arrives → apply #3, #4]
Final result:    All messages applied in sequential order (1→2→3→4)
```

## Configuration

Environment variables:

```bash
WORKER_COUNT=5    # Number of parallel workers (default: 3)
```

## Development

### Build Binary

```bash
make build
# Output: bin/rockets-server
```

### Run Tests

```bash
make test
```

### Clean Build Artifacts

```bash
make clean
```

### Makefile Commands

```bash
make run      # Run with go run
make build    # Compile binary
make start    # Build and run binary
make test     # Run tests
make clean    # Remove bin/
make help     # Show all commands
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

## Postman Collection

Import `Rockets.postman_collection.json` for testing:

- Launch rockets (5 examples)
- Query state (all rockets, single rocket)
- View event history
- Health check

## Design Decisions & Trade-offs

### 1. Event Sourcing Pattern
**Decision**: Store all state changes as immutable events rather than storing only the current state.

**Pros**:
- Complete audit trail of all rocket activities
- Ability to replay events and rebuild state at any point in time
- Natural fit for the message-based architecture required by the challenge
- Excellent debuggability (inspect exact sequence of events)

**Cons**:
- More complex than traditional CRUD operations
- Query performance requires event replay (can be mitigated with caching/snapshots)

**Alternative considered**: Simple CRUD updates. **Rejected because**: loses historical data and makes out-of-order message handling significantly more difficult.

---

### 2. In-Memory Buffer for Out-of-Order Messages
**Decision**: Use a `map[channel]map[messageNumber]*Message` structure to buffer future messages until gaps are filled.

**Pros**:
- Automatically handles out-of-order message arrivals
- Simple and efficient implementation
- Guarantees sequential processing per rocket channel
- Thread-safe with mutex protection

**Cons**:
- Potential unbounded memory growth if messages never arrive (e.g., missing msg#5 blocks msg#6+)
- All buffered messages are lost on server restart

**Alternative considered**: Reject out-of-order messages immediately. **Rejected because**: the challenge explicitly requires handling out-of-order delivery.

**Production improvement**: Add TTL to buffered messages (e.g., 1 hour expiration), persist buffer to Redis for crash recovery.

---

### 3. Worker Pool (3 workers by default)
**Decision**: Process messages in parallel using a goroutine pool with a shared job channel.

**Pros**:
- Efficiently handles high-concurrency workloads
- Easily configurable via `WORKER_COUNT` environment variable
- Multiple rockets can be processed simultaneously without blocking
- Leverages Go's native concurrency primitives

**Cons**:
- More complex implementation than sequential processing
- Requires proper mutex synchronization for shared buffer access

**Alternative considered**: Single-threaded sequential processing. **Rejected because**: unacceptable performance for production workloads with multiple concurrent rockets.

---

### 4. In-Memory Storage (No Database)
**Decision**: Store all events and state in Go maps protected by mutexes, with no external database.

**Pros**:
- Zero external dependencies (runs immediately after `go run`)
- Extremely fast reads and writes
- Simple deployment (single binary)
- Fully satisfies the challenge requirements

**Cons**:
- ❌ All data is lost on server restart
- ❌ Cannot scale horizontally (no shared state between pods)
- ❌ No durability guarantees

**Production alternative**: PostgreSQL for persistent event storage + Redis for distributed caching/buffering + Kafka for reliable message queuing. This trade-off was intentionally chosen to prioritize simplicity and ease of evaluation within the challenge timeframe.

---

### 5. Duplicate Message Handling
**Decision**: Reject messages where `messageNumber <= lastProcessedMessageNumber` for each rocket channel.

**Pros**:
- Simple and efficient idempotency check
- Prevents duplicate event application
- Correctly handles at-least-once delivery guarantees
- Zero overhead (single integer comparison)

**Cons**:
- Assumes strictly monotonic message numbers per channel (this assumption holds true for the challenge)

**Alternative considered**: UUID-based deduplication with a seen-messages set. **Rejected because**: message numbers already provide both ordering and deduplication capabilities, making UUIDs redundant.

---

### 6. Domain-Driven Design Structure
**Decision**: Organize code into separate domain, application, and infrastructure layers.

**Pros**:
- Clean architecture with clear separation of concerns
- Easy to swap implementations (e.g., replace in-memory store with PostgreSQL)
- Domain logic completely isolated from HTTP/infrastructure concerns
- Highly testable with dependency injection
- Maintainable and extensible

**Cons**:
- More files and directories to navigate
- Potentially over-engineered for a small proof-of-concept project

**Alternative considered**: Flat structure with all code in `main.go`. **Rejected because**: significantly harder to test, maintain, and extend as requirements grow.

---

### 7. No Database Transactions
**Decision**: Process each message independently without transactional guarantees.

**Pros**:
- Significantly simpler code
- Zero transaction overhead or coordination
- Perfectly acceptable for in-memory storage

**Cons**:
- In a production environment with PostgreSQL, could experience partial failures during concurrent writes

**Production alternative**: Wrap event persistence in database transactions to ensure atomic writes and maintain consistency guarantees.

---

## Testing

```bash
# Run all tests
make test

# Run with coverage
go test -cover ./...

# Run specific test
go test -run TestProcessMessageOutOfOrder ./internal/application/
```

**Comprehensive test coverage includes**:
- ✅ Domain logic and business rules (rocket state transitions, validation)
- ✅ Out-of-order message buffering and automatic reprocessing
- ✅ Duplicate message detectionhttps://meet.google.com/nsz-qdvp-irt and rejection (idempotency)
- ✅ HTTP API endpoints (all routes with success and error cases)https://meet.google.com/nsz-qdvp-irt
- ✅ All message types (Launch, SpeedIncrease, SpeedDecrease, Explode, MissionChange)
- ✅ Concurrent multi-rocket processing with worker pool

**18 tests passing** across domain, application, and API layers.

---

## Scaling Considerations

### Current Implementation (In-Memory)
- ✅ Perfect for single-process development and evaluation
- ✅ Fast and simple to run
- ❌ **NOT horizontally scalable** (cannot run multiple instances)
- ❌ **NOT persistent** (data lost on restart)
- ❌ **Buffered messages never expire** (memory grows unbounded)

### Production Multi-Pod Deployment Strategy

To scale horizontally with multiple server instances:

1. **Replace in-memory event store** → PostgreSQL with JSONB event payloads
   - Append-only table: `events(id, channel, messageNumber, payload, timestamp)`
   - Unique constraint: `(channel, messageNumber)`
   - Enables durability and horizontal reads

2. **Replace in-memory buffer** → Redis for shared message buffering
   - Distributed cache: `pending_messages:{channel}:{msgNum}`
   - Auto-expiration (TTL 1 hour) for lost messages
   - Accessible from any pod

3. **Add message queue** → Kafka for reliable message delivery
   - Topic partitioned by channel (key=channel)
   - Guarantees ordering per partition
   - Consumer group per environment

4. **Implement distributed locking** → Redis locks per channel
   - Prevents concurrent processing of same channel
   - Key: `lock:channel:xxx` with TTL

5. **Add database constraints** → PostgreSQL enforcement
   - Unique constraint on `(channel, messageNumber)` prevents duplicates
   - Foreign key validation for data integrity

6. **Use Kafka partitioning** → Partition by channel for ordering
   - Same channel always routed to same partition
   - Maintains global ordering per rocket

### Production Architecture Diagram

```
┌─────────────────────────────────────────────────────────┐
│                    Load Balancer                         │
└──────────┬──────────┬──────────┬──────────┬──────────────┘
           │          │          │          │
       ┌───▼──┐   ┌───▼──┐   ┌───▼──┐   ┌───▼──┐
       │Pod 1 │   │Pod 2 │   │Pod 3 │   │Pod N │ (Stateless)
       └───┬──┘   └───┬──┘   └───┬──┘   └───┬──┘
           │          │          │          │
       ┌───▼──────────▼──────────▼──────────▼──────┐
       │         Kafka Broker Cluster              │
       │  (Partitioned by channel for ordering)    │
       └───┬──────────────────────────────────────┘
           │
       ┌───▼─────────────────────┐
       │   PostgreSQL Events     │
       │  (Durable Event Store)  │
       └───────────┬─────────────┘
       ┌───────────▼─────────────┐
       │   Redis Cache/Buffer    │
       │  (Distributed locking)  │
       └─────────────────────────┘
```

### Expected Production Improvements

| Aspect | In-Memory | Production |
|--------|-----------|------------|
| **Scalability** | 1 pod only | N pods |
| **Durability** | Lost on restart | Persisted in DB |
| **Throughput** | ~1000 msg/s | ~10,000+ msg/s |
| **Latency** | <1ms | ~5-10ms (with DB) |
| **Availability** | 1 pod = no HA | N pods = high availability |
| **Buffer expiration** | Never | 1 hour TTL |
| **Cost** | Free (compute only) | $$$  (DB + cache + queue) |

---

## License

MIT
