package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"github.com/segmentio/kafka-go"
)

// Debezium compatible structure
type CDCEvent struct {
	Op    string      `json:"op"`
	After interface{} `json:"after"`
}

type User struct {
	ID        int    `json:"id"`
	Email     string `json:"email"`
	Name      string `json:"name"`
	CreatedAt int64  `json:"created_at"` // Microseconds
}

type Order struct {
	ID        int    `json:"id"`
	UserID    int    `json:"user_id"`
	Amount    string `json:"amount"` // Sent as string for compatibility
	CreatedAt int64  `json:"created_at"`
}

func main() {
	log.Println("--- Starting Backfill Process ---")
	
	// 0. Load .env file
	if err := godotenv.Load("../.env"); err != nil {
		log.Println("Warning: .env file not found, using system environment variables")
	}

	// 1. Connect to Monolith DB
	monolithDBUrl := os.Getenv("MONOLITH_DB_URL")
	if monolithDBUrl == "" {
		log.Fatal("MONOLITH_DB_URL is not set")
	}
	
	log.Printf("Connecting to Monolith DB: %s", monolithDBUrl)
	db, err := sql.Open("postgres", monolithDBUrl)
	if err != nil {
		log.Fatal("Failed to open DB:", err)
	}
	defer db.Close()

	// Check connection
	if err := db.Ping(); err != nil {
		log.Fatal("Failed to reach Monolith DB:", err)
	}
	log.Println("Successfully connected to Monolith DB.")

	// 2. Setup Kafka Broker address
	kafkaBroker := os.Getenv("KAFKA_BROKER")
	if kafkaBroker == "" {
		log.Fatal("KAFKA_BROKER is not set")
	}
	log.Printf("Using Kafka Broker: %s", kafkaBroker)

	// 3. Run Backfills
	log.Println("Starting User backfill...")
	backfillUsers(db, kafkaBroker)
	
	log.Println("Starting Order backfill...")
	backfillOrders(db, kafkaBroker)
	
	log.Println("--- Backfill Completed ---")
}

func backfillUsers(db *sql.DB, broker string) {
	writer := &kafka.Writer{
		Addr:     kafka.TCP(broker),
		Topic:    "monolith.public.users",
		Balancer: &kafka.LeastBytes{},
		Async:    false, // Wait for acks so we see errors immediately
	}
	defer writer.Close()

	log.Println("Querying users from Monolith...")
	rows, err := db.Query("SELECT id, email, name, created_at FROM users")
	if err != nil {
		log.Printf("Error querying users: %v", err)
		return
	}
	defer rows.Close()

	count := 0
	var messages []kafka.Message

	for rows.Next() {
		var u User
		var createdAt time.Time
		if err := rows.Scan(&u.ID, &u.Email, &u.Name, &createdAt); err != nil {
			log.Printf("Error scanning user: %v", err)
			continue
		}
		u.CreatedAt = createdAt.UnixNano() / 1000

		event := CDCEvent{Op: "r", After: &u}
		payload, _ := json.Marshal(event)

		messages = append(messages, kafka.Message{
			Key:   []byte(fmt.Sprintf("%d", u.ID)),
			Value: payload,
		})

		count++
		
		// Flush every 100 messages
		if len(messages) >= 100 {
			err := writer.WriteMessages(context.Background(), messages...)
			if err != nil {
				log.Printf("Error writing batch to Kafka: %v", err)
				return
			}
			messages = []kafka.Message{}
			log.Printf("Sent %d users...", count)
		}
	}

	// Flush remaining
	if len(messages) > 0 {
		writer.WriteMessages(context.Background(), messages...)
	}

	log.Printf("Successfully backfilled %d users.", count)
}

func backfillOrders(db *sql.DB, broker string) {
	writer := &kafka.Writer{
		Addr:     kafka.TCP(broker),
		Topic:    "monolith.public.orders",
		Balancer: &kafka.LeastBytes{},
		Async:    false,
	}
	defer writer.Close()

	log.Println("Querying orders from Monolith...")
	rows, err := db.Query("SELECT id, user_id, amount, created_at FROM orders")
	if err != nil {
		log.Printf("Error querying orders: %v", err)
		return
	}
	defer rows.Close()

	count := 0
	var messages []kafka.Message

	for rows.Next() {
		var o Order
		var amount float64
		var createdAt time.Time
		if err := rows.Scan(&o.ID, &o.UserID, &amount, &createdAt); err != nil {
			log.Printf("Error scanning order: %v", err)
			continue
		}
		o.Amount = fmt.Sprintf("%.2f", amount)
		o.CreatedAt = createdAt.UnixNano() / 1000

		event := CDCEvent{Op: "r", After: &o}
		payload, _ := json.Marshal(event)

		messages = append(messages, kafka.Message{
			Key:   []byte(fmt.Sprintf("%d", o.ID)),
			Value: payload,
		})

		count++

		// Flush every 100 messages
		if len(messages) >= 100 {
			err := writer.WriteMessages(context.Background(), messages...)
			if err != nil {
				log.Printf("Error writing batch to Kafka: %v", err)
				return
			}
			messages = []kafka.Message{}
			log.Printf("Sent %d orders...", count)
		}
	}

	// Flush remaining
	if len(messages) > 0 {
		writer.WriteMessages(context.Background(), messages...)
	}

	log.Printf("Successfully backfilled %d orders.", count)
}
