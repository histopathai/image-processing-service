BINARY_NAME := himgproc
INSTALL_PATH := /usr/local/bin
OS := $(shell uname -s)

.PHONY: build install uninstall clean deps deps-uninstall

deps:
ifeq ($(OS),Darwin)
	@echo "🍎 Installing dependencies for macOS..."
	brew install vips openslide exiftool
else
	@echo "🐧 Installing dependencies for Linux..."
	sudo apt update
	sudo apt install -y libvips-tools libopenslide-bin libimage-exiftool-perl
endif
	@echo "✅ Dependencies installed"

deps-uninstall:
ifeq ($(OS),Darwin)
	@echo "🍎 Uninstalling dependencies for macOS..."
	brew uninstall exiftool openslide vips
else
	@echo "🐧 Uninstalling dependencies for Linux..."
	sudo apt remove -y libvips-tools libopenslide-bin libimage-exiftool-perl
endif
	@echo "✅ Dependencies uninstalled"

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