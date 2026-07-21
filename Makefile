.PHONY: run test test-race fmt vet build clean

run:
	cd backend && go run ./cmd/api

test:
	cd backend && go test ./...

test-race:
	cd backend && go test -race -v ./...

fmt:
	cd backend && gofmt -w ./cmd ./internal

vet:
	cd backend && go vet ./...

build:
	cd backend && go build -o bin/inventory-api ./cmd/api

clean:
	cd backend && go clean
