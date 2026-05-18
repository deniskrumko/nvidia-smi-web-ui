package webui_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/deniskrumko/nvidia-smi-web-ui/pkg/gpuinfo"
	"github.com/deniskrumko/nvidia-smi-web-ui/pkg/webui"
)

func TestNewHandlerRendersIndexWithBranding(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	response := httptest.NewRecorder()

	webui.NewHandler(webui.Config{
		Branding: "Lab GPU Monitor",
		Title:    "Lab Dashboard",
	}).ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, response.Code)
	}
	if contentType := response.Header().Get("Content-Type"); contentType != "text/html; charset=utf-8" {
		t.Fatalf("expected HTML content type, got %q", contentType)
	}
	body := response.Body.String()
	if !strings.Contains(body, "<title>Lab Dashboard</title>") {
		t.Fatalf("expected configured title, got %q", body)
	}
	if !strings.Contains(body, "Lab GPU Monitor") {
		t.Fatalf("expected configured branding, got %q", body)
	}
	if !strings.Contains(body, "deniskrumko/nvidia-smi-web-ui local") {
		t.Fatalf("expected local version, got %q", body)
	}
	for _, unexpected := range []string{"Live GPU monitoring", "History is stored only in this browser tab"} {
		if strings.Contains(body, unexpected) {
			t.Fatalf("expected index not to contain %q, got %q", unexpected, body)
		}
	}
	for _, expected := range []string{"/static/logo_s.png", "data-gpu-trigger", "data-chart-trigger", "data-time-minutes", "data-refresh-interval"} {
		if !strings.Contains(body, expected) {
			t.Fatalf("expected index to contain %q, got %q", expected, body)
		}
	}
	for _, expected := range []string{"/favicon/favicon-96x96.png", "/favicon/favicon.svg", "/favicon/favicon.ico", "/favicon/apple-touch-icon.png", "/favicon/site.webmanifest"} {
		if !strings.Contains(body, expected) {
			t.Fatalf("expected index to contain favicon markup %q, got %q", expected, body)
		}
	}
}

func TestNewHandlerRendersConfiguredVersion(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	response := httptest.NewRecorder()

	webui.NewHandler(webui.Config{Version: "v1.0.0"}).ServeHTTP(response, request)

	if body := response.Body.String(); !strings.Contains(body, "deniskrumko/nvidia-smi-web-ui v1.0.0") {
		t.Fatalf("expected configured version, got %q", body)
	}
}

func TestNewHandlerReadsVersionFile(t *testing.T) {
	tempDir := t.TempDir()
	t.Chdir(tempDir)
	if err := os.WriteFile(".version", []byte("v1.2.3\n"), 0o600); err != nil {
		t.Fatalf("write version file: %v", err)
	}
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	response := httptest.NewRecorder()

	webui.NewHandler(webui.Config{}).ServeHTTP(response, request)

	if body := response.Body.String(); !strings.Contains(body, "deniskrumko/nvidia-smi-web-ui v1.2.3") {
		t.Fatalf("expected file version, got %q", body)
	}
}

func TestNewHandlerUsesDefaultBranding(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	response := httptest.NewRecorder()

	webui.NewHandler(webui.Config{}).ServeHTTP(response, request)

	if body := response.Body.String(); !strings.Contains(body, "Nvidia SMI Web UI") {
		t.Fatalf("expected default branding, got %q", body)
	}
}

func TestNewHandlerServesStaticDirectory(t *testing.T) {
	staticDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(staticDir, "generated.txt"), []byte("static file"), 0o600); err != nil {
		t.Fatalf("write static file: %v", err)
	}
	if err := os.Mkdir(filepath.Join(staticDir, "favicon"), 0o700); err != nil {
		t.Fatalf("make favicon dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(staticDir, "favicon", "generated.txt"), []byte("favicon file"), 0o600); err != nil {
		t.Fatalf("write favicon file: %v", err)
	}
	tests := []struct {
		path string
		want string
	}{
		{path: "/static/generated.txt", want: "static file"},
		{path: "/favicon/generated.txt", want: "favicon file"},
	}

	for _, test := range tests {
		t.Run(test.path, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, test.path, nil)
			response := httptest.NewRecorder()

			webui.NewHandler(webui.Config{StaticDir: staticDir}).ServeHTTP(response, request)

			if response.Code != http.StatusOK {
				t.Fatalf("expected status %d, got %d", http.StatusOK, response.Code)
			}
			if body := response.Body.String(); body != test.want {
				t.Fatalf("expected body %q, got %q", test.want, body)
			}
		})
	}
}

