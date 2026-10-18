package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"
)

type Column struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Color string `json:"color"`
	Order int    `json:"order"`
}

type Project struct {
	ID          string    `json:"id"`
	Key         string    `json:"key"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Category    string    `json:"category"`
	Lead        string    `json:"lead"`
	TaskCount   int       `json:"task_count"`
	CreatedAt   time.Time `json:"created_at"`
}

type ProjectStore struct {
	sync.RWMutex
	projects []Project
	columns  []Column
}

var store = &ProjectStore{
	projects: []Project{
		{
			ID:          "proj-1",
			Key:         "CORE",
			Name:        "SaaS Core Platform",
			Description: "High-performance project management & team workflow system",
			Category:    "Engineering",
			Lead:        "Alex Morgan",
			TaskCount:   5,
			CreatedAt:   time.Now().Add(-24 * 7 * time.Hour),
		},
		{
			ID:          "proj-2",
			Key:         "MOB",
			Name:        "Mobile Application",
			Description: "React Native mobile client for iOS and Android",
			Category:    "Mobile",
			Lead:        "Maya Patel",
			TaskCount:   3,
			CreatedAt:   time.Now().Add(-24 * 3 * time.Hour),
		},
	},
	columns: []Column{
		{ID: "col-backlog", Title: "Backlog", Color: "#94a3b8", Order: 1},
		{ID: "col-todo", Title: "To Do", Color: "#60a5fa", Order: 2},
		{ID: "col-inprogress", Title: "In Progress", Color: "#f59e0b", Order: 3},
		{ID: "col-review", Title: "In Review", Color: "#a855f7", Order: 4},
		{ID: "col-done", Title: "Done", Color: "#10b981", Order: 5},
	},
}

func main() {
	port := ":8002"
	mux := http.NewServeMux()

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"healthy","service":"project-service"}`))
	})

	mux.HandleFunc("/api/projects/columns", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		store.RLock()
		defer store.RUnlock()
		json.NewEncoder(w).Encode(store.columns)
	})

	mux.HandleFunc("/api/projects", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodPost {
			var newProject Project
			if err := json.NewDecoder(r.Body).Decode(&newProject); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			newProject.ID = fmt.Sprintf("proj-%d", time.Now().Unix())
			newProject.CreatedAt = time.Now()
			store.Lock()
			store.projects = append(store.projects, newProject)
			store.Unlock()
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(newProject)
			return
		}

		store.RLock()
		defer store.RUnlock()
		json.NewEncoder(w).Encode(store.projects)
	})

	fmt.Println("📁 Project Service running on http://localhost" + port)
	log.Fatal(http.ListenAndServe(port, mux))
}
