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
	// 0. Load .env file (try common paths)
	godotenv.Load("../.env")

	// 1. Connect to Monolith DB
	monolithDBUrl := os.Getenv("MONOLITH_DB_URL")
	if monolithDBUrl == "" {
		log.Fatal("MONOLITH_DB_URL is not set")
	}
	db, err := sql.Open("postgres", monolithDBUrl)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// 2. Setup Kafka Writer
	kafkaBroker := os.Getenv("KAFKA_BROKER")
	if kafkaBroker == "" {
		log.Fatal("KAFKA_BROKER is not set")
	}

	backfillUsers(db, kafkaBroker)
	backfillOrders(db, kafkaBroker)
}

func backfillUsers(db *sql.DB, broker string) {
	writer := &kafka.Writer{
		Addr:     kafka.TCP(broker),
		Topic:    "monolith.public.users",
		Balancer: &kafka.LeastBytes{},
	}
	defer writer.Close()

	rows, err := db.Query("SELECT id, email, name, created_at FROM users")
	if err != nil {
		log.Printf("Error querying users: %v", err)
		return
	}
	defer rows.Close()

	count := 0
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

		writer.WriteMessages(context.Background(),
			kafka.Message{Key: []byte(fmt.Sprintf("%d", u.ID)), Value: payload},
		)
		count++
	}
	log.Printf("Successfully backfilled %d users.", count)
}

func backfillOrders(db *sql.DB, broker string) {
	writer := &kafka.Writer{
		Addr:     kafka.TCP(broker),
		Topic:    "monolith.public.orders",
		Balancer: &kafka.LeastBytes{},
	}
	defer writer.Close()

	rows, err := db.Query("SELECT id, user_id, amount, created_at FROM orders")
	if err != nil {
		log.Printf("Error querying orders: %v", err)
		return
	}
	defer rows.Close()

	count := 0
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

		writer.WriteMessages(context.Background(),
			kafka.Message{Key: []byte(fmt.Sprintf("%d", o.ID)), Value: payload},
		)
		count++
	}
	log.Printf("Successfully backfilled %d orders.", count)
}
