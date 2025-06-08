# Makefile

# Variables
COMPOSE_FILE = docker-compose.yml
GO = go
APP = kafka-poc-1publish-2consumer
MAIN = main.go
TOPIC = item-sold

.PHONY: all build prod inv analytics up down logs clean

all: build

build:
	$(GO) mod tidy
	$(GO) build -o bin/producer -tags producer .
	$(GO) build -o bin/inventory -tags inventory .
	$(GO) build -o bin/analytics -tags analytics .

# Start Kafka + Zookeeper
up:
	docker compose -f $(COMPOSE_FILE) up -d

# Stop containers
down:
	docker compose -f $(COMPOSE_FILE) down

# Inspect logs
logs:
	docker compose -f $(COMPOSE_FILE) logs -f

# Run producer
prod: build
	./bin/producer

# Run inventory consumer (in separate terminal)
inv: build
	./bin/inventory

# Run analytics consumer (in separate terminal)
analytics: build
	./bin/analytics

# Full cycle demo
demo: up build
	@echo "Starting producer in background..."
	./bin/producer &
	@echo "Starting consumers..."
	./bin/inventory & ./bin/analytics &
	@echo "Use Ctrl-C to stop..."

# Clean generated files
clean:
	-rm -rf bin
