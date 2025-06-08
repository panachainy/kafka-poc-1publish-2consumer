# kafka-poc-1publish-2consumer

## Overview

This project demonstrates a simple Kafka setup with one producer and two consumers. The producer publishes messages to a Kafka topic, while the consumers read from that topic and process the messages.

### POC List

- [x] 1 producer and 2 consumers getting messages from the same topic
- [ ] In fail case, the consumer should retry
  - [x] Support retry with exponential backoff
  - [x] Support retry with a maximum number of attempts
  - [ ] Support retry with multiple the same consumer group instances.
- [ ] In fail case over 3times, the consumer should send a message to a dead letter queue

## Development

### Configuration

The application uses environment variables for configuration. Copy `.env.example` to `.env` and modify as needed:

```bash
cp .env.example .env
```

Available environment variables:

- `KAFKA_TOPIC` - Kafka topic name (default: `item-sold`)
- `KAFKA_BROKER` - Kafka broker address (default: `localhost:9092`)

### Commands

Makefile gives you a smooth developer workflow:

1. `make up` → spin up Kafka/Zookeeper
2. `make prod` → publish an event
3. `make inv` and `make analytics` → start the two consumers
4. Use `make logs` to trace what's happening
5. `make down` and `make clean` to clean up
