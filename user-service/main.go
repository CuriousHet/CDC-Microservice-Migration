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
	After *struct {
		ID        int    `json:"id"`
		Email     string `json:"email"`
		Name      string `json:"name"`
		CreatedAt int64  `json:"created_at"`
	} `json:"after"`
}

var db *sql.DB

func main() {
	// 0. Load .env file (try common paths)
	godotenv.Load("../.env")

	// 1. Connect to User DB
	var err error
	dbConn := os.Getenv("USER_DB_URL")
	if dbConn == "" {
		log.Fatal("USER_DB_URL is not set")	
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

	// 2. Sync ID Sequence (Crucial for Primary Cutover)
	syncIDSequence()

	// 3. Start Migration Worker in a goroutine
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

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "healthy", "service": "user-service"})
	})

	// POST /users: Create a new user (Direct Write as Primary)
	r.POST("/users", func(c *gin.Context) {
		var input struct {
			Email string `json:"email" binding:"required"`
			Name  string `json:"name" binding:"required"`
		}

		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		var user User
		err := db.QueryRow(
			"INSERT INTO users (email, name) VALUES ($1, $2) RETURNING id, email, name, created_at",
			input.Email, input.Name,
		).Scan(&user.ID, &user.Email, &user.Name, &user.CreatedAt)

		if err != nil {
			log.Printf("Error creating user: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "could not create user"})
			return
		}

		log.Printf("[WRITE] New User Created: %s (ID: %d)", user.Name, user.ID)
		c.JSON(http.StatusCreated, user)
	})

	// Parity Checker: Compare local data with "expected" data from monolith
	r.POST("/verify/:id", func(c *gin.Context) {
		id := c.Param("id")
		var expectedUser User
		if err := c.ShouldBindJSON(&expectedUser); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
			return
		}

		var localUser User
		err := db.QueryRow("SELECT id, email, name, created_at FROM users WHERE id = $1", id).
			Scan(&localUser.ID, &localUser.Email, &localUser.Name, &localUser.CreatedAt)

		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"match": false, "error": "User not found locally"})
			return
		} else if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"match": false, "error": err.Error()})
			return
		}

		// Simple comparison (ignoring timestamps if they differ slightly)
		match := localUser.Name == expectedUser.Name && localUser.Email == expectedUser.Email
		
		c.JSON(http.StatusOK, gin.H{
			"id":    id,
			"match": match,
			"local": localUser,
		})
	})

	log.Println("User Service API starting on :8081")
	r.Run(":8081")
}

func startMigrationWorker() {
	topic := "monolith.public.users"
	broker := os.Getenv("KAFKA_BROKER")
	if broker == "" {
    log.Fatal("KAFKA_BROKER is not set")
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

func syncIDSequence() {
	var maxID sql.NullInt64
	err := db.QueryRow("SELECT MAX(id) FROM users").Scan(&maxID)
	if err != nil {
		log.Printf("Error getting max user ID: %v", err)
		return
	}

	if maxID.Valid {
		_, err = db.Exec("SELECT setval('users_id_seq', $1)", maxID.Int64)
		if err != nil {
			log.Printf("Error setting user ID sequence: %v", err)
		} else {
			log.Printf("Synced users_id_seq to %d", maxID.Int64)
		}
	}
}
