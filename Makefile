.PHONY: build run test docker-build docker-up docker-down

APP_NAME=todobot
DOCKER_IMAGE=todobot:latest

build:
	go build -o bin/$(APP_NAME) ./cmd/todobot/main.go

run: build
	./bin/$(APP_NAME)

test:
	go test ./...

docker-build:
	docker build -t $(DOCKER_IMAGE) .

docker-up:
	docker-compose up -d

docker-down:
	docker-compose down

lint:
	golangci-lint run ./...
