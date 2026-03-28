package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"time"
)

type User struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

type Order struct {
	UserID int     `json:"user_id"`
	Amount float64 `json:"amount"`
}

func main() {
	fmt.Println("Traffic Generator starting...")
	
	for {
		// Simulate User Creation
		userName := fmt.Sprintf("User_%d", rand.Intn(1000000))
		userEmail := fmt.Sprintf("%s@example.com", userName)
		
		user := User{Name: userName, Email: userEmail}
		userData, _ := json.Marshal(user)
		
		resp, err := http.Post("http://localhost:8000/users", "application/json", bytes.NewBuffer(userData))
		if err == nil && resp.StatusCode == http.StatusCreated {
			var createdUser struct{ ID int }
			json.NewDecoder(resp.Body).Decode(&createdUser)
			resp.Body.Close()
			
			fmt.Printf("Created User ID: %d\n", createdUser.ID)
			
			// Simulate Order Creation for this user
			order := Order{
				UserID: createdUser.ID,
				Amount: float64(rand.Intn(500)) + rand.Float64(),
			}
			orderData, _ := json.Marshal(order)
			http.Post("http://localhost:8000/orders", "application/json", bytes.NewBuffer(orderData))
		}

		time.Sleep(1 * time.Second) // Adjust frequency as needed
	}
}
