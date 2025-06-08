package kafka

import (
	"fmt"
	"time"
)

// HandleInventory processes inventory-related messages
func HandleInventory(msg Message) {
	fmt.Printf("[Inventory] updating stock for %s sold at %v\n", msg.ItemID, time.Unix(msg.SoldAt, 0))
}

// HandleAnalytics processes analytics-related messages
func HandleAnalytics(msg Message) {
	fmt.Printf("[Analytics] recording sale of %s sold at %v\n", msg.ItemID, time.Unix(msg.SoldAt, 0))
}
