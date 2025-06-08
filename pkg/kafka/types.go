package kafka

import "time"

// Message defines our event structure
type Message struct {
	ItemID   string `json:"item_id"`
	SoldAt   int64  `json:"sold_at"`
	UniqueID string `json:"unique_id"`
}

// DeadLetterMessage defines the structure for messages sent to dead letter queue
type DeadLetterMessage struct {
	OriginalMessage Message   `json:"original_message"`
	ErrorMessage    string    `json:"error_message"`
	RetryCount      int       `json:"retry_count"`
	FailedAt        time.Time `json:"failed_at"`
	ConsumerGroup   string    `json:"consumer_group"`
}

// MessageHandler defines the function signature for handling messages with error return
type MessageHandler func(Message) error
