package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"

	"github.com/segmentio/kafka-go"
)

// MessageHandler defines the function signature for handling messages
type MessageHandler func(Message)

// Consumer handles Kafka message consumption
type Consumer struct {
	reader *kafka.Reader
	group  string
}

// NewConsumer creates a new Kafka consumer
func NewConsumer(broker, topic, group string) *Consumer {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers: []string{broker},
		Topic:   topic,
		GroupID: group,
	})

	return &Consumer{
		reader: reader,
		group:  group,
	}
}

// Consume starts consuming messages and calls the handler for each message
func (c *Consumer) Consume(handler MessageHandler) {
	seen := make(map[string]bool)
	var mu sync.Mutex

	fmt.Printf("Started consumer [%s]\n", c.group)
	for {
		m, err := c.reader.FetchMessage(context.Background())
		if err != nil {
			log.Fatal("fetch:", err)
		}

		var msg Message
		if err := json.Unmarshal(m.Value, &msg); err != nil {
			log.Println("unmarshal:", err)
			continue
		}

		mu.Lock()
		if seen[msg.UniqueID] {
			log.Println("duplicate skipped:", msg.UniqueID)
			mu.Unlock()
		} else {
			seen[msg.UniqueID] = true
			mu.Unlock()
			handler(msg)
		}

		if err := c.reader.CommitMessages(context.Background(), m); err != nil {
			log.Println("commit:", err)
		}
	}
}

// Close closes the consumer
func (c *Consumer) Close() error {
	return c.reader.Close()
}
