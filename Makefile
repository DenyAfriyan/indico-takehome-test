.PHONY: run run-backend run-frontend frontend-install frontend-build test test-race fmt vet build clean

run: run-backend

run-backend:
	cd backend && go run ./cmd/api

run-frontend:
	cd frontend && pnpm dev

frontend-install:
	cd frontend && pnpm install

frontend-build:
	cd frontend && pnpm build

test:
	cd backend && go test ./...

test-race:
	cd backend && go test -race -v ./...

fmt:
	cd backend && go fmt ./...

vet:
	cd backend && go vet ./...

build:
	cd backend && go build -o bin/inventory-api ./cmd/api

clean:
	cd backend && go clean
