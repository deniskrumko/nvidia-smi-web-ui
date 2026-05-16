# AGENTS.md

## Project Purpose

This repository is a Go backend foundation for collecting NVIDIA GPU data through NVML. The current product surface is a local Cobra CLI. Future work will add an HTTP API and web UI that reuse the same application layer and DTOs.

## Architecture Rules

- Keep CLI packages in `cmd/...` thin. They should parse flags, call `app/gpu`, and render output.
- Keep hardware access in `pkg/nvmlclient`. NVML initialization and shutdown must happen once per command.
- Keep API-ready response structs in `pkg/gpuinfo`.
- Keep rendering in `pkg/output`.
- Prefer small interfaces around NVML calls so tests do not require GPU hardware.
- Treat unsupported optional NVML metrics as warnings, not fatal errors.
- Fatal NVML errors should be limited to initialization, device count, invalid target device, or handle lookup failures.

## Commands

```bash
make fmt
make tests
make lint
make run
make docker-build
make docker-run
```

`make tests` runs:

```bash
go run gotest.tools/gotestsum@latest -- ./...
```

Optional GPU integration test:

```bash
NVML_INTEGRATION=1 go test ./pkg/nvmlclient -run TestIntegrationNVML -count=1
```

## Coding Guidelines

- Write code, identifiers, comments, README text, and errors in English.
- Prefer idiomatic, boring Go over clever abstractions.
- Keep exported names documented when they are part of package-level contracts.
- Use `gofmt -s -w .` before finishing changes.
- Do not require a GPU for default unit tests.
- Do not make optional NVML fields mandatory unless the product contract explicitly changes.

## GPU Runtime Notes

- `github.com/NVIDIA/go-nvml` depends on `libnvidia-ml.so.1` at runtime.
- Local macOS development can compile and run fake-based unit tests, but real NVML commands require a Linux NVIDIA driver environment.
- Docker runs require NVIDIA Container Toolkit and `--gpus all`.
