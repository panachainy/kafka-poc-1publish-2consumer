package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/segmentio/kafka-go"
)

// Consumer handles Kafka message consumption
type Consumer struct {
	reader     *kafka.Reader
	group      string
	topic      string
	broker     string
	producer   *Producer
	maxRetries int
}

// NewConsumer creates a new Kafka consumer
func NewConsumer(broker, topic, group string, maxRetries int) *Consumer {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers: []string{broker},
		Topic:   topic,
		GroupID: group,
	})

	// Create a producer for dead letter queue
	producer := NewProducer(broker, topic)

	return &Consumer{
		reader:     reader,
		group:      group,
		topic:      topic,
		broker:     broker,
		producer:   producer,
		maxRetries: maxRetries,
	}
}

// Consume starts consuming messages and calls the handler for each message
func (c *Consumer) Consume(handler MessageHandler) {
	seen := make(map[string]bool)
	retryCount := make(map[string]int)
	var mu sync.Mutex

	fmt.Printf("Started consumer [%s] with max retries: %d\n", c.group, c.maxRetries)
	for {
		m, err := c.reader.FetchMessage(context.Background())
		if err != nil {
			log.Fatal("fetch:", err)
		}

		var msg Message
		if err := json.Unmarshal(m.Value, &msg); err != nil {
			log.Println("unmarshal:", err)
			// Commit the message to skip malformed messages
			if err := c.reader.CommitMessages(context.Background(), m); err != nil {
				log.Println("commit after unmarshal error:", err)
			}
			continue
		}

		mu.Lock()
		if seen[msg.UniqueID] {
			log.Println("duplicate skipped:", msg.UniqueID)
			mu.Unlock()
			// Commit the duplicate message
			if err := c.reader.CommitMessages(context.Background(), m); err != nil {
				log.Println("commit duplicate:", err)
			}
			continue
		}

		currentRetries := retryCount[msg.UniqueID]
		mu.Unlock()

		// Try to process the message with retry logic
		if err := c.processWithRetry(msg, handler, currentRetries); err != nil {
			mu.Lock()
			retryCount[msg.UniqueID]++
			currentRetries = retryCount[msg.UniqueID]
			mu.Unlock()

			if currentRetries >= c.maxRetries {
				// Send to dead letter queue after max retries
				if dlqErr := c.producer.SendToDeadLetterQueue(msg, err.Error(), currentRetries, c.group); dlqErr != nil {
					log.Printf("Failed to send to DLQ: %v", dlqErr)
					// TODO: check this it really ok?
				}

				// Mark as processed and commit
				mu.Lock()
				seen[msg.UniqueID] = true
				delete(retryCount, msg.UniqueID)
				mu.Unlock()

				log.Printf("Message %s sent to DLQ after %d retries", msg.UniqueID, currentRetries)
			} else {
				log.Printf("Message %s failed, retry %d/%d: %v", msg.UniqueID, currentRetries, c.maxRetries, err)
				// Don't commit yet, will retry
				continue
			}
		} else {
			// Success - mark as seen and clean up retry count
			mu.Lock()
			seen[msg.UniqueID] = true
			delete(retryCount, msg.UniqueID)
			mu.Unlock()
		}

		// Commit the message
		if err := c.reader.CommitMessages(context.Background(), m); err != nil {
			log.Println("commit:", err)
		}
	}
}

// processWithRetry processes a message with exponential backoff
func (c *Consumer) processWithRetry(msg Message, handler MessageHandler, retryCount int) error {
	// Exponential backoff: 1s, 2s, 4s, etc.
	if retryCount > 0 {
		backoffDuration := time.Duration(1<<uint(retryCount-1)) * time.Second
		log.Printf("Waiting %v before retry %d for message %s", backoffDuration, retryCount, msg.UniqueID)
		time.Sleep(backoffDuration)
	}

	// Handle panics and convert them to errors
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Panic recovered for message %s: %v", msg.UniqueID, r)
		}
	}()

	return handler(msg)
}

// Close closes the consumer
func (c *Consumer) Close() error {
	if err := c.reader.Close(); err != nil {
		return err
	}
	return c.producer.Close()
}
