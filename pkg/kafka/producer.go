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
	writer    *kafka.Writer
	dlqWriter *kafka.Writer
	broker    string
}

// NewProducer creates a new Kafka producer
func NewProducer(broker, topic string) *Producer {
	// Create topic if it doesn't exist
	err := createTopicIfNotExists(broker, topic)
	if err != nil {
		log.Printf("create topic warning: %v (this is ok if topic already exists)", err)
	}

	// Create dead letter queue topic
	dlqTopic := topic + "-dlq"
	err = createTopicIfNotExists(broker, dlqTopic)
	if err != nil {
		log.Printf("create DLQ topic warning: %v (this is ok if topic already exists)", err)
	}

	writer := kafka.NewWriter(kafka.WriterConfig{
		Brokers: []string{broker},
		Topic:   topic,
		Async:   false,
	})

	dlqWriter := kafka.NewWriter(kafka.WriterConfig{
		Brokers: []string{broker},
		Topic:   dlqTopic,
		Async:   false,
	})

	return &Producer{
		writer:    writer,
		dlqWriter: dlqWriter,
		broker:    broker,
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
	if err := p.writer.Close(); err != nil {
		return err
	}
	return p.dlqWriter.Close()
}

// SendToDeadLetterQueue sends a failed message to the dead letter queue
func (p *Producer) SendToDeadLetterQueue(originalMsg Message, errorMsg string, retryCount int, consumerGroup string) error {
	dlqMsg := DeadLetterMessage{
		OriginalMessage: originalMsg,
		ErrorMessage:    errorMsg,
		RetryCount:      retryCount,
		FailedAt:        time.Now(),
		ConsumerGroup:   consumerGroup,
	}

	b, err := json.Marshal(dlqMsg)
	if err != nil {
		return fmt.Errorf("marshal DLQ message: %w", err)
	}

	err = p.dlqWriter.WriteMessages(context.Background(),
		kafka.Message{
			Key:   []byte(originalMsg.ItemID),
			Value: b,
		})
	if err != nil {
		return fmt.Errorf("write DLQ message: %w", err)
	}

	log.Printf("Sent to DLQ: ItemID=%s, Error=%s, RetryCount=%d", originalMsg.ItemID, errorMsg, retryCount)
	return nil
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
