package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/segmentio/kafka-go"
)

// Producer handles Kafka message production
type Producer struct {
	writer *kafka.Writer
}

// NewProducer creates a new Kafka producer
func NewProducer(broker, topic string) *Producer {
	// Create topic if it doesn't exist
	err := createTopicIfNotExists(broker, topic)
	if err != nil {
		log.Printf("create topic warning: %v (this is ok if topic already exists)", err)
	}

	writer := kafka.NewWriter(kafka.WriterConfig{
		Brokers: []string{broker},
		Topic:   topic,
		Async:   false,
	})

	return &Producer{
		writer: writer,
	}
}

// Produce sends a message to Kafka
func (p *Producer) Produce(itemID string) error {
	msg := Message{
		ItemID:   itemID,
		SoldAt:   time.Now().Unix(),
		UniqueID: fmt.Sprintf("%d", time.Now().UnixNano()),
	}

	b, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal message: %w", err)
	}

	err = p.writer.WriteMessages(context.Background(),
		kafka.Message{
			Key:   []byte(msg.ItemID),
			Value: b,
		})
	if err != nil {
		return fmt.Errorf("write message: %w", err)
	}

	fmt.Println("Produced:", msg)
	return nil
}

// Close closes the producer
func (p *Producer) Close() error {
	return p.writer.Close()
}

// createTopicIfNotExists creates a Kafka topic if it doesn't exist
func createTopicIfNotExists(broker, topic string) error {
	conn, err := kafka.Dial("tcp", broker)
	if err != nil {
		return fmt.Errorf("dial: %w", err)
	}
	defer conn.Close()

	controller, err := conn.Controller()
	if err != nil {
		return fmt.Errorf("controller: %w", err)
	}

	controllerConn, err := kafka.Dial("tcp", fmt.Sprintf("%s:%d", controller.Host, controller.Port))
	if err != nil {
		return fmt.Errorf("dial controller: %w", err)
	}
	defer controllerConn.Close()

	topicConfigs := []kafka.TopicConfig{
		{
			Topic:             topic,
			NumPartitions:     1,
			ReplicationFactor: 1,
		},
	}

	err = controllerConn.CreateTopics(topicConfigs...)
	if err != nil {
		return err
	}

	log.Printf("created topic: %s", topic)
	return nil
}
