# Copilot Instructions for kafka-poc-1publish-2consumer

## Architecture Overview
This is a Go-based Kafka proof-of-concept with 1 producer and 2 consumers (inventory and analytics groups). Messages flow from producer → Kafka topic → consumers, with per-consumer SQLite tracking for deduplication and retries.

**Key Components:**
- `pkg/kafka/`: Core Kafka logic (producer, consumer, types)
- `internal/analytics/`, `internal/inventory/`: Consumer handlers
- `pkg/sqlite/`: Message tracking per consumer group
- `cmd/main.go`: CLI entry point with modes (`-mode=producer|inventory|analytics`)

**Data Flow:**
- Producer publishes `Message{ItemID, SoldAt, UniqueID}` to "item-sold" topic
- Consumers check SQLite for seen messages (per group) before processing
- Failed messages retry with exponential backoff (1s, 2s, 4s...) up to 3 attempts
- Max retries → send to DLQ topic ("item-sold-dlq")

## Developer Workflows
Use Makefile for all operations:
- `make up`: Start Kafka (KRaft mode, no Zookeeper) via podman-compose
- `make prod`: Build and run producer
- `make inv` / `make analytics`: Run consumers (separate terminals)
- `make logs`: View container logs
- `make down`: Stop containers

**Configuration:** Environment variables (copy `.env.example` to `.env`):
- `KAFKA_TOPIC`: Default "item-sold"
- `KAFKA_BROKER`: Default "localhost:9092"
- `KAFKA_MAX_RETRIES`: Default 3

## Project Conventions
- **Message Uniqueness:** Use `UniqueID` (nanosecond timestamp) for deduplication, tracked per consumer group in SQLite
- **Handler Pattern:** `func(Message) error` - return error for retry, nil for success
- **Retry Logic:** Exponential backoff `time.Duration(1<<uint(retryCount-1)) * time.Second`
- **SQLite Tracking:** Separate DB per group (`data/{group}_messages.db`) with `seen_messages` and `retry_counts` tables
- **Topic Creation:** Producer auto-creates topics and DLQ topics if missing
- **Consumer Groups:** "inventory-group", "analytics-group" - each processes all messages but tracks independently
- **Dead Letter Queue:** Failed messages after max retries sent to `{topic}-dlq`

**Examples:**
- Add new consumer: Create `internal/{name}/handler.go` with `Handler func(Message) error`, add mode in `main.go`
- Custom retry: Modify `processWithRetry` in `consumer.go`
- New message fields: Update `Message` struct in `types.go`, ensure JSON tags

## Integration Points
- **Kafka:** segmentio/kafka-go library, KRaft mode in compose.yml
- **Database:** SQLite for message tracking (github.com/mattn/go-sqlite3)
- **Container:** podman-compose for local Kafka setup</content>
<parameter name="filePath">/Users/panachainy/repo/personal/kafka-poc-1publish-2consumer/.github/copilot-instructions.md