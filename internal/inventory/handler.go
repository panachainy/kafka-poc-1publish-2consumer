package inventory

import (
	"fmt"
	"time"

	"kafka-poc-1publish-2consumer/pkg/kafka"
)

// Handler processes inventory-related messages
func Handler(msg kafka.Message) {
	// Simulate 50% failure rate for testing dead letter handling
	if time.Now().UnixNano()%2 == 0 {
		panic(fmt.Sprintf("[Inventory] simulated failure for item %s", msg.ItemID))
	}

	fmt.Printf("[Inventory] updating stock for %s sold at %v\n", msg.ItemID, time.Unix(msg.SoldAt, 0))
}
