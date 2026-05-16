APP_NAME ?= nvidia-smi-web-ui
IMAGE ?= $(APP_NAME):latest
GOBIN ?= $$(go env GOPATH)/bin

.PHONY: run
run:
	go run main.go list

.PHONY: run-json
run-json:
	go run main.go list --json

.PHONY: tidy
tidy:
	go mod tidy

.PHONY: download-dependencies
download-dependencies:
	go mod download

.PHONY: build
build: download-dependencies
	go build -o out/bin/$(APP_NAME) .

.PHONY: fmt
fmt:
	gofmt -s -w .

.PHONY: lint
lint:
	golangci-lint run

.PHONY: tests
tests:
	go run gotest.tools/gotestsum@latest -- ./...

.PHONY: coverage
coverage:
	go test ./... -coverprofile=cover.out -covermode=atomic
	go tool cover -func=cover.out

.PHONY: test-integration
test-integration:
	NVML_INTEGRATION=1 go test ./pkg/nvmlclient -run TestIntegrationNVML -count=1

.PHONY: docker-build
docker-build:
	docker build -t $(IMAGE) .

.PHONY: docker-run
docker-run:
	docker run --rm --gpus all $(IMAGE) list

.PHONY: install-tools
install-tools:
	go install gotest.tools/gotestsum@latest
	go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest
