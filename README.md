# nvidia-smi-web-ui

[![GitHub Actions Workflow Status](https://img.shields.io/github/actions/workflow/status/deniskrumko/nvidia-smi-web-ui/build-image-and-push.yml)](https://github.com/deniskrumko/nvidia-smi-web-ui/actions)
[![GitHub Release](https://img.shields.io/github/v/release/deniskrumko/nvidia-smi-web-ui)](https://github.com/deniskrumko/nvidia-smi-web-ui/releases)
[![Docker pulls](https://img.shields.io/docker/pulls/deniskrumko/nvidia-smi-web-ui)](https://hub.docker.com/r/deniskrumko/nvidia-smi-web-ui/tags)

Web dashboard for monitoring NVIDIA GPUs through NVML. Like `nvidia-smi` but cooler 🤙

<img width="1309" height="743" alt="preview" src="https://github.com/user-attachments/assets/b7aa17e0-271f-4576-a169-de02b1ce41b1" />

`nvidia-smi-web-ui` runs as a local web application and shows live GPU utilization, memory usage, temperature, power, clocks, PCI details, ECC data, and other metrics exposed by NVIDIA Management Library. The browser keeps chart history in memory, so refreshing or closing the tab starts a fresh monitoring session.

Unsupported optional NVML metrics do not stop the application. They are reported as warnings because metric availability can depend on GPU generation, driver version, MIG mode, permissions, and platform.

## Run app using Docker

### Requirements

- Direct access to host with GPU devices
- Installed [Docker](https://docs.docker.com/engine/install/) to pull/run image
- Installed [NVIDIA Container Toolkit](https://docs.nvidia.com/datacenter/cloud-native/container-toolkit/latest/install-guide.html) to access GPUs from Docker

### How to run app

Pull the docker image (only `90.2MB`):

```bash
docker pull deniskrumko/nvidia-smi-web-ui:latest
```

Run the web UI on a GPU host:

```bash
docker run --rm \
    --gpus all \
    -p 8080:8080 \
    deniskrumko/nvidia-smi-web-ui:latest
```

Open the dashboard at http://localhost:8080

### Docker run examples

Run with custom page branding in the top-left corner of UI (`WEB_PAGE_BRANDING`) and tab title (`WEB_PAGE_TITLE`):

```bash
docker run --rm \
  --gpus all \
  -p 8080:8080 \
  -e WEB_PAGE_BRANDING="My GPU Monitor" \
  -e WEB_PAGE_TITLE="GPU Dashboard" \
  deniskrumko/nvidia-smi-web-ui:latest
```

Limit visible GPUs with Docker's GPU device selector:

```bash
docker run --rm \
  --gpus '"device=0,1"' \
  -p 8080:8080 \
  deniskrumko/nvidia-smi-web-ui:latest
```

## Run app from source

Use this path when you want to run the app directly with Go instead of Docker.

### Requirements

- Go 1.26.3 or newer
- Linux host with NVIDIA drivers and `libnvidia-ml.so.1` available at runtime
- NVIDIA GPU visible to NVML

### Run from a local checkout

Clone the repository and start the web server:

```bash
git clone https://github.com/deniskrumko/nvidia-smi-web-ui.git
cd nvidia-smi-web-ui
go run . web --addr :8080
```

Open the dashboard at http://localhost:8080

Run on another port:

```bash
go run . web --addr :9090
```

Run without NVML by using synthetic GPU data:

```bash
NVIDIA_SMI_WEB_UI_DEBUG=1 DEBUG_GPU_COUNT=4 go run . web --addr :8080
```

### Run directly from GitHub

If the project is available at [deniskrumko/nvidia-smi-web-ui](https://github.com/deniskrumko/nvidia-smi-web-ui), Go can download and run it directly:

```bash
go run github.com/deniskrumko/nvidia-smi-web-ui@latest web --addr :8080
```

Run a specific release tag:

```bash
go run github.com/deniskrumko/nvidia-smi-web-ui@v0.1.0 web --addr :8080
```

Install the binary into `GOBIN`:

```bash
go install github.com/deniskrumko/nvidia-smi-web-ui@latest
nvidia-smi-web-ui web --addr :8080
```

## Configuration

Runtime configuration is provided through environment variables. The image starts the web server on `:8080` by default.

| Name | Default | Description |
| --- | --- | --- |
| `WEB_PAGE_BRANDING` | `Nvidia SMI Web UI` | Text displayed as the dashboard branding. Also used as the page title when `WEB_PAGE_TITLE` is not set. |
| `WEB_PAGE_TITLE` | `WEB_PAGE_BRANDING` or `Nvidia SMI Web UI` | Browser page title. |
| `NVIDIA_SMI_WEB_UI_DEBUG` | unset | Enables synthetic GPU data and skips NVML initialization when set to `1`, `true`, or `yes`. |
| `DEBUG_GPU_COUNT` | `2` | Number of synthetic GPUs shown in debug mode. Invalid values fall back to the default. |

Override the HTTP listen address by passing a custom web command:

```bash
docker run --rm \
  --gpus all \
  -p 9090:9090 \
  deniskrumko/nvidia-smi-web-ui:latest web --addr :9090
```

Build-time Docker arguments:

| Name | Default | Description |
| --- | --- | --- |
| `GO_IMAGE` | `golang:1.26.3-bookworm` | Go builder image used by the Docker build. |
| `RUNTIME_IMAGE` | `debian:bookworm-slim` | Final runtime image used by the Docker build. |
| `NVIDIA_SMI_WEB_UI_VERSION` | `local` | Docker build argument written to `.version` and displayed by the web UI. |

## API

The web application serves the dashboard at `/` and exposes the current GPU snapshot at:

```text
GET /api/gpus
```

The endpoint is stateless. Each request reads a fresh NVML snapshot and returns JSON DTOs used by the dashboard.

Health checks can use:

```text
GET /api/health
```

It returns HTTP 200 with `{"status":"ok"}` only when the application can read at least one GPU through the configured GPU provider. If GPU access is unavailable, it returns HTTP 503.

## About this project

The project is written in Go. The backend uses the standard `net/http` server, Cobra for command wiring, and `github.com/NVIDIA/go-nvml` for NVML access. The web UI is server-rendered HTML plus static CSS and vanilla JavaScript.

There is no Node.js build step, package manager, bundler, or third-party JavaScript dependency tree. The Docker image only ships the Go binary, HTML templates, and static assets from this repository.

### Project structure

- `main.go`: signal-aware application entrypoint.
- `cmd/web`: web command wiring and environment-based configuration.
- `app/web`: HTTP server lifecycle.
- `app/gpu`: application layer for GPU snapshots.
- `pkg/nvmlclient`: NVML lifecycle, hardware adapter, snapshot collection, and warnings.
- `pkg/gpuinfo`: API-ready response DTOs.
- `pkg/webui`: server-rendered HTML shell, JSON API handlers, templates, and static assets.

NVML initialization and shutdown happen once per application run. Fatal NVML errors are limited to initialization, device count, invalid target device, and handle lookup failures.
