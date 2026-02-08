# Makefile

# Variables
COMPOSE_FILE = compose.yml
# COMPOSE_CMD = docker-compose
COMPOSE_CMD = podman-compose
GO = go
APP = kafka-poc-1publish-2consumer
MAIN = main.go
TOPIC = item-sold

.PHONY: all build prod inv analytics up down logs clean

all: build

build:
	$(GO) mod tidy
	$(GO) build -o bin/$(APP) ./cmd

# Start Kafka + Zookeeper
up:
	$(COMPOSE_CMD) -f $(COMPOSE_FILE) up -d

# Stop containers
down:
	$(COMPOSE_CMD) -f $(COMPOSE_FILE) down

# Inspect logs
logs:
	$(COMPOSE_CMD) -f $(COMPOSE_FILE) logs -f

# Run producer
prod: build
	./bin/$(APP) -mode=producer

# Run inventory consumer (in separate terminal)
inv: build
	./bin/$(APP) -mode=inventory

# Run analytics consumer (in separate terminal)
analytics: build
	./bin/$(APP) -mode=analytics

# Full cycle demo
demo: up build
	@echo "Starting producer in background..."
	./bin/$(APP) -mode=producer &
	@echo "Starting consumers..."
	./bin/$(APP) -mode=inventory & ./bin/$(APP) -mode=analytics &
	@echo "Use Ctrl-C to stop..."

# Clean generated files
clean:
	-rm -rf bin
