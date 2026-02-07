VERSION ?= dev
LDFLAGS = -ldflags "-s -w -X main.version=$(VERSION)"

.PHONY: build clean test lint fmt run

build:
	go build $(LDFLAGS) -o bin/certwatch-webhook-cli .

clean:
	rm -rf bin/

test:
	go test ./...

lint:
	go vet ./...

fmt:
	gofmt -w .

run: build
	./bin/certwatch-webhook-cli $(ARGS)
