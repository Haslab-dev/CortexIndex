BINARY_NAME=cortex
BUILD_DIR=bin
GO_MAIN=./cmd/cortex
VERSION=2.0.0

.PHONY: all build install clean

all: build

build:
	@echo "==> Building cortex v$(VERSION)..."
	@mkdir -p $(BUILD_DIR)
	go build -ldflags="-s -w" -o $(BUILD_DIR)/$(BINARY_NAME) $(GO_MAIN)
	@echo "✓ Built at $(BUILD_DIR)/$(BINARY_NAME)"

install: build
	@echo "==> Installing cortex v$(VERSION)..."
	@mkdir -p $(HOME)/.cortex/bin
	@cp $(BUILD_DIR)/$(BINARY_NAME) $(HOME)/.cortex/bin/$(BINARY_NAME)
	@chmod +x $(HOME)/.cortex/bin/$(BINARY_NAME)
	@echo "✓ Installed to $(HOME)/.cortex/bin/cortex"

clean:
	@rm -rf $(BUILD_DIR)
