# CodeTracker Hooks - Go Binary Build

BINARY_SUBMIT = user_prompt_submit
BINARY_STOP = stop
VERSION ?= 1.0.0
BUILD_TIME = $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')
LDFLAGS = -s -w -X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME)

# Supported platforms
PLATFORMS = linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64

.PHONY: all build build-all clean test help

# Default target
all: build

# Build for current platform
build:
	@mkdir -p dist
	go build -ldflags "$(LDFLAGS)" -o dist/$(BINARY_SUBMIT) ./cmd/user_prompt_submit
	go build -ldflags "$(LDFLAGS)" -o dist/$(BINARY_STOP) ./cmd/stop
	@echo "Built binaries in dist/"

# Build for all platforms
build-all:
	@for platform in $(PLATFORMS); do \
		os=$${platform%/*}; \
		arch=$${platform#*/}; \
		output_dir=dist/$$os-$$arch; \
		mkdir -p $$output_dir; \
		echo "Building for $$os/$$arch..."; \
		ext=""; \
		if [ "$$os" = "windows" ]; then ext=".exe"; fi; \
		GOOS=$$os GOARCH=$$arch go build -ldflags "$(LDFLAGS)" \
			-o $$output_dir/$(BINARY_SUBMIT)$$ext ./cmd/user_prompt_submit; \
		GOOS=$$os GOARCH=$$arch go build -ldflags "$(LDFLAGS)" \
			-o $$output_dir/$(BINARY_STOP)$$ext ./cmd/stop; \
	done
	@echo "Built all platforms in dist/"

# Create release archives
release: build-all
	@for platform in $(PLATFORMS); do \
		os=$${platform%/*}; \
		arch=$${platform#*/}; \
		cd dist/$$os-$$arch && \
		if [ "$$os" = "windows" ]; then \
			zip -q ../codetracker-hooks-$$os-$$arch.zip *; \
		else \
			tar -czf ../codetracker-hooks-$$os-$$arch.tar.gz *; \
		fi; \
		cd - > /dev/null; \
	done
	@echo "Created release archives in dist/"

# Run tests
test:
	go test -v ./...

# Clean build artifacts
clean:
	rm -rf dist/

# Show help
help:
	@echo "CodeTracker Hooks Build System"
	@echo ""
	@echo "Targets:"
	@echo "  build      Build for current platform"
	@echo "  build-all  Build for all supported platforms"
	@echo "  release    Create release archives"
	@echo "  test       Run tests"
	@echo "  clean      Remove build artifacts"
	@echo ""
	@echo "Supported platforms:"
	@echo "  $(PLATFORMS)"
