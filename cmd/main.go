package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"kafka-poc-1publish-2consumer/internal/analytics"
	"kafka-poc-1publish-2consumer/internal/inventory"
	"kafka-poc-1publish-2consumer/pkg/kafka"
)

// getEnv returns the value of an environment variable or a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func main() {
	mode := flag.String("mode", "producer", "mode: producer | inventory | analytics")
	flag.Parse()

	fmt.Printf("Running in mode: %s\n", *mode)

	topic := getEnv("KAFKA_TOPIC", "item-sold")
	broker := getEnv("KAFKA_BROKER", "localhost:9092")

	switch *mode {
	case "producer":
		log.Println("Starting producer...")
		producer := kafka.NewProducer(broker, topic)
		defer producer.Close()

		err := producer.Produce("ABC123")
		if err != nil {
			log.Fatal("produce error:", err)
		}
	case "inventory":
		log.Println("Starting inventory consumer...")
		consumer := kafka.NewConsumer(broker, topic, "inventory-group")
		defer consumer.Close()
		consumer.Consume(inventory.Handler)
	case "analytics":
		log.Println("Starting analytics consumer...")
		consumer := kafka.NewConsumer(broker, topic, "analytics-group")
		defer consumer.Close()
		consumer.Consume(analytics.Handler)
	default:
		log.Fatalf("invalid mode %s", *mode)
	}
}
