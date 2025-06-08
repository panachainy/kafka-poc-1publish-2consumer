package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"path/filepath"
	"time"

	"kafka-poc-1publish-2consumer/pkg/sqlite"

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
	tracker    *sqlite.MessageTracker
}

// NewConsumer creates a new Kafka consumer
func NewConsumer(broker, topic, group string, maxRetries int) *Consumer {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:        []string{broker},
		Topic:          topic,
		GroupID:        group,
		CommitInterval: 0,
	})

	// Create a producer for dead letter queue
	producer := NewProducer(broker, topic)

	// Initialize SQLite message tracker
	dbPath := filepath.Join("data", fmt.Sprintf("%s_messages.db", group))
	tracker, err := sqlite.NewMessageTracker(dbPath)
	if err != nil {
		log.Fatalf("Failed to initialize message tracker: %v", err)
	}

	return &Consumer{
		reader:     reader,
		group:      group,
		topic:      topic,
		broker:     broker,
		producer:   producer,
		maxRetries: maxRetries,
		tracker:    tracker,
	}
}

// Consume starts consuming messages and calls the handler for each message
func (c *Consumer) Consume(handler MessageHandler) {
	fmt.Printf("Started consumer [%s] with max retries: %d\n", c.group, c.maxRetries)
	for {
		log.Println("Step 1: Fetching message from Kafka")
		m, err := c.reader.FetchMessage(context.Background())
		if err != nil {
			log.Fatal("fetch:", err)
		}

		log.Printf("Step 2: Received message, attempting to unmarshal (size: %d bytes)", len(m.Value))
		var msg Message
		if err := json.Unmarshal(m.Value, &msg); err != nil {
			log.Println("Step 2 failed - unmarshal:", err)
			// Commit the message to skip malformed messages
			if err := c.reader.CommitMessages(context.Background(), m); err != nil {
				log.Println("commit after unmarshal error:", err)
			}
			continue
		}

		log.Printf("Step 3: Message unmarshaled successfully, ID: %s", msg.UniqueID)

		// Check if message has been seen before using SQLite
		isSeen, err := c.tracker.IsSeen(msg.UniqueID, c.group)
		if err != nil {
			log.Printf("Error checking if message is seen: %v", err)
			continue
		}

		if isSeen {
			log.Printf("Step 4: Duplicate message detected and skipped: %s", msg.UniqueID)
			// Commit the duplicate message
			if err := c.reader.CommitMessages(context.Background(), m); err != nil {
				log.Println("commit duplicate:", err)
			}
			continue
		}

		// Get current retry count from SQLite
		currentRetries, err := c.tracker.GetRetryCount(msg.UniqueID, c.group)
		if err != nil {
			log.Printf("Error getting retry count: %v", err)
			continue
		}

		log.Printf("Step 4: Processing message %s (retry count: %d)", msg.UniqueID, currentRetries)

		// Try to process the message with retry logic
		if err := c.processWithRetry(msg, handler, currentRetries); err != nil {
			log.Printf("Step 5: Message processing failed for %s: %v", msg.UniqueID, err)

			// Increment retry count in SQLite
			newRetryCount, retryErr := c.tracker.IncrementRetryCount(msg.UniqueID, c.group)
			if retryErr != nil {
				log.Printf("Error incrementing retry count: %v", retryErr)
				continue
			}

			if newRetryCount >= c.maxRetries {
				log.Printf("Step 6: Max retries reached for %s, sending to DLQ", msg.UniqueID)
				// Send to dead letter queue after max retries
				if dlqErr := c.producer.SendToDeadLetterQueue(msg, err.Error(), newRetryCount, c.group); dlqErr != nil {
					log.Printf("Failed to send to DLQ: %v", dlqErr)
					// TODO: check this it really ok?
				}

				// Mark as processed and clean up retry count
				if markErr := c.tracker.MarkAsSeen(msg.UniqueID, c.group); markErr != nil {
					log.Printf("Error marking message as seen: %v", markErr)
				}
				if delErr := c.tracker.DeleteRetryCount(msg.UniqueID, c.group); delErr != nil {
					log.Printf("Error deleting retry count: %v", delErr)
				}

				log.Printf("Step 7: Message %s sent to DLQ after %d retries", msg.UniqueID, newRetryCount)
			} else {
				log.Printf("Step 6: Message %s will retry (%d/%d): %v", msg.UniqueID, newRetryCount, c.maxRetries, err)
				// Don't commit yet, will retry
				continue
			}
		} else {
			log.Printf("Step 5: Message %s processed successfully", msg.UniqueID)
			// Success - mark as seen and clean up retry count
			if markErr := c.tracker.MarkAsSeen(msg.UniqueID, c.group); markErr != nil {
				log.Printf("Error marking message as seen: %v", markErr)
			}
			if delErr := c.tracker.DeleteRetryCount(msg.UniqueID, c.group); delErr != nil {
				log.Printf("Error deleting retry count: %v", delErr)
			}
		}

		log.Printf("Step 8: Committing message %s", msg.UniqueID)
		// Commit the message
		if err := c.reader.CommitMessages(context.Background(), m); err != nil {
			log.Println("commit:", err)
		}
		log.Printf("Step 9: Message %s processing completed", msg.UniqueID)
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
	if err := c.tracker.Close(); err != nil {
		return err
	}
	return c.producer.Close()
}
