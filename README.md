# nvidia-smi-web-ui

[![GitHub Actions Workflow Status](https://img.shields.io/github/actions/workflow/status/deniskrumko/nvidia-smi-web-ui/build-image-and-push.yml)](https://github.com/deniskrumko/nvidia-smi-web-ui/actions)
[![GitHub Release](https://img.shields.io/github/v/release/deniskrumko/nvidia-smi-web-ui)](https://github.com/deniskrumko/nvidia-smi-web-ui/releases)
[![Docker pulls](https://img.shields.io/docker/pulls/deniskrumko/nvidia-smi-web-ui)](https://hub.docker.com/r/deniskrumko/nvidia-smi-web-ui/tags)

Web dashboard for monitoring NVIDIA GPUs through NVML. Like `nvidia-smi` but cooler 🤙

<img width="1309" height="743" alt="preview" src="https://github.com/user-attachments/assets/b7aa17e0-271f-4576-a169-de02b1ce41b1" />


`nvidia-smi-web-ui` runs as a local web application and shows live GPU utilization, memory usage, temperature, power, clocks, PCI details, ECC data, and other metrics exposed by NVIDIA Management Library. The browser keeps chart history in memory, so refreshing or closing the tab starts a fresh monitoring session.

Unsupported optional NVML metrics do not stop the application. They are reported as warnings because metric availability can depend on GPU generation, driver version, MIG mode, permissions, and platform.

## Docker

Pull the published image:

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

Open the dashboard at http://localhost:8080.

Run with custom page branding:

```bash
docker run --rm \
  --gpus all \
  -p 8080:8080 \
  -e WEB_PAGE_BRANDING="Server GPU Monitor" \
  -e WEB_PAGE_TITLE="GPU Dashboard" \
  deniskrumko/nvidia-smi-web-ui:latest
```

Run without NVML by using synthetic GPU data:

```bash
docker run --rm \
  -p 8080:8080 \
  -e NVIDIA_SMI_WEB_UI_DEBUG=1 \
  -e DEBUG_GPU_COUNT=4 \
  deniskrumko/nvidia-smi-web-ui:latest
```

Limit visible GPUs with Docker's GPU device selector:

```bash
docker run --rm \
  --gpus '"device=0,1"' \
  -p 8080:8080 \
  deniskrumko/nvidia-smi-web-ui:latest
```

Build the image locally:

```bash
docker build \
  --build-arg NVIDIA_SMI_WEB_UI_VERSION=local \
  -t deniskrumko/nvidia-smi-web-ui:latest .
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

## Architecture

- `main.go`: signal-aware application entrypoint.
- `cmd/web`: web command wiring and environment-based configuration.
- `app/web`: HTTP server lifecycle.
- `app/gpu`: application layer for GPU snapshots.
- `pkg/nvmlclient`: NVML lifecycle, hardware adapter, snapshot collection, and warnings.
- `pkg/gpuinfo`: API-ready response DTOs.
- `pkg/webui`: server-rendered HTML shell, JSON API handlers, templates, and static assets.

NVML initialization and shutdown happen once per application run. Fatal NVML errors are limited to initialization, device count, invalid target device, and handle lookup failures.
