# nvidia-smi-web-ui

CLI-first Go backend foundation and local web UI for collecting NVIDIA GPU information through NVML.

The project currently provides a local CLI similar in spirit to `nvidia-smi` plus a single-page monitoring dashboard for live GPU charts.

## Requirements

- Go 1.26.3 or newer toolchain.
- Linux host with NVIDIA drivers and `libnvidia-ml.so.1` available at runtime.
- NVIDIA GPU visible to NVML.
- Docker GPU execution requires NVIDIA Container Toolkit and `docker run --gpus all`.

The project can compile and run unit tests on non-GPU machines. Real NVML commands must run on a machine with NVIDIA drivers.

## Local Usage

Show CLI help:

```bash
go run main.go
```

List GPUs:

```bash
go run main.go list
```

List GPUs as JSON:

```bash
go run main.go list --json
```

Inspect one GPU by index or UUID:

```bash
go run main.go inspect --id 0
go run main.go inspect --uuid GPU-00000000-0000-0000-0000-000000000000
```

List GPU processes:

```bash
go run main.go processes
```

Run the local web UI:

```bash
go run main.go web
```

The web UI serves a stateless API at `/api/gpus` and keeps chart history only in the memory of the open browser tab. Refreshing or closing the tab clears the collected chart data.

Optional web UI branding:

```bash
WEB_PAGE_BRANDING="Server GPU Monitor" go run main.go web
WEB_PAGE_TITLE="GPU Dashboard" WEB_PAGE_BRANDING="Server GPU Monitor" go run main.go web
```

Useful flags:

- `--json`: render machine-readable JSON.
- `--warnings`: include per-field NVML collection warnings in table mode.
- `--no-processes`: skip process collection for `list` and `inspect`.

Unsupported metrics do not fail the whole command. They are recorded as warnings because NVML support differs by GPU generation, driver version, MIG mode, permissions, and platform.

## Make Commands

```bash
make run
make web
make fmt
make lint
make tests
make docker-build
make docker-run
```

`make tests` runs the standard Go test suite with `go test ./...`.

## Docker

Build the image:

```bash
make docker-build
```

Run on a GPU host:

```bash
docker run --rm --gpus all nvidia-smi-web-ui:latest list
docker run --rm --gpus all nvidia-smi-web-ui:latest list --json
```

The default runtime image is `debian:bookworm-slim`. A CUDA-based image is not required for NVML-only reads: `libnvidia-ml.so.1` is provided by the NVIDIA driver on the host and mounted into the container by NVIDIA Container Toolkit when the container is started with `--gpus all`.

The runtime image can be overridden:

```bash
docker build \
  --build-arg RUNTIME_IMAGE=debian:bookworm-slim \
  -t nvidia-smi-web-ui:latest .
```

The previous CUDA runtime variant is kept as `Dockerfile.cuda` for reference:

```bash
docker build -f Dockerfile.cuda -t nvidia-smi-web-ui:cuda .
```

## Architecture

- `main.go`: thin entrypoint with signal-aware context.
- `cmd/...`: Cobra command wiring.
- `app/gpu`: use-case layer for CLI and future API transports.
- `app/web`: HTTP server lifecycle for the local web UI.
- `pkg/nvmlclient`: NVML lifecycle, hardware adapter, snapshot collection, warnings.
- `pkg/gpuinfo`: exported DTOs intended for future HTTP responses.
- `pkg/output`: JSON and table rendering.
- `pkg/webui`: server-rendered HTML shell, stateless JSON API handlers, templates, and static assets.

The NVML client initializes NVML once per command and shuts it down at the same ownership level. Fatal errors are limited to initialization, device count, and device handle lookup. Optional per-metric failures become structured warnings.

## Testing

Run unit tests:

```bash
make tests
```

Run the optional integration test on a Linux GPU host:

```bash
NVML_INTEGRATION=1 go test ./pkg/nvmlclient -run TestIntegrationNVML -count=1
```

The default test suite uses fake NVML implementations, so it does not require GPU hardware.
