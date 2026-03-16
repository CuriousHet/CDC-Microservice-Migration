package main

import (
	"context"
	"fmt"
	"log"

	"github.com/segmentio/kafka-go"
)

func main() {
	// Consuming from the users topic created by Debezium
	// Format: <server>.<schema>.<table>
	topic := "monolith.public.users"
	partition := 0

	conn, err := kafka.DialLeader(context.Background(), "tcp", "localhost:9092", topic, partition)
	if err != nil {
		log.Fatal("failed to dial leader:", err)
	}

	fmt.Printf("Monitoring topic: %s...\n", topic)

	for {
		m, err := conn.ReadMessage(10e6) // 10MB max
		if err != nil {
			break
		}
		fmt.Printf("Message received: %s\n", string(m.Value))
	}

	if err := conn.Close(); err != nil {
		log.Fatal("failed to close connection:", err)
	}
}
