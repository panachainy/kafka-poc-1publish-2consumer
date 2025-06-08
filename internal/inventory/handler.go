package inventory

import (
	"fmt"
	"time"

	"kafka-poc-1publish-2consumer/pkg/kafka"
)

// Handler processes inventory-related messages
func Handler(msg kafka.Message) {
	fmt.Printf("[Inventory] updating stock for %s sold at %v\n", msg.ItemID, time.Unix(msg.SoldAt, 0))
}
