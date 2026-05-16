ARG GO_IMAGE=golang:1.26.3-bookworm
ARG CUDA_RUNTIME_IMAGE=nvidia/cuda:13.0.1-base-ubuntu24.04

FROM ${GO_IMAGE} AS build

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=1 GOOS=linux go build -o /out/nvidia-smi-web-ui .

FROM ${CUDA_RUNTIME_IMAGE}

COPY --from=build /out/nvidia-smi-web-ui /usr/local/bin/nvidia-smi-web-ui

ENTRYPOINT ["nvidia-smi-web-ui"]
CMD ["help"]
