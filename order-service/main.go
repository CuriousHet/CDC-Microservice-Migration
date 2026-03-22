package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"github.com/segmentio/kafka-go"
)

// Order represents the data stored in the Order Service
type Order struct {
	ID        int       `json:"id" db:"id"`
	UserID    int       `json:"user_id" db:"user_id"`
	Amount    float64   `json:"amount" db:"amount"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// CDCEvent represents the Debezium Kafka message structure
type CDCEvent struct {
	Op     string `json:"op"`
	Before *struct {
		ID        int     `json:"id"`
		UserID    int     `json:"user_id"`
		Amount    float64 `json:"amount,string"`
		CreatedAt int64   `json:"created_at"`
	} `json:"before"`
	After *struct {
		ID        int     `json:"id"`
		UserID    int     `json:"user_id"`
		Amount    float64 `json:"amount,string"`
		CreatedAt int64   `json:"created_at"`
	} `json:"after"`
}

var db *sql.DB

func main() {
	// 0. Load .env file (try common paths)
	godotenv.Load("../.env")

	// 1. Connect to Order DB (port 5434)
	var err error
	dbConn := os.Getenv("ORDER_DB_URL")
	if dbConn == "" {
		log.Fatal("ORDER_DB_URL is not set")
	}

	for i := 0; i < 5; i++ {
		db, err = sql.Open("postgres", dbConn)
		if err == nil {
			err = db.Ping()
		}
		if err == nil {
			break
		}
		log.Printf("Failed to connect to db (attempt %d): %v", i+1, err)
		time.Sleep(2 * time.Second)
	}
	if err != nil {
		log.Fatalf("Could not connect to order-db: %v", err)
	}
	log.Println("Connected to Order DB")
	
	initDB()

	// 2. Start Migration Worker
	go startMigrationWorker()

	// 3. Start API Server (port 8082)
	r := gin.Default()

	r.GET("/orders/:id", func(c *gin.Context) {
		id := c.Param("id")
		var order Order
		err := db.QueryRow("SELECT id, user_id, amount, created_at FROM orders WHERE id = $1", id).
			Scan(&order.ID, &order.UserID, &order.Amount, &order.CreatedAt)
		
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "order not found"})
			return
		} else if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, order)
	})

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "healthy", "service": "order-service"})
	})

	log.Println("Order Service API starting on :8082")
	r.Run(":8082")
}

func startMigrationWorker() {
	topic := "monolith.public.orders" // Monolith orders table
	broker := os.Getenv("KAFKA_BROKER")
	if broker == "" {
		log.Fatal("KAFKA BROKER is not set.")
	}

	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  []string{broker},
		Topic:    topic,
		GroupID:  "order-service-migration",
		MinBytes: 10e3,
		MaxBytes: 10e6,
	})

	log.Printf("Order Migration Worker started listening on: %s", topic)

	for {
		m, err := reader.ReadMessage(context.Background())
		if err != nil {
			log.Printf("Error reading message: %v", err)
			continue
		}

		var event CDCEvent
		if err := json.Unmarshal(m.Value, &event); err != nil {
			log.Printf("Error unmarshaling message: %v", err)
			continue
		}

		// Handle Snapshot (r), Create (c), or Update (u)
		if event.Op == "c" || event.Op == "u" || event.Op == "r" {
			order := event.After
			if order == nil {
				continue
			}

			// Convert Debezium microseconds to Go time.Time
			createdAt := time.Unix(0, order.CreatedAt*1000)

			// Idempotent Upsert
			query := `
				INSERT INTO orders (id, user_id, amount, created_at)
				VALUES ($1, $2, $3, $4)
				ON CONFLICT (id) DO UPDATE SET
					user_id = EXCLUDED.user_id,
					amount = EXCLUDED.amount;
			`
			_, err := db.Exec(query, order.ID, order.UserID, order.Amount, createdAt)
			if err != nil {
				log.Printf("Error syncing order %d: %v", order.ID, err)
			} else {
				log.Printf("Synced order: %d for user: %d", order.ID, order.UserID)
			}
		}
	}
}

func initDB() {
	query := `
		CREATE TABLE IF NOT EXISTS orders (
			id SERIAL PRIMARY KEY,
			user_id INTEGER,
			amount DECIMAL(10,2) NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);
	`
	_, err := db.Exec(query)
	if err != nil {
		log.Fatalf("Could not initialize order-db: %v", err)
	}
	log.Println("Order DB schema initialized")
}
