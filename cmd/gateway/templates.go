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

var homeTemplate *template.Template
var directoryTemplate *template.Template

func init() {
	var err error
	homeTemplate, err = template.New("home").ParseFS(templateFS, "templates/base.html", "templates/home.html")
	if err != nil {
		panic("Failed to parse home templates: " + err.Error())
	}

	directoryTemplate, err = template.New("directory").ParseFS(templateFS, "templates/base.html", "templates/directory.html")
	if err != nil {
		panic("Failed to parse directory templates: " + err.Error())
	}
}

type BaseTemplateData struct {
	Title string
}

type HomeTemplateData struct {
	BaseTemplateData
}

type DirectoryTemplateData struct {
	BaseTemplateData
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

func RenderHome(w http.ResponseWriter) error {
	data := HomeTemplateData{
		BaseTemplateData: BaseTemplateData{
			Title: "Blobcast Explorer",
		},
	}
	return homeTemplate.ExecuteTemplate(w, "home.html", data)
}

func RenderDirectory(w http.ResponseWriter, data DirectoryTemplateData) error {
	return directoryTemplate.ExecuteTemplate(w, "directory.html", data)
}
