package webui

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/deniskrumko/nvidia-smi-web-ui/pkg/gpuinfo"
	"github.com/deniskrumko/nvidia-smi-web-ui/pkg/utils"
)

const defaultPageTitle = "Nvidia SMI"
const defaultVersion = "local"

// SnapshotProvider provides point-in-time GPU snapshots for the web API.
type SnapshotProvider interface {
	List(context.Context, bool) (gpuinfo.Snapshot, error)
}

// RemoteHost describes a GPU API served by another nvidia-smi-web-ui instance.
type RemoteHost struct {
	Name    string
	URL     string
	Default bool
}

type httpDoer interface {
	Do(*http.Request) (*http.Response, error)
}

// Config contains web UI handler settings.
type Config struct {
	SnapshotProvider SnapshotProvider
	RemoteHosts      []RemoteHost
	HTTPClient       httpDoer
	DisableAccessLog bool
	AccessLogLevel   utils.LogLevel
	Branding         string
	Title            string
	Version          string
	StaticDir        string // optional directory served at /static and used as the favicon root
	TemplatesDir     string // optional directory containing web UI HTML templates
	Now              func() time.Time
}

// NewHandler creates an HTTP handler for the web UI and stateless JSON API.
func NewHandler(config Config) http.Handler {
	staticDir := utils.TextOrDefault(config.StaticDir, utils.DefaultAssetDir("static", assetDirCandidates("static")...))
	templatesDir := utils.TextOrDefault(config.TemplatesDir, utils.DefaultAssetDir("templates", assetDirCandidates("templates")...))
	renderer := &renderer{
		templates: template.Must(template.ParseGlob(filepath.Join(templatesDir, "*.html"))),
		branding:  utils.TextOrDefault(config.Branding, defaultPageTitle),
		title:     utils.TextOrDefault(config.Title, utils.TextOrDefault(config.Branding, defaultPageTitle)),
		version:   utils.TextOrDefault(config.Version, utils.ReadVersionFile(".version", defaultVersion)),
		provider:  config.SnapshotProvider,
		hosts:     config.RemoteHosts,
		client:    config.HTTPClient,
		now:       config.Now,
	}
	if renderer.now == nil {
		renderer.now = time.Now
	}
	if renderer.client == nil {
		renderer.client = &http.Client{Timeout: 10 * time.Second}
	}

	mux := http.NewServeMux()
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.Dir(staticDir))))
	mux.Handle("GET /favicon/", http.StripPrefix("/favicon/", http.FileServer(http.Dir(filepath.Join(staticDir, "favicon")))))
	mux.HandleFunc("GET /api/health", renderer.health)
	mux.HandleFunc("GET /api/gpus", renderer.gpus)
	mux.HandleFunc("GET /", renderer.index)
	if config.DisableAccessLog {
		return mux
	}
	return utils.AccessLog(mux, renderer.now, config.AccessLogLevel)
}

type renderer struct {
	templates *template.Template
	branding  string
	title     string
	version   string
	provider  SnapshotProvider
	hosts     []RemoteHost
	client    httpDoer
	now       func() time.Time
}

type pageData struct {
	Title     string
	Branding  string
	Version   string
	HostsJSON string
	HasHosts  bool
}

type gpuResponse struct {
	CollectedAt time.Time        `json:"collected_at"`
	Snapshot    gpuinfo.Snapshot `json:"snapshot"`
}

type healthResponse struct {
	Status string `json:"status"`
}

type hostOption struct {
	Index   int    `json:"index"`
	Name    string `json:"name"`
	Default bool   `json:"default"`
}

func (renderer *renderer) index(response http.ResponseWriter, request *http.Request) {
	if request.URL.Path != "/" {
		utils.LogHTTPError(request, http.StatusNotFound, "page not found")
		http.NotFound(response, request)
		return
	}

	data := pageData{
		Title:     renderer.title,
		Branding:  renderer.branding,
		Version:   renderer.version,
		HostsJSON: renderer.hostsJSON(),
		HasHosts:  len(renderer.hosts) > 0,
	}

	response.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := renderer.templates.ExecuteTemplate(response, "layout.html", data); err != nil {
		utils.LogHTTPError(request, http.StatusInternalServerError, err.Error())
		http.Error(response, err.Error(), http.StatusInternalServerError)
	}
}

