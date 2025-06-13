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

// Template function map
var funcMap = template.FuncMap{
	"add": func(a, b int) int {
		return a + b
	},
	"sub": func(a, b int) int {
		return a - b
	},
	"mul": func(a, b int) int {
		return a * b
	},
	"div": func(a, b int) int {
		if b != 0 {
			return a / b
		}
		return 0
	},
}

func init() {
	var err error
	homeTemplate, err = template.New("home").Funcs(funcMap).ParseFS(templateFS, "templates/base.html", "templates/home.html")
	if err != nil {
		panic("Failed to parse home templates: " + err.Error())
	}

	directoryTemplate, err = template.New("directory").
		Funcs(funcMap).
		ParseFS(templateFS, "templates/base.html", "templates/directory.html", "templates/pagination.html")
	if err != nil {
		panic("Failed to parse directory templates: " + err.Error())
	}
}

type BaseTemplateData struct {
	Title string
}

type Pagination struct {
	BaseURL     string
	CurrentPage int
	TotalPages  int
	HasPrev     bool
	HasNext     bool
	PrevPage    int
	NextPage    int
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
	TotalFiles  int
	Pagination  *Pagination
}

type DirectoryItem struct {
	Name string
	Path string
}

type FileItem struct {
	Name         string
	Path         string
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
			Title: "Blobcast Gateway",
		},
	}
	return homeTemplate.ExecuteTemplate(w, "home.html", data)
}

func RenderDirectory(w http.ResponseWriter, data DirectoryTemplateData) error {
	return directoryTemplate.ExecuteTemplate(w, "directory.html", data)
}
