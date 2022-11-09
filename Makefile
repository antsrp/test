run:
	go run ./cmd/api

build:
	@go mod tidy
	CGO_ENABLED=0 go build -o main ./cmd/api

swag:
	swag init -g cmd/api/main.go

test:
	go test -v ./...

docker-start:
	docker compose build --no-cache
	docker compose up 