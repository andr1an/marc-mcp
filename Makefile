APP_NAME ?= marc-mcp

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
DATE ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

LDFLAGS := -s -w \
	-X 'main.version=$(VERSION)' \
	-X 'main.commit=$(COMMIT)' \
	-X 'main.date=$(DATE)'

GOOS_GOARCH ?= \
	linux/amd64 \
	linux/arm64 \
	darwin/amd64 \
	darwin/arm64 \
	windows/amd64

.PHONY: build build-all clean test print-version

build:
	CGO_ENABLED=0 go build -trimpath -ldflags "$(LDFLAGS)" -o $(APP_NAME) .

build-all:
	@set -e; \
	for target in $(GOOS_GOARCH); do \
		os=$${target%/*}; \
		arch=$${target#*/}; \
		out="$(APP_NAME)-$${os}-$${arch}"; \
		if [ "$$os" = "windows" ]; then out="$$out.exe"; fi; \
		echo "Building $$out"; \
		CGO_ENABLED=0 GOOS=$$os GOARCH=$$arch go build -trimpath -ldflags "$(LDFLAGS)" -o "$$out" .; \
	done

test:
	go test ./...

print-version:
	@echo "VERSION=$(VERSION)"
	@echo "COMMIT=$(COMMIT)"
	@echo "DATE=$(DATE)"

clean:
	rm -f $(APP_NAME) $(APP_NAME)-*
