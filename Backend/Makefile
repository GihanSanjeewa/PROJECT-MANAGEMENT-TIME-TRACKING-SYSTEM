.PHONY: build run test test-race lint migrate-up migrate-down clean

BINARY_NAME=pulseflow-api

build:
	go build -o bin/$(BINARY_NAME) cmd/api/main.go

run:
	go run cmd/api/main.go

test:
	go test -v ./...

test-race:
	go test -race -v ./...

lint:
	golangci-lint run ./...

clean:
	rm -rf bin/

migrate-up:
	@echo "Applying migrations..."

migrate-down:
	@echo "Rolling back migrations..."
