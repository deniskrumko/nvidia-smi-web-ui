package webui

import (
	"context"
	"embed"
	"encoding/json"
	"html/template"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/deniskrumko/nvidia-smi-web-ui/pkg/gpuinfo"
)

//go:embed static/*.css static/*.js templates/*.html
var assetFS embed.FS

const defaultPageTitle = "Nvidia SMI Web UI"
const defaultVersion = "local"

// SnapshotProvider provides point-in-time GPU snapshots for the web API.
type SnapshotProvider interface {
	List(context.Context, bool) (gpuinfo.Snapshot, error)
}

// Config contains web UI handler settings.
type Config struct {
	SnapshotProvider SnapshotProvider
	Branding         string
	Title            string
	Version          string
	Now              func() time.Time
}

// NewHandler creates an HTTP handler for the web UI and stateless JSON API.
func NewHandler(config Config) http.Handler {
	renderer := &renderer{
		templates: template.Must(template.ParseFS(assetFS, "templates/*.html")),
		branding:  textOrDefault(config.Branding, defaultPageTitle),
		title:     textOrDefault(config.Title, textOrDefault(config.Branding, defaultPageTitle)),
		version:   textOrDefault(config.Version, readVersionFile(".version")),
		provider:  config.SnapshotProvider,
		now:       config.Now,
	}
	if renderer.now == nil {
		renderer.now = time.Now
	}

	mux := http.NewServeMux()
	mux.Handle("GET /static/", http.FileServer(http.FS(assetFS)))
	mux.HandleFunc("GET /api/gpus", renderer.gpus)
	mux.HandleFunc("GET /", renderer.index)
	return mux
}

type renderer struct {
	templates *template.Template
	branding  string
	title     string
	version   string
	provider  SnapshotProvider
	now       func() time.Time
}

type pageData struct {
	Title    string
	Branding string
	Version  string
}

type gpuResponse struct {
	CollectedAt time.Time        `json:"collected_at"`
	Snapshot    gpuinfo.Snapshot `json:"snapshot"`
}

type errorResponse struct {
	Error string `json:"error"`
}

func (renderer *renderer) index(response http.ResponseWriter, request *http.Request) {
	if request.URL.Path != "/" {
		http.NotFound(response, request)
		return
	}

	data := pageData{
		Title:    renderer.title,
		Branding: renderer.branding,
		Version:  renderer.version,
	}

	response.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := renderer.templates.ExecuteTemplate(response, "layout.html", data); err != nil {
		http.Error(response, err.Error(), http.StatusInternalServerError)
	}
}

func (renderer *renderer) gpus(response http.ResponseWriter, request *http.Request) {
	if renderer.provider == nil {
		writeJSONError(response, http.StatusServiceUnavailable, "GPU provider is not configured")
		return
	}

	snapshot, err := renderer.provider.List(request.Context(), false)
	if err != nil {
		writeJSONError(response, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(response, http.StatusOK, gpuResponse{
		CollectedAt: renderer.now().UTC(),
		Snapshot:    snapshot,
	})
}

func writeJSON(response http.ResponseWriter, status int, value any) {
	response.Header().Set("Content-Type", "application/json; charset=utf-8")
	response.WriteHeader(status)
	_ = json.NewEncoder(response).Encode(value)
}

func writeJSONError(response http.ResponseWriter, status int, message string) {
	writeJSON(response, status, errorResponse{Error: message})
}

func textOrDefault(value string, fallback string) string {
	if value != "" {
		return value
	}
	return fallback
}

func readVersionFile(path string) string {
	content, err := os.ReadFile(path)
	if err != nil {
		return defaultVersion
	}
	return textOrDefault(strings.TrimSpace(string(content)), defaultVersion)
}
