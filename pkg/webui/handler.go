package webui

import (
	"embed"
	"html/template"
	"net/http"
)

//go:embed static/*.css templates/*.html
var assetFS embed.FS

// NewHandler creates an HTTP handler for the server-rendered web UI.
func NewHandler() http.Handler {
	renderer := &renderer{
		templates: template.Must(template.ParseFS(assetFS, "templates/*.html")),
	}

	mux := http.NewServeMux()
	mux.Handle("GET /static/", http.FileServer(http.FS(assetFS)))
	mux.HandleFunc("GET /", renderer.index)
	return mux
}

type renderer struct {
	templates *template.Template
}

type pageData struct {
	Title   string
	Heading string
	Message string
}

func (renderer *renderer) index(response http.ResponseWriter, request *http.Request) {
	if request.URL.Path != "/" {
		http.NotFound(response, request)
		return
	}

	data := pageData{
		Title:   "nvidia-smi-web-ui",
		Heading: "Hello World",
		Message: "The web UI skeleton is running.",
	}

	response.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := renderer.templates.ExecuteTemplate(response, "layout.html", data); err != nil {
		http.Error(response, err.Error(), http.StatusInternalServerError)
	}
}
