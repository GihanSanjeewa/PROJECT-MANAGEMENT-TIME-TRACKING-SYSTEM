package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"
)

type Task struct {
	ID          string    `json:"id"`
	ProjectID   string    `json:"project_id"`
	ColumnID    string    `json:"column_id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Priority    string    `json:"priority"` // Low, Medium, High, Urgent
	Assignee    string    `json:"assignee"`
	Order       int       `json:"order"`
	DueDate     string    `json:"due_date"`
	CreatedAt   time.Time `json:"created_at"`
}

type TaskStore struct {
	sync.RWMutex
	tasks []Task
}

var store = &TaskStore{
	tasks: []Task{
		{
			ID:          "task-1",
			ProjectID:   "proj-1",
			ColumnID:    "col-todo",
			Title:       "Set up Go Microservices Architecture",
			Description: "Configure API Gateway, User Service, Project Service, and Task Service with routing.",
			Priority:    "High",
			Assignee:    "Jordan Lee",
			Order:       1,
			DueDate:     "Oct 24",
			CreatedAt:   time.Now().Add(-48 * time.Hour),
		},
		{
			ID:          "task-2",
			ProjectID:   "proj-1",
			ColumnID:    "col-inprogress",
			Title:       "Design React Kanban Board UI",
			Description: "Implement sleek drag-and-drop columns, task priority badges, and quick creation modal.",
			Priority:    "Urgent",
			Assignee:    "Maya Patel",
			Order:       1,
			DueDate:     "Oct 22",
			CreatedAt:   time.Now().Add(-24 * time.Hour),
		},
		{
			ID:          "task-3",
			ProjectID:   "proj-1",
			ColumnID:    "col-review",
			Title:       "Implement JWT Authentication Middleware",
			Description: "Verify tokens on the API Gateway before forwarding requests to internal services.",
			Priority:    "Medium",
			Assignee:    "Alex Morgan",
			Order:       1,
			DueDate:     "Oct 25",
			CreatedAt:   time.Now().Add(-12 * time.Hour),
		},
		{
			ID:          "task-4",
			ProjectID:   "proj-1",
			ColumnID:    "col-done",
			Title:       "Initialize Workspace & Git Repository",
			Description: "Set up folder layout, Frontend and Backend structures.",
			Priority:    "Low",
			Assignee:    "Taylor Kim",
			Order:       1,
			DueDate:     "Oct 18",
			CreatedAt:   time.Now().Add(-72 * time.Hour),
		},
		{
			ID:          "task-5",
			ProjectID:   "proj-1",
			ColumnID:    "col-backlog",
			Title:       "Database Clustering & Replication",
			Description: "Evaluate PostgreSQL read replicas vs distributed event bus with Kafka.",
			Priority:    "Low",
			Assignee:    "Jordan Lee",
			Order:       1,
			DueDate:     "Nov 02",
			CreatedAt:   time.Now().Add(-5 * time.Hour),
		},
	},
}

func main() {
	port := ":8003"
	mux := http.NewServeMux()

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"healthy","service":"task-service"}`))
	})

	mux.HandleFunc("/api/tasks", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodPost {
			var newTask Task
			if err := json.NewDecoder(r.Body).Decode(&newTask); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			newTask.ID = fmt.Sprintf("task-%d", time.Now().UnixNano()%100000)
			newTask.CreatedAt = time.Now()
			if newTask.ColumnID == "" {
				newTask.ColumnID = "col-todo"
			}
			if newTask.Priority == "" {
				newTask.Priority = "Medium"
			}

			store.Lock()
			store.tasks = append(store.tasks, newTask)
			store.Unlock()

			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(newTask)
			return
		}

		// GET /api/tasks?project_id=...
		projectID := r.URL.Query().Get("project_id")
		store.RLock()
		defer store.RUnlock()

		if projectID == "" {
			json.NewEncoder(w).Encode(store.tasks)
			return
		}

		filtered := make([]Task, 0)
		for _, t := range store.tasks {
			if t.ProjectID == projectID {
				filtered = append(filtered, t)
			}
		}
		json.NewEncoder(w).Encode(filtered)
	})

	// /api/tasks/{id} or /api/tasks/{id}/move
	mux.HandleFunc("/api/tasks/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		path := strings.TrimPrefix(r.URL.Path, "/api/tasks/")

		// Handle /api/tasks/{id}/move
		if strings.HasSuffix(path, "/move") && (r.Method == http.MethodPatch || r.Method == http.MethodPost) {
			taskID := strings.TrimSuffix(path, "/move")
			var req struct {
				ColumnID string `json:"column_id"`
				Order    int    `json:"order"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}

			store.Lock()
			defer store.Unlock()
			for i, t := range store.tasks {
				if t.ID == taskID {
					store.tasks[i].ColumnID = req.ColumnID
					if req.Order > 0 {
						store.tasks[i].Order = req.Order
					}
					json.NewEncoder(w).Encode(store.tasks[i])
					return
				}
			}
			http.Error(w, `{"error":"Task not found"}`, http.StatusNotFound)
			return
		}

		// Handle DELETE /api/tasks/{id}
		if r.Method == http.MethodDelete {
			taskID := path
			store.Lock()
			defer store.Unlock()
			for i, t := range store.tasks {
				if t.ID == taskID {
					store.tasks = append(store.tasks[:i], store.tasks[i+1:]...)
					w.WriteHeader(http.StatusNoContent)
					return
				}
			}
			http.Error(w, `{"error":"Task not found"}`, http.StatusNotFound)
			return
		}
	})

	fmt.Println("📋 Task Service running on http://localhost" + port)
	log.Fatal(http.ListenAndServe(port, mux))
}
