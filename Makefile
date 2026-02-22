BINARY := civic-summary
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
DATE := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS := -ldflags "-X github.com/AvogadroSG1/civic-summary/cmd.version=$(VERSION) \
	-X github.com/AvogadroSG1/civic-summary/cmd.commit=$(COMMIT) \
	-X github.com/AvogadroSG1/civic-summary/cmd.date=$(DATE)"

.PHONY: build test lint clean coverage install

build:
	go build $(LDFLAGS) -o $(BINARY) .

install:
	go install $(LDFLAGS) .

test:
	go test ./... -v

lint:
	golangci-lint run

coverage:
	go test ./... -coverprofile=coverage.out -covermode=atomic
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"
	@go tool cover -func=coverage.out | tail -1

clean:
	rm -f $(BINARY) coverage.out coverage.html

release:
	goreleaser release --snapshot --clean
