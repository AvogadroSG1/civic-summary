BINARY := civic-summary
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
DATE := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS := -ldflags "-X github.com/AvogadroSG1/civic-summary/cmd.version=$(VERSION) \
	-X github.com/AvogadroSG1/civic-summary/cmd.commit=$(COMMIT) \
	-X github.com/AvogadroSG1/civic-summary/cmd.date=$(DATE)"

.PHONY: build test lint clean coverage install check setup

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

check:
	@bash scripts/check-prerequisites.sh

setup: check
	@mkdir -p $(HOME)/.civic-summary/templates
	@if [ ! -f $(HOME)/.civic-summary/config.yaml ]; then \
		cp config.example.yaml $(HOME)/.civic-summary/config.yaml; \
		echo "Created $(HOME)/.civic-summary/config.yaml from example"; \
	else \
		echo "Config already exists: $(HOME)/.civic-summary/config.yaml"; \
	fi
	@echo ""
	@echo "Next steps:"
	@echo "  1. Edit $(HOME)/.civic-summary/config.yaml with your settings"
	@echo "  2. Copy a prompt template to $(HOME)/.civic-summary/templates/"
	@echo "  3. Run: make build"
	@echo "  4. Run: ./civic-summary discover --body=your-body"

release:
	goreleaser release --snapshot --clean
