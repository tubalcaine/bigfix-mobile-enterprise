# BigFix Enterprise Mobile (BEM) Server - Build System
#
# Targets:
#   make build          - Build for current platform
#   make release        - Build for all platforms (Linux, macOS, Windows)
#   make packages       - Create release archives (tar.gz and zip)
#   make clean          - Remove build artifacts
#   make test           - Run tests
#   make version        - Display version information

# Version information
VERSION := $(shell cat VERSION)
BUILD_DATE := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

# Binary name
BINARY_NAME := bem
PACKAGE_NAME := bigfix-mobile-enterprise

# Build directory
BUILD_DIR := build
DIST_DIR := dist

# Go build flags
LDFLAGS := -ldflags "-X main.Version=$(VERSION) -X main.BuildDate=$(BUILD_DATE) -X main.GitCommit=$(GIT_COMMIT)"

# Source files
GO_FILES := $(shell find . -name '*.go' -not -path "./vendor/*")

# Default target
.PHONY: all
all: build

# Build for current platform
.PHONY: build
build: $(BUILD_DIR)/$(BINARY_NAME)

$(BUILD_DIR)/$(BINARY_NAME): $(GO_FILES) VERSION
	@echo "Building $(BINARY_NAME) v$(VERSION) for current platform..."
	@mkdir -p $(BUILD_DIR)
	go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/bem/

# Build for all platforms
.PHONY: release
release: clean
	@echo "Building releases for all platforms..."
	@mkdir -p $(BUILD_DIR)

	@echo "Building Linux AMD64..."
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 ./cmd/bem/

	@echo "Building Linux ARM64..."
	GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64 ./cmd/bem/

	@echo "Building macOS AMD64..."
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 ./cmd/bem/

	@echo "Building macOS ARM64 (Apple Silicon)..."
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 ./cmd/bem/

	@echo "Building Windows AMD64..."
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe ./cmd/bem/

	@echo "Release builds complete!"
	@ls -lh $(BUILD_DIR)/

# Create distribution packages
.PHONY: packages
packages: release
	@echo "Creating distribution packages for v$(VERSION)..."
	@mkdir -p $(DIST_DIR)

	@echo "Creating source archive..."
	@git archive --format=tar.gz --prefix=$(PACKAGE_NAME)-$(VERSION)/ HEAD > $(DIST_DIR)/$(PACKAGE_NAME)-$(VERSION)-source.tar.gz
	@git archive --format=zip --prefix=$(PACKAGE_NAME)-$(VERSION)/ HEAD > $(DIST_DIR)/$(PACKAGE_NAME)-$(VERSION)-source.zip

	@echo "Creating binary archives..."
	# Linux AMD64
	@tar -czf $(DIST_DIR)/$(PACKAGE_NAME)-$(VERSION)-linux-amd64.tar.gz \
		-C $(BUILD_DIR) $(BINARY_NAME)-linux-amd64 \
		--transform 's/$(BINARY_NAME)-linux-amd64/$(BINARY_NAME)/'

	# Linux ARM64
	@tar -czf $(DIST_DIR)/$(PACKAGE_NAME)-$(VERSION)-linux-arm64.tar.gz \
		-C $(BUILD_DIR) $(BINARY_NAME)-linux-arm64 \
		--transform 's/$(BINARY_NAME)-linux-arm64/$(BINARY_NAME)/'

	# macOS AMD64
	@tar -czf $(DIST_DIR)/$(PACKAGE_NAME)-$(VERSION)-darwin-amd64.tar.gz \
		-C $(BUILD_DIR) $(BINARY_NAME)-darwin-amd64 \
		--transform 's/$(BINARY_NAME)-darwin-amd64/$(BINARY_NAME)/'

	# macOS ARM64
	@tar -czf $(DIST_DIR)/$(PACKAGE_NAME)-$(VERSION)-darwin-arm64.tar.gz \
		-C $(BUILD_DIR) $(BINARY_NAME)-darwin-arm64 \
		--transform 's/$(BINARY_NAME)-darwin-arm64/$(BINARY_NAME)/'

	# Windows AMD64 (zip)
	@cd $(BUILD_DIR) && zip -q ../$(DIST_DIR)/$(PACKAGE_NAME)-$(VERSION)-windows-amd64.zip $(BINARY_NAME)-windows-amd64.exe
	@cd $(DIST_DIR) && unzip -l $(PACKAGE_NAME)-$(VERSION)-windows-amd64.zip | grep $(BINARY_NAME)

	@echo "Packages created in $(DIST_DIR)/:"
	@ls -lh $(DIST_DIR)/

# Run tests
.PHONY: test
test:
	@echo "Running tests..."
	go test -v ./...

# Clean build artifacts
.PHONY: clean
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf $(BUILD_DIR)
	@rm -rf $(DIST_DIR)
	@rm -f bem bem-test
	@echo "Clean complete."

# Display version information
.PHONY: version
version:
	@echo "Version:     $(VERSION)"
	@echo "Build Date:  $(BUILD_DATE)"
	@echo "Git Commit:  $(GIT_COMMIT)"

# Install to system (requires sudo on Linux/macOS)
.PHONY: install
install: build
	@echo "Installing $(BINARY_NAME) to /usr/local/bin..."
	@install -m 755 $(BUILD_DIR)/$(BINARY_NAME) /usr/local/bin/$(BINARY_NAME)
	@echo "Install complete. Run 'bem --version' to verify."

# Uninstall from system
.PHONY: uninstall
uninstall:
	@echo "Uninstalling $(BINARY_NAME)..."
	@rm -f /usr/local/bin/$(BINARY_NAME)
	@echo "Uninstall complete."

# Show help
.PHONY: help
help:
	@echo "BEM Server Build System"
	@echo ""
	@echo "Targets:"
	@echo "  build          Build for current platform"
	@echo "  release        Build for all platforms (Linux, macOS, Windows)"
	@echo "  packages       Create release archives (tar.gz and zip)"
	@echo "  test           Run tests"
	@echo "  clean          Remove build artifacts"
	@echo "  version        Display version information"
	@echo "  install        Install to /usr/local/bin (requires sudo)"
	@echo "  uninstall      Remove from /usr/local/bin (requires sudo)"
	@echo "  help           Show this help message"
	@echo ""
	@echo "Current version: $(VERSION)"
