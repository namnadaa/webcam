# ==== APP ====
APP_NAME=webcam
BUILD_DIR=build
MAIN=./cmd/cam

# Get macOS SDK path (required for CGO)
SDKROOT := $(shell xcrun --show-sdk-path)
# Path to C++ standard library headers
CPATH := $(SDKROOT)/usr/include/c++/v1
# Path to OpenCV pkg-config files
PKG_CONFIG_PATH := /opt/homebrew/opt/opencv/lib/pkgconfig

# Export environment variables for all commands below
export SDKROOT
export CPATH
export PKG_CONFIG_PATH
export CGO_ENABLED=1

.PHONY: test test-hardware test-pkg test-func run build build-mac clean release

# ==== TESTS ====
# Run all tests, skip hardware (camera) tests
test:
	@echo "Using SDKROOT=$(SDKROOT)"
	go test -short -v -timeout 30s ./...

# Run all tests including hardware (requires a connected camera)
test-hardware:
	@echo "Using SDKROOT=$(SDKROOT)"
	go test -v -timeout 30s ./...

# Run all tests in one package with coverage (timeout 30s)
# Example: make test-pkg PKG=./internal/camera
test-pkg:
	@echo "Using SDKROOT=$(SDKROOT)"
	go test -v -timeout 30s -cover $(PKG)

# Run one test function in one package (timeout 30s)
# Example: make test-func PKG=./internal/camera FUNC=TestOpen
test-func:
	@echo "Using SDKROOT=$(SDKROOT)"
	go test -v -timeout 30s -run ^$(FUNC)$$ $(PKG)

# ==== RUN ====
# Run the main application
run:
	@echo "Using SDKROOT=$(SDKROOT)"
	go run $(MAIN)

# ==== BUILD ====
build: build-mac

# Build macOS application (ARM64)
# Compiles the project for Apple Silicon (M1/M2/M3) Macs
# and signs the binary for local execution (codesign -s -)
build-mac:
	mkdir -p $(BUILD_DIR)
	GOOS=darwin GOARCH=arm64 go build -o $(BUILD_DIR)/$(APP_NAME)-macos $(MAIN)
	codesign -s - $(BUILD_DIR)/$(APP_NAME)-macos

# ==== CLEAN ====
# Remove all build artifacts
# Deletes the entire build directory
clean:
	rm -rf $(BUILD_DIR)

# ==== RELEASE ====
# Create a macOS release package
# Builds the macOS binary, copies README, and zips everything into a release archive
release: clean build
	cp README.md $(BUILD_DIR)/
	cd $(BUILD_DIR) && zip $(APP_NAME)-release.zip webcam-macos README.md