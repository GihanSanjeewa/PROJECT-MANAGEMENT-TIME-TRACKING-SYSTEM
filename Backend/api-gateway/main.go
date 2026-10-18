package main

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
)

func createReverseProxy(target string) (*httputil.ReverseProxy, error) {
	parsedURL, err := url.Parse(target)
	if err != nil {
		return nil, err
	}
	return httputil.NewSingleHostReverseProxy(parsedURL), nil
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func main() {
	port := ":8000"

	userProxy, _ := createReverseProxy("http://localhost:8001")
	projectProxy, _ := createReverseProxy("http://localhost:8002")
	taskProxy, _ := createReverseProxy("http://localhost:8003")

	mux := http.NewServeMux()

	// Health check
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"healthy","service":"api-gateway","version":"1.0.0"}`))
	})

	// Dynamic routing to microservices
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		switch {
		case strings.HasPrefix(path, "/api/auth") || strings.HasPrefix(path, "/api/users"):
			userProxy.ServeHTTP(w, r)
		case strings.HasPrefix(path, "/api/projects"):
			projectProxy.ServeHTTP(w, r)
		case strings.HasPrefix(path, "/api/tasks"):
			taskProxy.ServeHTTP(w, r)
		default:
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(`{"error":"Route not found in API Gateway"}`))
		}
	})

	handler := corsMiddleware(mux)

	fmt.Println("==================================================")
	fmt.Println("🚀 API Gateway running on http://localhost" + port)
	fmt.Println("   ├── /api/auth   -> User Service    (:8001)")
	fmt.Println("   ├── /api/projects -> Project Service (:8002)")
	fmt.Println("   └── /api/tasks  -> Task Service    (:8003)")
	fmt.Println("==================================================")
	log.Fatal(http.ListenAndServe(port, handler))
}
