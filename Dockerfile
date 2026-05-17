ARG GO_IMAGE=golang:1.26.3-bookworm
ARG RUNTIME_IMAGE=debian:bookworm-slim

FROM ${GO_IMAGE} AS build

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
ARG NVIDIA_SMI_WEB_UI_VERSION="local"
RUN printf '%s\n' "${NVIDIA_SMI_WEB_UI_VERSION}" > .version
RUN CGO_ENABLED=1 GOOS=linux go build -o /out/nvidia-smi-web-ui .

FROM ${RUNTIME_IMAGE}

WORKDIR /app
COPY --from=build /out/nvidia-smi-web-ui /usr/local/bin/nvidia-smi-web-ui
COPY --from=build /src/.version .version
COPY --from=build /src/pkg/webui/static pkg/webui/static
COPY --from=build /src/pkg/webui/templates pkg/webui/templates

EXPOSE 8080

ENTRYPOINT ["nvidia-smi-web-ui"]
CMD ["web", "--addr", ":8080"]
