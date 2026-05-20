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

Run with custom page branding in the top-left corner of UI (`UI_BRANDING`) and tab title (`UI_TITLE`):

```bash
docker run --rm \
  --gpus all \
  -p 8080:8080 \
  -e UI_BRANDING="My GPU Monitor" \
  -e UI_TITLE="GPU Dashboard" \
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
DEBUG_MODE_ENABLED=1 DEBUG_MODE_GPU_COUNT=4 go run . web --addr :8080
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

## Multi-host configuration

By default, the web UI reads GPU data from the same process that serves the dashboard. To use one dashboard as a shared entry point for several existing `nvidia-smi-web-ui` servers, configure remote hosts with indexed environment variables:

```bash
REMOTE_HOST_0_DISPLAY_NAME="test"
REMOTE_HOST_0_HOST_NAME="nvidia-web-ui-rnd-kube.examoke.com"
REMOTE_HOST_0_DEFAULT="true"
```

Add more hosts by incrementing the index: `REMOTE_HOST_1_DISPLAY_NAME`, `REMOTE_HOST_1_HOST_NAME`, and so on. Indexes must start at `0` and be contiguous because the UI stores the selected host as `host=N` in the URL. `REMOTE_HOST_*_DISPLAY_NAME` is displayed in the UI host selector. `REMOTE_HOST_*_HOST_NAME` must point to the remote server origin without a path, for example `http://nvidia-web-ui-gpu-1080-ti-main:8080`. Host names without a scheme default to `https://`.

`REMOTE_HOST_*_PATH` is optional and defaults to `/api/gpus`:

```bash
REMOTE_HOST_1_DISPLAY_NAME="1080 Ti"
REMOTE_HOST_1_HOST_NAME="http://nvidia-web-ui-gpu-1080-ti-${DEPLOY_BRANCH}:8080"
REMOTE_HOST_1_PATH="/api/gpus"
```

When at least one `REMOTE_HOST_*` entry is configured, the local NVML provider is disabled for the web command. The local process only serves the dashboard and proxies `/api/gpus?host=N` to the selected remote host. The local host is not added to the selector automatically.

If no remote hosts are configured, the selector is replaced with static text:

```text
Host: local
```

`REMOTE_HOST_*_DEFAULT` is optional and accepts `1`, `true`, or `yes`. If no default is set, the first configured host is selected. Only one host can be marked as default.

## Configuration

Runtime configuration is provided through environment variables. The image starts the web server on `:8080` by default.

| Name | Default | Description |
| --- | --- | --- |
| `UI_BRANDING` | `Nvidia SMI Web UI` | Text displayed as the dashboard branding. Also used as the page title when `UI_TITLE` is not set. |
| `UI_TITLE` | `UI_BRANDING` or `Nvidia SMI Web UI` | Browser page title. |
| `DEBUG_MODE_ENABLED` | unset | Enables synthetic GPU data and skips NVML initialization when set to `1`, `true`, or `yes`. |
| `DEBUG_MODE_GPU_COUNT` | `2` | Number of synthetic GPUs shown in debug mode. Invalid values fall back to the default. |
| `LOG_ACCESS_LOG_LEVEL` | `info` | Log level used only for JSON HTTP access logs: `debug`, `info`, `warn`, or `error`. |
| `LOG_ACCESS_LOG_ENABLED` | `false` | Enables JSON HTTP access logs when set to `1`, `true`, or `yes`. Set to `false` or `0` to disable them. |

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

It returns HTTP 200 with `{"status":"ok"}` when the web server is running. It does not check local or remote GPU access.

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
