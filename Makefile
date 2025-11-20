VERSION = $(shell git describe --tags --always --dirty)
# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
BINARY_NAME=nagini

.PHONY: build clean test version help

build:
	$(GOBUILD) -o $(BINARY_NAME) -ldflags="-X main.VERSION=$(VERSION)" -v

clean:
	$(GOCLEAN)
	rm -f $(BINARY_NAME)

test:
	$(GOTEST) -v ./...

version:
	@echo $(VERSION)

help:
	@echo "Available targets:"
	@echo "  build   - Build the binary with version information"
	@echo "  clean   - Clean build artifacts"
	@echo "  test    - Run tests"
	@echo "  version - Show current version"
	@echo "  help    - Show this help message"
	@echo ""
	@echo "Current version: $(VERSION)"
