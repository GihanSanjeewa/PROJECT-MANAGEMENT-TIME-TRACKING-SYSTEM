package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"
)

type User struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	Role      string    `json:"role"`
	Avatar    string    `json:"avatar"`
	CreatedAt time.Time `json:"created_at"`
}

type UserStore struct {
	sync.RWMutex
	users []User
}

var store = &UserStore{
	users: []User{
		{ID: "usr-1", Name: "Alex Morgan", Email: "alex@company.com", Role: "Project Lead", Avatar: "https://api.dicebear.com/7.x/avataaars/svg?seed=Alex", CreatedAt: time.Now()},
		{ID: "usr-2", Name: "Maya Patel", Email: "maya@company.com", Role: "Senior Frontend Engineer", Avatar: "https://api.dicebear.com/7.x/avataaars/svg?seed=Maya", CreatedAt: time.Now()},
		{ID: "usr-3", Name: "Jordan Lee", Email: "jordan@company.com", Role: "Backend Engineer (Go)", Avatar: "https://api.dicebear.com/7.x/avataaars/svg?seed=Jordan", CreatedAt: time.Now()},
		{ID: "usr-4", Name: "Taylor Kim", Email: "taylor@company.com", Role: "Product Designer", Avatar: "https://api.dicebear.com/7.x/avataaars/svg?seed=Taylor", CreatedAt: time.Now()},
	},
}

func main() {
	port := ":8001"
	mux := http.NewServeMux()

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"healthy","service":"user-service"}`))
	})

	mux.HandleFunc("/api/users", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		store.RLock()
		defer store.RUnlock()
		json.NewEncoder(w).Encode(store.users)
	})

	mux.HandleFunc("/api/auth/login", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var creds struct {
			Email    string `json:"email"`
			Password string `json:"password"`
		}
		json.NewDecoder(r.Body).Decode(&creds)

		// Return mock JWT and default profile
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"token": "mock-jwt-token-xyz-12345",
			"user":  store.users[0],
		})
	})

	fmt.Println("👤 User Service running on http://localhost" + port)
	log.Fatal(http.ListenAndServe(port, mux))
}
