# kafka-poc-1publish-2consumer

## Configuration

The application uses environment variables for configuration. Copy `.env.example` to `.env` and modify as needed:

```bash
cp .env.example .env
```

Available environment variables:

- `KAFKA_TOPIC` - Kafka topic name (default: `item-sold`)
- `KAFKA_BROKER` - Kafka broker address (default: `localhost:9092`)

## Commands

Makefile gives you a smooth developer workflow:

1. `make up` → spin up Kafka/Zookeeper
2. `make prod` → publish an event
3. `make inv` and `make analytics` → start the two consumers
4. Use `make logs` to trace what's happening
5. `make down` and `make clean` to clean up
