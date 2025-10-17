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

# Run all tests in all packages (timeout 30s)
test:
	@echo "Using SDKROOT=$(SDKROOT)"
	CGO_ENABLED=1 go test -v -timeout 30s ./...

# Run all tests in one package with coverage (timeout 30s)
# Example: make test-pkg PKG=./internal/camera
test-pkg:
	@echo "Using SDKROOT=$(SDKROOT)"
	CGO_ENABLED=1 go test -v -timeout 30s -cover $(PKG)

# Run one test function in one package (timeout 30s)
# Example: make test-func PKG=./internal/camera FUNC=TestOpen
test-func:
	@echo "Using SDKROOT=$(SDKROOT)"
	CGO_ENABLED=1 go test -v -timeout 30s -run ^$(FUNC)$$ $(PKG)

# Run the main application
run:
	@echo "Using SDKROOT=$(SDKROOT)"
	go run ./cmd/cam