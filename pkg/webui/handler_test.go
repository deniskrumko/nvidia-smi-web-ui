package webui_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
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
	for _, unexpected := range []string{"Live GPU monitoring", "History is stored only in this browser tab"} {
		if strings.Contains(body, unexpected) {
			t.Fatalf("expected index not to contain %q, got %q", unexpected, body)
		}
	}
	for _, expected := range []string{"data-gpu-trigger", "data-chart-trigger", "data-time-minutes", "data-refresh-interval"} {
		if !strings.Contains(body, expected) {
			t.Fatalf("expected index to contain %q, got %q", expected, body)
		}
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

func TestNewHandlerServesStaticAssets(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		contains string
	}{
		{name: "css", path: "/static/app.css", contains: ".topbar"},
		{name: "js", path: "/static/app.js", contains: "refreshInterval: 1000"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, test.path, nil)
			response := httptest.NewRecorder()

			webui.NewHandler(webui.Config{}).ServeHTTP(response, request)

			if response.Code != http.StatusOK {
				t.Fatalf("expected status %d, got %d", http.StatusOK, response.Code)
			}
			if body := response.Body.String(); !strings.Contains(body, test.contains) {
				t.Fatalf("expected static response to contain %q, got %q", test.contains, body)
			}
		})
	}
}

func TestStaticAssetsDoNotContainRemovedChartZoomControls(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/static/app.js", nil)
	response := httptest.NewRecorder()

	webui.NewHandler(webui.Config{}).ServeHTTP(response, request)

	body := response.Body.String()
	for _, unexpected := range []string{"Reset zoom", "handleWheel", "data-reset-zoom", "Refreshing..."} {
		if strings.Contains(body, unexpected) {
			t.Fatalf("expected app.js not to contain %q", unexpected)
		}
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

type fakeProvider struct {
	snapshot         gpuinfo.Snapshot
	err              error
	includeProcesses bool
}

func (provider *fakeProvider) List(_ context.Context, includeProcesses bool) (gpuinfo.Snapshot, error) {
	provider.includeProcesses = includeProcesses
	return provider.snapshot, provider.err
}
