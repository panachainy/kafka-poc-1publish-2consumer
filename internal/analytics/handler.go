package analytics

import (
	"fmt"
	"time"

	"kafka-poc-1publish-2consumer/pkg/kafka"
)

// Handler processes analytics-related messages
func Handler(msg kafka.Message) error {
	fmt.Printf("[Analytics] recording sale of %s sold at %v\n", msg.ItemID, time.Unix(msg.SoldAt, 0))
	return nil
}
