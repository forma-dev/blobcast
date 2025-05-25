package gateway

import (
	"embed"
	"html/template"
	"net/http"
	"strings"
)

//go:embed templates
var templateFS embed.FS

//go:embed static
var staticFS embed.FS

var templates *template.Template

func init() {
	var err error
	templates, err = template.New("").ParseFS(templateFS, "templates/*.html")
	if err != nil {
		panic("Failed to parse templates: " + err.Error())
	}
}

type TemplateData struct {
	Title       string
	ManifestID  string
	DisplayID   string
	SubPath     string
	ParentPath  string
	Directories []DirectoryItem
	Files       []FileItem
	HasParent   bool
}

type DirectoryItem struct {
	Name string
	Path string
}

type FileItem struct {
	Name         string
	Path         string
	Icon         string
	Size         string
	MimeType     string
	ManifestID   string
	ManifestLink string
}

func ServeStatic(w http.ResponseWriter, r *http.Request) {
	// Serve CSS and other static files
	file := r.URL.Path[1:] // Remove leading /
	content, err := staticFS.ReadFile(file)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	if strings.HasSuffix(file, ".css") {
		w.Header().Set("Content-Type", "text/css")
	}

	if strings.HasSuffix(file, ".png") {
		w.Header().Set("Content-Type", "image/png")
	}

	w.Write(content)
}

func RenderDirectory(w http.ResponseWriter, data TemplateData) error {
	return templates.ExecuteTemplate(w, "base.html", data)
}
