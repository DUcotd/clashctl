.PHONY: build clean test lint fmt vet all release

APP_NAME := clashctl
VERSION := $(shell grep '^\s*AppVersion' internal/core/defaults.go | sed 's/.*"\(v[^"]*\)".*/\1/')
LDFLAGS := -s -w -X clashctl/internal/core.AppVersion=$(VERSION)

PLATFORMS := linux/amd64 linux/arm64 linux/arm

# Default build for current platform
build:
	go build -ldflags="$(LDFLAGS)" -o $(APP_NAME) .

# Build for all release platforms
release:
	@for platform in $(PLATFORMS); do \
		GOOS=$${platform%/*}; \
		GOARCH=$${platform#*/}; \
		output="$(APP_NAME)-$${GOOS}-$${GOARCH}"; \
		echo "Building $${output}..."; \
		CGO_ENABLED=0 GOOS=$${GOOS} GOARCH=$${GOARCH} \
			go build -ldflags="$(LDFLAGS)" -o $${output} . || exit 1; \
	done
	@sha256sum $(APP_NAME)-* > checksums-sha256.txt
	@echo "✅ Release binaries built:"
	@ls -lh $(APP_NAME)-*
	@echo ""
	@echo "📋 SHA256 checksums:"
	@cat checksums-sha256.txt

# Run tests
test:
	go test ./... -v -count=1

# Run tests with coverage
cover:
	go test ./... -coverprofile=coverage.out
	go tool cover -html=coverage.out -o coverage.html

# Format code
fmt:
	gofmt -s -w .

# Vet code
vet:
	go vet ./...

# Lint (requires golangci-lint)
lint:
	golangci-lint run ./...

# Clean build artifacts
clean:
	rm -f $(APP_NAME) $(APP_NAME)-* coverage.out coverage.html

# Full check: fmt, vet, test
check: fmt vet test

# Install to /usr/local/bin
install: build
	install -m 755 $(APP_NAME) /usr/local/bin/$(APP_NAME)

# Print version
version:
	@echo $(VERSION)
