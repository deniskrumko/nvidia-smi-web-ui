.PHONY: list web debug tidy download-dependencies build fmt lint tests coverage check test-integration docker-build docker-run install-tools

APP_NAME ?= nvidia-smi-web-ui
IMAGE ?= $(APP_NAME):latest
GOBIN ?= $$(go env GOPATH)/bin

list:
	go run main.go list

web:
	go run main.go web

debug:
	NVIDIA_SMI_WEB_UI_DEBUG=1 go run main.go web

tidy:
	go mod tidy

download-dependencies:
	go mod download

build: download-dependencies
	go build -o out/bin/$(APP_NAME) .

fmt:
	gofmt -s -w .

lint:
	golangci-lint run

tests:
	go test ./...

coverage:
	go test ./... -coverprofile=cover.out -covermode=atomic
	go tool cover -func=cover.out

check: fmt lint tests

test-integration:
	NVML_INTEGRATION=1 go test ./pkg/nvmlclient -run TestIntegrationNVML -count=1

docker-build:
	docker build -t $(IMAGE) .

docker-run:
	docker run --rm --gpus all $(IMAGE) list

install-tools:
	go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest
