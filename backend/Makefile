.PHONY: tidy run test build docker

tidy:
	go mod tidy

run:
	go run ./cmd/api

test:
	go test ./...

build:
	go build -trimpath -o bin/api ./cmd/api

docker:
	docker build -t wa-saas-api:latest .
