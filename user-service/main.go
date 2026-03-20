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
	_ "github.com/lib/pq"
	"github.com/segmentio/kafka-go"
)

// User represents the data stored in the User Service
type User struct {
	ID        int       `json:"id" db:"id"`
	Email     string    `json:"email" db:"email"`
	Name      string    `json:"name" db:"name"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// CDCEvent represents the structure of a Debezium Kafka message
type CDCEvent struct {
	Op     string `json:"op"` // 'c' for create, 'u' for update, 'd' for delete
	Before *struct {
		ID        int    `json:"id"`
		Email     string `json:"email"`
		Name      string `json:"name"`
		CreatedAt int64  `json:"created_at"`
	} `json:"before"`
	After  *struct {
		ID        int    `json:"id"`
		Email     string `json:"email"`
		Name      string `json:"name"`
		CreatedAt int64  `json:"created_at"`
	} `json:"after"`
}

var db *sql.DB

func main() {
	// 1. Connect to User DB
	var err error
	dbConn := os.Getenv("DATABASE_URL")
	if dbConn == "" {
		dbConn = "postgres://postgres:password@localhost:5433/user_service?sslmode=disable"
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
		log.Fatalf("Could not connect to user-db: %v", err)
	}
	log.Println("Connected to User DB")

	// 2. Start Migration Worker in a goroutine
	go startMigrationWorker()

	// 3. Start API Server
	r := gin.Default()

	r.GET("/users/:id", func(c *gin.Context) {
		id := c.Param("id")
		var user User
		err := db.QueryRow("SELECT id, email, name, created_at FROM users WHERE id = $1", id).
			Scan(&user.ID, &user.Email, &user.Name, &user.CreatedAt)
		
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
			return
		} else if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, user)
	})

	log.Println("User Service API starting on :8081")
	r.Run(":8081")
}

func startMigrationWorker() {
	topic := "monolith.public.users"
	broker := os.Getenv("KAFKA_BROKER")
	if broker == "" {
		broker = "localhost:9092"
	}

	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  []string{broker},
		Topic:    topic,
		GroupID:  "user-service-migration",
		MinBytes: 10e3, // 10KB
		MaxBytes: 10e6, // 10MB
	})

	log.Printf("Migration Worker started listening on topic: %s", topic)

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

// Handle CREATE or UPDATE
		if event.Op == "c" || event.Op == "u" || event.Op == "r" {
			user := event.After
			if user == nil {
				continue
			}

			// Convert Debezium microsecond timestamp to time.Time
			createdAt := time.Unix(0, user.CreatedAt*1000)

			// Upsert (IDEMPOTENCY logic)
			query := `
				INSERT INTO users (id, email, name, created_at)
				VALUES ($1, $2, $3, $4)
				ON CONFLICT (id) DO UPDATE SET
					email = EXCLUDED.email,
					name = EXCLUDED.name;
			`
			_, err := db.Exec(query, user.ID, user.Email, user.Name, createdAt)
			if err != nil {
				log.Printf("Error syncing user %d: %v", user.ID, err)
			} else {
				log.Printf("Synced user: %s (ID: %d)", user.Name, user.ID)
			}
		}
	}
}
