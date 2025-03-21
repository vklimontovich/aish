# Variables
BINARY_NAME=ai
BUILD_DIR=build
BUILD_PATH=$(BUILD_DIR)/$(BINARY_NAME)
TARGET_DIR=$(HOME)/bin
TARGET_PATH=$(TARGET_DIR)/$(BINARY_NAME)

# Default target
.PHONY: build
build:
	@echo "🔧 Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	@go build -o $(BUILD_PATH) .
	@echo "✅ Build complete: $(BUILD_PATH)"

# Build and install locally if LOCAL_BIN is set
.PHONY: install
install: build
ifeq ($(LOCAL_BIN),true)
	@echo "🚀 Installing to $(TARGET_PATH)..."
	@mkdir -p $(TARGET_DIR)
	@if [ -f "$(TARGET_PATH)" ]; then \
		echo "🗑️ Removing existing binary at $(TARGET_PATH)"; \
		rm -f "$(TARGET_PATH)"; \
	fi
	@cp "$(BUILD_PATH)" "$(TARGET_PATH)"
	@chmod +x "$(TARGET_PATH)"
	@echo "✅ Installed $(BINARY_NAME) to $(TARGET_PATH)"
else
	@echo "ℹ️ LOCAL_BIN not set. Skipping install to $(TARGET_PATH)"
endif

# Clean build artifacts
.PHONY: clean
clean:
	@echo "🧹 Cleaning build artifacts..."
	@rm -rf $(BUILD_DIR)