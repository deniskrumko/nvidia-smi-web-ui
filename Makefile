.PHONY: list web debug debug-remote-hosts tidy deps build fmt lint tests coverage check test-integration docker-build docker-run install-tools

APP_NAME ?= nvidia-smi-web-ui
IMAGE ?= $(APP_NAME):latest
GOBIN ?= $$(go env GOPATH)/bin
DEBUG_MODE_GPU_COUNT ?= 8
DEBUG_REMOTE_HOST_COUNT ?= 4
DEBUG_REMOTE_BASE_PORT ?= 18080
DEBUG_REMOTE_UI_PORT ?= 8080

# Show list of available GPUs and their details
list:
	go run main.go list

# Run web application
web:
	go run main.go web

# Run web application in debug mode
debug:
	@DEBUG_MODE_ENABLED=1 DEBUG_MODE_GPU_COUNT=$(DEBUG_MODE_GPU_COUNT) go run main.go web

# Run web application in debug mode with multiple fake remote hosts
debug-remote-hosts:
	@set -e; \
	if [ $(DEBUG_REMOTE_HOST_COUNT) -lt 1 ]; then \
		echo "DEBUG_REMOTE_HOST_COUNT must be greater than 0"; \
		exit 1; \
	fi; \
	pids=""; \
	remote_hosts=""; \
	cleanup() { \
		for pid in $$pids; do \
			kill $$pid >/dev/null 2>&1 || true; \
		done; \
	}; \
	trap cleanup EXIT INT TERM; \
	index=0; \
	while [ $$index -lt $(DEBUG_REMOTE_HOST_COUNT) ]; do \
		port=$$(( $(DEBUG_REMOTE_BASE_PORT) + $$index )); \
		host_number=$$(( $$index + 1 )); \
		echo "Starting fake GPU host $$host_number on http://localhost:$$port"; \
		DEBUG_MODE_ENABLED=1 DEBUG_MODE_GPU_COUNT=$(DEBUG_MODE_GPU_COUNT) UI_BRANDING="Fake GPU Host $$host_number" go run main.go web --addr :$$port & \
		pids="$$pids $$!"; \
		remote_hosts="$$remote_hosts REMOTE_HOST_$${index}_DISPLAY_NAME=fake-host-$$host_number REMOTE_HOST_$${index}_HOST_NAME=http://localhost:$$port"; \
		index=$$(( $$index + 1 )); \
	done; \
	remote_hosts="$$remote_hosts REMOTE_HOST_0_DEFAULT=true"; \
	echo "Starting debug multi-host UI on http://localhost:$(DEBUG_REMOTE_UI_PORT)"; \
	env $$remote_hosts go run main.go web --addr :$(DEBUG_REMOTE_UI_PORT)

# Clean up go.mod and go.sum files
tidy:
	go mod tidy

# Download all dependencies
deps:
	go mod download

# Build the application using Go
build: deps
	go build -o out/bin/$(APP_NAME) .

# Format code
fmt:
	gofmt -s -w .

# Lint code
lint:
	golangci-lint run

# Run tests
tests:
	go test ./...

# Run tests with coverage
coverage:
	go test ./... -coverprofile=cover.out -covermode=atomic
	go tool cover -func=cover.out

# Run basic checks
check: fmt lint tests

# Run integration tests
test-integration:
	NVML_INTEGRATION=1 go test ./pkg/nvmlclient -run TestIntegrationNVML -count=1

# Build a Docker image
docker-build:
	docker build -t $(IMAGE) .

# Run the application in a Docker
docker-run:
	docker run --rm --gpus all -p 8080:8080 $(IMAGE)

# Install extra tools
install-tools:
	go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest
