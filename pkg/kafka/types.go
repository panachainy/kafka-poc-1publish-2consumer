package kafka

// Message defines our event structure
type Message struct {
	ItemID   string `json:"item_id"`
	SoldAt   int64  `json:"sold_at"`
	UniqueID string `json:"unique_id"`
}

// MessageHandler defines the function signature for handling messages with error return
type MessageHandler func(Message) error