func (renderer *renderer) health(response http.ResponseWriter, request *http.Request) {
	utils.WriteJSON(response, http.StatusOK, healthResponse{Status: "ok"})
}

func (renderer *renderer) gpus(response http.ResponseWriter, request *http.Request) {
	if len(renderer.hosts) > 0 {
		renderer.remoteHostGPUs(response, request)
		return
	}

	renderer.localGPUs(response, request)
}

func (renderer *renderer) localGPUs(response http.ResponseWriter, request *http.Request) {
	if renderer.provider == nil {
		utils.WriteJSONError(response, request, http.StatusServiceUnavailable, "GPU provider is not configured")
		return
	}

	snapshot, err := renderer.provider.List(request.Context(), false)
	if err != nil {
		utils.WriteJSONError(response, request, http.StatusInternalServerError, err.Error())
		return
	}

	utils.WriteJSON(response, http.StatusOK, gpuResponse{
		CollectedAt: renderer.now().UTC(),
		Snapshot:    snapshot,
	})
}

func (renderer *renderer) remoteHostGPUs(response http.ResponseWriter, request *http.Request) {
	hostIndex, err := renderer.requestedHostIndex(request)
	if err != nil {
		utils.WriteJSONError(response, request, http.StatusBadRequest, err.Error())
		return
	}

	payload, status, err := renderer.remoteGPUs(request.Context(), hostIndex)
	if err != nil {
		utils.WriteJSONError(response, request, status, err.Error())
		return
	}

	utils.WriteJSON(response, http.StatusOK, payload)
}

func (renderer *renderer) requestedHostIndex(request *http.Request) (int, error) {
	value := strings.TrimSpace(request.URL.Query().Get("host"))
	if value == "" {
		return 0, fmt.Errorf("multi-host mode is enabled, specify a concrete host with the host query parameter")
	}

	index, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("host must be a number")
	}
	if index < 0 || index >= len(renderer.hosts) {
		return 0, fmt.Errorf("host %d is not configured", index)
	}

	return index, nil
}

func (renderer *renderer) remoteGPUs(ctx context.Context, index int) (gpuResponse, int, error) {
	host := renderer.hosts[index]
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, host.URL, nil)
	if err != nil {
		return gpuResponse{}, http.StatusInternalServerError, err
	}
	request.Header.Set("Accept", "application/json")

	response, err := renderer.client.Do(request)
	if err != nil {
		return gpuResponse{}, http.StatusBadGateway, fmt.Errorf("fetch %s: %w", host.Name, err)
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return gpuResponse{}, http.StatusBadGateway, fmt.Errorf("read %s response: %w", host.Name, err)
	}

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		message := strings.TrimSpace(string(body))
		if message == "" {
			message = response.Status
		}
		return gpuResponse{}, response.StatusCode, fmt.Errorf("%s: %s", host.Name, message)
	}

	var payload gpuResponse
	if err := json.Unmarshal(body, &payload); err != nil {
		return gpuResponse{}, http.StatusBadGateway, fmt.Errorf("decode %s response: %w", host.Name, err)
	}

	return payload, http.StatusOK, nil
}

func (renderer *renderer) defaultHostIndex() int {
	for index, host := range renderer.hosts {
		if host.Default {
			return index
		}
	}
	return 0
}

func (renderer *renderer) hostsJSON() string {
	options := make([]hostOption, 0, len(renderer.hosts))
	for index, host := range renderer.hosts {
		options = append(options, hostOption{
			Index:   index,
			Name:    host.Name,
			Default: host.Default,
		})
	}

	content, err := json.Marshal(options)
	if err != nil {
		return "[]"
	}
	return string(content)
}

func assetDirCandidates(name string) []string {
	return []string{
		name,
		filepath.Join("pkg", "webui", name),
		filepath.Join(utils.PackageDir(), name),
	}
}