func TestGPUAPIReturnsSnapshot(t *testing.T) {
	collectedAt := time.Date(2026, 5, 16, 10, 30, 0, 0, time.UTC)
	index := 0
	name := "NVIDIA Test GPU"
	uuid := "GPU-test"
	provider := &fakeProvider{
		snapshot: gpuinfo.Snapshot{
			Devices: []gpuinfo.Device{
				{
					Index: &index,
					Name:  &name,
					UUID:  &uuid,
					Memory: &gpuinfo.MemoryInfo{
						TotalBytes: 16,
						UsedBytes:  8,
						FreeBytes:  8,
					},
				},
			},
		},
	}
	request := httptest.NewRequest(http.MethodGet, "/api/gpus", nil)
	response := httptest.NewRecorder()

	webui.NewHandler(webui.Config{
		SnapshotProvider: provider,
		Now:              func() time.Time { return collectedAt },
	}).ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, response.Code)
	}
	if provider.includeProcesses {
		t.Fatal("expected API to skip process collection")
	}
	if contentType := response.Header().Get("Content-Type"); contentType != "application/json; charset=utf-8" {
		t.Fatalf("expected JSON content type, got %q", contentType)
	}

	var payload struct {
		CollectedAt time.Time        `json:"collected_at"`
		Snapshot    gpuinfo.Snapshot `json:"snapshot"`
	}
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !payload.CollectedAt.Equal(collectedAt) {
		t.Fatalf("expected collected_at %s, got %s", collectedAt, payload.CollectedAt)
	}
	if got := payload.Snapshot.Devices[0].Name; got == nil || *got != name {
		t.Fatalf("expected device name %q, got %#v", name, got)
	}
}

func TestGPUAPIReturnsProviderError(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/api/gpus", nil)
	response := httptest.NewRecorder()

	webui.NewHandler(webui.Config{
		SnapshotProvider: &fakeProvider{err: errors.New("nvml failed")},
	}).ServeHTTP(response, request)

	if response.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, response.Code)
	}
	if body := response.Body.String(); !strings.Contains(body, "nvml failed") {
		t.Fatalf("expected error response, got %q", body)
	}
}

func TestGPUAPIReturnsUnavailableWithoutProvider(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/api/gpus", nil)
	response := httptest.NewRecorder()

	webui.NewHandler(webui.Config{}).ServeHTTP(response, request)

	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status %d, got %d", http.StatusServiceUnavailable, response.Code)
	}
}

func TestHealthAPIReturnsOKWhenGPUIsAvailable(t *testing.T) {
	index := 0
	provider := &fakeProvider{
		snapshot: gpuinfo.Snapshot{
			Devices: []gpuinfo.Device{{Index: &index}},
		},
	}
	request := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	response := httptest.NewRecorder()

	webui.NewHandler(webui.Config{SnapshotProvider: provider}).ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, response.Code)
	}
	if provider.includeProcesses {
		t.Fatal("expected health check to skip process collection")
	}
	if contentType := response.Header().Get("Content-Type"); contentType != "application/json; charset=utf-8" {
		t.Fatalf("expected JSON content type, got %q", contentType)
	}

	var payload struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.Status != "ok" {
		t.Fatalf("expected status %q, got %q", "ok", payload.Status)
	}
}

func TestHealthAPIReturnsUnavailableForProviderError(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	response := httptest.NewRecorder()

	webui.NewHandler(webui.Config{
		SnapshotProvider: &fakeProvider{err: errors.New("nvml failed")},
	}).ServeHTTP(response, request)

	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status %d, got %d", http.StatusServiceUnavailable, response.Code)
	}
	if body := response.Body.String(); !strings.Contains(body, "nvml failed") {
		t.Fatalf("expected error response, got %q", body)
	}
}

func TestHealthAPIReturnsUnavailableWithoutProvider(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	response := httptest.NewRecorder()

	webui.NewHandler(webui.Config{}).ServeHTTP(response, request)

	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status %d, got %d", http.StatusServiceUnavailable, response.Code)
	}
}

func TestHealthAPIReturnsUnavailableWithoutGPUs(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	response := httptest.NewRecorder()

	webui.NewHandler(webui.Config{
		SnapshotProvider: &fakeProvider{},
	}).ServeHTTP(response, request)

	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status %d, got %d", http.StatusServiceUnavailable, response.Code)
	}
	if body := response.Body.String(); !strings.Contains(body, "GPU is not available") {
		t.Fatalf("expected unavailable response, got %q", body)
	}
}

type fakeProvider struct {
	snapshot         gpuinfo.Snapshot
	err              error
	includeProcesses bool
}

func (provider *fakeProvider) List(_ context.Context, includeProcesses bool) (gpuinfo.Snapshot, error) {
	provider.includeProcesses = includeProcesses
	return provider.snapshot, provider.err
}
