package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/segmentio/kafka-go"
)

// getEnv returns the value of an environment variable or a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// Message defines our event
type Message struct {
	ItemID   string `json:"item_id"`
	SoldAt   int64  `json:"sold_at"`
	UniqueID string `json:"unique_id"`
}

func main() {
	mode := flag.String("mode", "producer", "mode: producer | inventory | analytics")
	flag.Parse()

	topic := getEnv("KAFKA_TOPIC", "item-sold")
	broker := getEnv("KAFKA_BROKER", "localhost:9092")

	switch *mode {
	case "producer":
		log.Println("Starting producer...")
		produce(broker, topic)
	case "inventory":
		log.Println("Starting inventory consumer...")
		consume(broker, topic, "inventory-group", handleInventory)
	case "analytics":
		log.Println("Starting analytics consumer...")
		consume(broker, topic, "analytics-group", handleAnalytics)
	default:
		log.Fatalf("invalid mode %s", *mode)
	}
}

func produce(broker, topic string) {
	// Create topic if it doesn't exist
	conn, err := kafka.Dial("tcp", broker)
	if err != nil {
		log.Fatal("dial:", err)
	}
	defer conn.Close()

	controller, err := conn.Controller()
	if err != nil {
		log.Fatal("controller:", err)
	}
	var controllerConn *kafka.Conn
	controllerConn, err = kafka.Dial("tcp", fmt.Sprintf("%s:%d", controller.Host, controller.Port))
	if err != nil {
		log.Fatal("dial controller:", err)
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
		log.Printf("create topic warning: %v (this is ok if topic already exists)", err)
	} else {
		log.Printf("created topic: %s", topic)
	}

	w := kafka.NewWriter(kafka.WriterConfig{
		Brokers: []string{broker},
		Topic:   topic,
		Async:   false,
	})
	defer w.Close()

	msg := Message{
		ItemID:   "ABC123",
		SoldAt:   time.Now().Unix(),
		UniqueID: fmt.Sprintf("%d", time.Now().UnixNano()),
	}
	b, _ := json.Marshal(msg)
	err = w.WriteMessages(context.Background(),
		kafka.Message{
			Key:   []byte(msg.ItemID),
			Value: b,
		})
	if err != nil {
		log.Fatal("write:", err)
	}
	fmt.Println("Produced:", msg)
}

func consume(broker, topic, group string, handler func(Message)) {
	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers: []string{broker},
		Topic:   topic,
		GroupID: group,
	})
	defer r.Close()

	seen := make(map[string]bool)
	var mu sync.Mutex

	fmt.Printf("Started consumer [%s]\n", group)
	for {
		m, err := r.FetchMessage(context.Background())
		if err != nil {
			log.Fatal("fetch:", err)
		}
		var msg Message
		if err := json.Unmarshal(m.Value, &msg); err != nil {
			log.Println("unmarshal:", err)
			continue
		}
		mu.Lock()
		if seen[msg.UniqueID] {
			log.Println("duplicate skipped:", msg.UniqueID)
			mu.Unlock()
		} else {
			seen[msg.UniqueID] = true
			mu.Unlock()
			handler(msg)
		}
		if err := r.CommitMessages(context.Background(), m); err != nil {
			log.Println("commit:", err)
		}
	}
}

func handleInventory(msg Message) {
	fmt.Printf("[Inventory] updating stock for %s sold at %v\n", msg.ItemID, time.Unix(msg.SoldAt, 0))
}

func handleAnalytics(msg Message) {
	fmt.Printf("[Analytics] recording sale of %s sold at %v\n", msg.ItemID, time.Unix(msg.SoldAt, 0))
}
