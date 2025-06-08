package kafka

// Message defines our event structure
type Message struct {
	ItemID   string `json:"item_id"`
	SoldAt   int64  `json:"sold_at"`
	UniqueID string `json:"unique_id"`
}
