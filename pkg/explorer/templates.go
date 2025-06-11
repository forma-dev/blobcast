package explorer

import (
	"embed"
	"html/template"
	"net/http"
)

//go:embed templates
var templateFS embed.FS

var (
	homeTemplate             *template.Template
	blocksTemplate           *template.Template
	blockDetailsTemplate     *template.Template
	filesTemplate            *template.Template
	fileDetailsTemplate      *template.Template
	directoriesTemplate      *template.Template
	directoryDetailsTemplate *template.Template
	searchTemplate           *template.Template
)

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

	blocksTemplate, err = template.New("blocks").Funcs(funcMap).ParseFS(templateFS, "templates/base.html", "templates/blocks.html", "templates/pagination.html")
	if err != nil {
		panic("Failed to parse blocks templates: " + err.Error())
	}

	blockDetailsTemplate, err = template.New("block_details").Funcs(funcMap).ParseFS(templateFS, "templates/base.html", "templates/block_details.html")
	if err != nil {
		panic("Failed to parse block details templates: " + err.Error())
	}

	filesTemplate, err = template.New("files").Funcs(funcMap).ParseFS(templateFS, "templates/base.html", "templates/files.html", "templates/pagination.html")
	if err != nil {
		panic("Failed to parse files templates: " + err.Error())
	}

	fileDetailsTemplate, err = template.New("file_details").Funcs(funcMap).ParseFS(templateFS, "templates/base.html", "templates/file_details.html")
	if err != nil {
		panic("Failed to parse file details templates: " + err.Error())
	}

	directoriesTemplate, err = template.New("directories").
		Funcs(funcMap).
		ParseFS(templateFS, "templates/base.html", "templates/directories.html", "templates/pagination.html")
	if err != nil {
		panic("Failed to parse directories templates: " + err.Error())
	}

	directoryDetailsTemplate, err = template.New("directory_details").
		Funcs(funcMap).
		ParseFS(templateFS, "templates/base.html", "templates/directory_details.html")
	if err != nil {
		panic("Failed to parse directory details templates: " + err.Error())
	}

	searchTemplate, err = template.New("search").Funcs(funcMap).ParseFS(templateFS, "templates/base.html", "templates/search.html")
	if err != nil {
		panic("Failed to parse search templates: " + err.Error())
	}
}

// StorageAnalytics represents analytics data for templates
type StorageAnalytics struct {
	TotalBlocks          int
	TotalChunks          int
	TotalFiles           int
	TotalDirectories     int
	TotalStorage         interface{} // Can be uint64 or string
	AvgBlockSize         interface{} // Can be uint64 or string
	MostCommonMimeTypes  map[string]int
	FileTypeDistribution map[string]interface{} // Can be uint64 or string
}

type BaseTemplateData struct {
	Title      string
	GatewayUrl string
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
	Analytics    *StorageAnalytics
	RecentBlocks []BlockData
}

type BlocksTemplateData struct {
	BaseTemplateData
	Blocks      []BlockData
	TotalBlocks int
	Pagination  *Pagination
}

type BlockDetailsTemplateData struct {
	BaseTemplateData
	Block       *BlockData
	Files       []FileData
	Directories []DirectoryData
	Chunks      []ChunkData
	HasFiles    bool
	HasError    bool
	ErrorMsg    string
}

type FilesTemplateData struct {
	BaseTemplateData
	Files      []FileData
	TotalFiles int
	Pagination *Pagination
}

type FileDetailsTemplateData struct {
	BaseTemplateData
	File   *FileData
	Chunks []ChunkData
}

type DirectoriesTemplateData struct {
	BaseTemplateData
	Directories      []DirectoryData
	TotalDirectories int
	Pagination       *Pagination
}

type DirectoryDetailsTemplateData struct {
	BaseTemplateData
	Directory *DirectoryData
}

type SearchTemplateData struct {
	BaseTemplateData
	Query       string
	Files       []FileData
	HasSearched bool
}

// Data structures for template rendering
type BlockData struct {
	Height           uint64
	Hash             string
	Timestamp        string
	CelestiaHeight   uint64
	TotalFiles       int
	TotalDirectories int
	TotalChunks      int
	StorageUsed      string
	ParentHash       string
	DirsRoot         string
	FilesRoot        string
	ChunksRoot       string
	StateRoot        string
}

type FileData struct {
	BlobID      string
	FileName    string
	MimeType    string
	FileSize    string
	FileHash    string
	BlockHeight uint64
	Timestamp   string
	ChunkCount  int
}

type DirectoryData struct {
	BlobID        string
	DirectoryName string
	DirectoryHash string
	FileCount     int
	TotalSize     string
	BlockHeight   uint64
	Timestamp     string
	FileTypes     []string
}

type ChunkData struct {
	BlobID      string
	Index       int
	BlockHeight uint64
	ChunkSize   string
	ChunkHash   string
}

// Template rendering functions
func RenderHome(w http.ResponseWriter, data HomeTemplateData) error {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	return homeTemplate.ExecuteTemplate(w, "base.html", data)
}

func RenderBlocks(w http.ResponseWriter, data BlocksTemplateData) error {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	return blocksTemplate.ExecuteTemplate(w, "base.html", data)
}

func RenderBlockDetails(w http.ResponseWriter, data BlockDetailsTemplateData) error {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	return blockDetailsTemplate.ExecuteTemplate(w, "base.html", data)
}

func RenderFiles(w http.ResponseWriter, data FilesTemplateData) error {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	return filesTemplate.ExecuteTemplate(w, "base.html", data)
}

func RenderFileDetails(w http.ResponseWriter, data FileDetailsTemplateData) error {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	return fileDetailsTemplate.ExecuteTemplate(w, "base.html", data)
}

func RenderDirectories(w http.ResponseWriter, data DirectoriesTemplateData) error {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	return directoriesTemplate.ExecuteTemplate(w, "base.html", data)
}

func RenderDirectoryDetails(w http.ResponseWriter, data DirectoryDetailsTemplateData) error {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	return directoryDetailsTemplate.ExecuteTemplate(w, "base.html", data)
}

func RenderSearch(w http.ResponseWriter, data SearchTemplateData) error {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	return searchTemplate.ExecuteTemplate(w, "base.html", data)
}
