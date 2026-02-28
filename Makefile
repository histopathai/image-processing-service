BINARY_NAME := himgproc
INSTALL_PATH := /usr/local/bin

.PHONY: build install uninstall clean

build:
	@echo "🔨 Building $(BINARY_NAME)..."
	go build -o $(BINARY_NAME) ./cmd/main.go
	@echo "✅ Built: ./$(BINARY_NAME)"

install:
	@echo "📦 Installing $(BINARY_NAME) to $(INSTALL_PATH)..."
	install -m 0755 $(BINARY_NAME) $(INSTALL_PATH)/$(BINARY_NAME)
	@echo "✅ Installed: $(INSTALL_PATH)/$(BINARY_NAME)"

uninstall:
	@echo "🗑️  Removing $(INSTALL_PATH)/$(BINARY_NAME)..."
	rm -f $(INSTALL_PATH)/$(BINARY_NAME)
	@echo "✅ Uninstalled"

clean:
	@echo "🧹 Cleaning..."
	rm -f $(BINARY_NAME)
	@echo "✅ Clean"