.PHONY: build test lint docker-up docker-down migrate run-index clean

BINARY=goatlas

build:
	go build -o $(BINARY) .

test:
	go test ./...

lint:
	golangci-lint run ./...

docker-up:
	docker compose up -d

docker-down:
	docker compose down

migrate:
	go run . migrate

run-index:
	go run . index $(REPO_PATH)

clean:
	rm -f $(BINARY)
