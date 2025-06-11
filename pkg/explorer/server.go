package explorer

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/forma-dev/blobcast/pkg/state"
	"github.com/gorilla/mux"
)

type Server struct {
	indexDB    *state.IndexerDatabase
	gatewayUrl string
	router     *mux.Router
}

func NewServer(indexDB *state.IndexerDatabase, gatewayUrl string) *Server {
	s := &Server{
		indexDB:    indexDB,
		gatewayUrl: gatewayUrl,
		router:     mux.NewRouter(),
	}
	s.setupRoutes()
	return s
}

func (s *Server) Router() *mux.Router {
	return s.router
}

func (s *Server) setupRoutes() {
	// Web interface routes
	s.router.HandleFunc("/", s.handleHome).Methods("GET")
	s.router.HandleFunc("/blocks", s.handleBlocks).Methods("GET")
	s.router.HandleFunc("/blocks/{height}", s.handleBlockDetails).Methods("GET")
	s.router.HandleFunc("/files", s.handleFiles).Methods("GET")
	s.router.HandleFunc("/files/{blobId}", s.handleFileDetails).Methods("GET")
	s.router.HandleFunc("/directories", s.handleDirectories).Methods("GET")
	s.router.HandleFunc("/directories/{blobId}", s.handleDirectoryDetails).Methods("GET")
	s.router.HandleFunc("/search", s.handleSearch).Methods("GET")

	// API routes
	api := s.router.PathPrefix("/api/v1").Subrouter()
	api.HandleFunc("/blocks", s.apiGetBlocks).Methods("GET")
	api.HandleFunc("/analytics", s.apiGetAnalytics).Methods("GET")
}

func (s *Server) handleHome(w http.ResponseWriter, r *http.Request) {
	// Get analytics for the home page
	analytics, err := s.indexDB.GetStorageAnalytics()
	if err != nil {
		analytics = &state.StorageAnalytics{} // Empty analytics if error
	}

	// Get recent blocks
	recentBlocks, err := s.indexDB.GetBlocks(50, 0)
	if err != nil {
		recentBlocks = []*state.BlockIndex{} // Empty if error
	}

	// Convert blocks to template data
	blockData := make([]BlockData, len(recentBlocks))
	for i, block := range recentBlocks {
		blockData[i] = BlockData{
			Height:           block.Height,
			Hash:             truncateString(block.Hash, 20),
			Timestamp:        block.Timestamp.Format("2006-01-02 15:04:05"),
			CelestiaHeight:   block.CelestiaHeight,
			TotalFiles:       block.TotalFiles,
			TotalDirectories: block.TotalDirectories,
			TotalChunks:      block.TotalChunks,
			StorageUsed:      formatBytes(block.StorageUsed),
		}
	}

	// Convert analytics to template data
	templateAnalytics := &StorageAnalytics{
		TotalBlocks:          int(analytics.TotalBlocks),
		TotalChunks:          int(analytics.TotalChunks),
		TotalFiles:           int(analytics.TotalFiles),
		TotalDirectories:     int(analytics.TotalDirectories),
		TotalStorage:         formatBytes(analytics.TotalStorage),
		AvgBlockSize:         formatBytes(analytics.AvgBlockSize),
		MostCommonMimeTypes:  analytics.MostCommonMimeTypes,
		FileTypeDistribution: make(map[string]interface{}),
	}

	// Convert file type distribution
	for k, v := range analytics.FileTypeDistribution {
		templateAnalytics.FileTypeDistribution[k] = formatBytes(v)
	}

	data := HomeTemplateData{
		BaseTemplateData: BaseTemplateData{
			Title:      "Home",
			GatewayUrl: s.gatewayUrl,
		},
		Analytics:    templateAnalytics,
		RecentBlocks: blockData,
	}

	if err := RenderHome(w, data); err != nil {
		http.Error(w, "Template error: "+err.Error(), http.StatusInternalServerError)
	}
}

func (s *Server) handleBlocks(w http.ResponseWriter, r *http.Request) {
	// Parse pagination parameters
	page := 1
	if p := r.URL.Query().Get("page"); p != "" {
		if parsed, err := strconv.Atoi(p); err == nil && parsed > 0 {
			page = parsed
		}
	}

	limit := 20
	offset := (page - 1) * limit

	// Get blocks from database
	blocks, err := s.indexDB.GetBlocks(limit, offset)
	if err != nil {
		http.Error(w, "Error fetching blocks: "+err.Error(), http.StatusInternalServerError)
		return
	}

	totalBlocks, err := s.indexDB.GetTotalBlocks()
	if err != nil {
		totalBlocks = 0
	}

	totalPages := (totalBlocks + limit - 1) / limit

	// Convert blocks to template data
	blockData := make([]BlockData, len(blocks))
	for i, block := range blocks {
		blockData[i] = BlockData{
			Height:           block.Height,
			Hash:             truncateString(block.Hash, 20),
			Timestamp:        block.Timestamp.Format("2006-01-02 15:04:05"),
			CelestiaHeight:   block.CelestiaHeight,
			TotalFiles:       block.TotalFiles,
			TotalDirectories: block.TotalDirectories,
			TotalChunks:      block.TotalChunks,
			StorageUsed:      formatBytes(block.StorageUsed),
		}
	}

	// Create pagination data
	pagination := &Pagination{
		BaseURL:     "/blocks",
		CurrentPage: page,
		TotalPages:  totalPages,
		HasPrev:     page > 1,
		HasNext:     page < totalPages,
		PrevPage:    page - 1,
		NextPage:    page + 1,
	}

	data := BlocksTemplateData{
		BaseTemplateData: BaseTemplateData{
			Title:      "Blocks",
			GatewayUrl: s.gatewayUrl,
		},
		Blocks:      blockData,
		TotalBlocks: totalBlocks,
		Pagination:  pagination,
	}

	if err := RenderBlocks(w, data); err != nil {
		http.Error(w, "Template error: "+err.Error(), http.StatusInternalServerError)
	}
}

func (s *Server) handleBlockDetails(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	heightStr := vars["height"]

	height, err := strconv.ParseUint(heightStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid block height", http.StatusBadRequest)
		return
	}

	// Get block from indexer
	block, err := s.indexDB.GetBlockIndex(height)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Block not found", http.StatusNotFound)
		} else {
			http.Error(w, "Error fetching block: "+err.Error(), http.StatusInternalServerError)
		}
		return
	}

	// Get files in this block
	files, err := s.indexDB.GetFilesByBlockHeight(height)
	if err != nil {
		files = []*state.FileIndex{} // Empty if error
	}

	// Get directories in this block
	directories, err := s.indexDB.GetDirectoriesByBlockHeight(height)
	if err != nil {
		directories = []*state.DirectoryIndex{} // Empty if error
	}

	// Get chunks in this block from indexer (no longer calling the node!)
	chunks, err := s.indexDB.GetChunksByBlockHeight(height)
	if err != nil {
		chunks = []*state.ChunkIndex{} // Empty if error
	}

	// Convert chunks to structured data from indexer
	chunksData := make([]ChunkData, len(chunks))
	for i, chunk := range chunks {
		chunksData[i] = ChunkData{
			BlobID:      chunk.BlobID,
			Index:       chunk.Index,
			BlockHeight: chunk.BlockHeight,
			ChunkSize:   formatBytes(chunk.ChunkSize),
			ChunkHash:   chunk.ChunkHash,
		}
	}

	// Convert to template data with all block information unified
	blockData := &BlockData{
		Height:           block.Height,
		Hash:             block.Hash,
		Timestamp:        block.Timestamp.Format("2006-01-02 15:04:05 UTC"),
		CelestiaHeight:   block.CelestiaHeight,
		TotalFiles:       block.TotalFiles,
		TotalDirectories: block.TotalDirectories,
		TotalChunks:      block.TotalChunks,
		StorageUsed:      formatBytes(block.StorageUsed),
		ParentHash:       block.ParentHash,
		DirsRoot:         block.DirsRoot,
		FilesRoot:        block.FilesRoot,
		ChunksRoot:       block.ChunksRoot,
		StateRoot:        block.StateRoot,
	}

	// Convert files
	fileData := make([]FileData, len(files))
	for i, file := range files {
		fileData[i] = FileData{
			BlobID:      file.BlobID,
			FileName:    file.FileName,
			MimeType:    file.MimeType,
			FileSize:    formatBytes(file.FileSize),
			FileHash:    file.FileHash,
			BlockHeight: file.BlockHeight,
			ChunkCount:  file.ChunkCount,
		}
	}

	// Convert directories
	directoryData := make([]DirectoryData, len(directories))
	for i, dir := range directories {
		directoryData[i] = DirectoryData{
			BlobID:        dir.BlobID,
			DirectoryName: dir.DirectoryName,
			DirectoryHash: dir.DirectoryHash,
			FileCount:     dir.FileCount,
			TotalSize:     formatBytes(dir.TotalSize),
			BlockHeight:   dir.BlockHeight,
			FileTypes:     dir.FileTypes,
		}
	}

	data := BlockDetailsTemplateData{
		BaseTemplateData: BaseTemplateData{
			Title:      fmt.Sprintf("Block %d", height),
			GatewayUrl: s.gatewayUrl,
		},
		Block:       blockData,
		Files:       fileData,
		Directories: directoryData,
		Chunks:      chunksData,
		HasFiles:    len(fileData) > 0,
		HasError:    false, // No longer have errors since we're not calling the node
		ErrorMsg:    "",
	}

	if err := RenderBlockDetails(w, data); err != nil {
		http.Error(w, "Template error: "+err.Error(), http.StatusInternalServerError)
	}
}

func (s *Server) handleFiles(w http.ResponseWriter, r *http.Request) {
	// Parse pagination parameters
	page := 1
	if p := r.URL.Query().Get("page"); p != "" {
		if parsed, err := strconv.Atoi(p); err == nil && parsed > 0 {
			page = parsed
		}
	}

	limit := 20
	offset := (page - 1) * limit

	// Get files from database
	files, err := s.indexDB.GetFiles(limit, offset)
	if err != nil {
		http.Error(w, "Error fetching files: "+err.Error(), http.StatusInternalServerError)
		return
	}

	totalFiles, err := s.indexDB.GetTotalFiles()
	if err != nil {
		totalFiles = 0
	}

	totalPages := (totalFiles + limit - 1) / limit

	// Convert files to template data
	fileData := make([]FileData, len(files))
	for i, file := range files {
		fileData[i] = FileData{
			BlobID:      file.BlobID,
			FileName:    file.FileName,
			MimeType:    file.MimeType,
			FileSize:    formatBytes(file.FileSize),
			FileHash:    file.FileHash,
			BlockHeight: file.BlockHeight,
			ChunkCount:  file.ChunkCount,
		}
	}

	// Create pagination data
	pagination := &Pagination{
		BaseURL:     "/files",
		CurrentPage: page,
		TotalPages:  totalPages,
		HasPrev:     page > 1,
		HasNext:     page < totalPages,
		PrevPage:    page - 1,
		NextPage:    page + 1,
	}

	data := FilesTemplateData{
		BaseTemplateData: BaseTemplateData{
			Title:      "Files",
			GatewayUrl: s.gatewayUrl,
		},
		Files:      fileData,
		TotalFiles: totalFiles,
		Pagination: pagination,
	}

	if err := RenderFiles(w, data); err != nil {
		http.Error(w, "Template error: "+err.Error(), http.StatusInternalServerError)
	}
}

func (s *Server) handleFileDetails(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	blobId := vars["blobId"]

	// Get file from indexer
	file, err := s.indexDB.GetFileIndex(blobId)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "File not found", http.StatusNotFound)
		} else {
			http.Error(w, "Error fetching file: "+err.Error(), http.StatusInternalServerError)
		}
		return
	}

	fileData := &FileData{
		BlobID:      file.BlobID,
		FileName:    file.FileName,
		MimeType:    file.MimeType,
		FileSize:    formatBytes(file.FileSize),
		FileHash:    file.FileHash,
		BlockHeight: file.BlockHeight,
		ChunkCount:  file.ChunkCount,
	}

	chunks, err := s.indexDB.GetFileChunks(file.BlobID)
	if err != nil {
		chunks = []*state.ChunkIndex{}
	}

	chunksData := make([]ChunkData, len(chunks))
	for i, chunk := range chunks {
		chunksData[i] = ChunkData{
			BlobID:      chunk.BlobID,
			Index:       chunk.Index,
			BlockHeight: chunk.BlockHeight,
			ChunkSize:   formatBytes(chunk.ChunkSize),
			ChunkHash:   chunk.ChunkHash,
		}
	}

	data := FileDetailsTemplateData{
		BaseTemplateData: BaseTemplateData{
			Title:      truncateString(file.FileName, 30),
			GatewayUrl: s.gatewayUrl,
		},
		File:   fileData,
		Chunks: chunksData,
	}

	if err := RenderFileDetails(w, data); err != nil {
		http.Error(w, "Template error: "+err.Error(), http.StatusInternalServerError)
	}
}

func (s *Server) handleDirectories(w http.ResponseWriter, r *http.Request) {
	// Parse pagination parameters
	page := 1
	if p := r.URL.Query().Get("page"); p != "" {
		if parsed, err := strconv.Atoi(p); err == nil && parsed > 0 {
			page = parsed
		}
	}

	limit := 20
	offset := (page - 1) * limit

	// Get directories from database
	directories, err := s.indexDB.GetDirectories(limit, offset)
	if err != nil {
		http.Error(w, "Error fetching directories: "+err.Error(), http.StatusInternalServerError)
		return
	}

	totalDirectories, err := s.indexDB.GetTotalDirectories()
	if err != nil {
		totalDirectories = 0
	}

	totalPages := (totalDirectories + limit - 1) / limit

	// Convert directories to template data
	directoryData := make([]DirectoryData, len(directories))
	for i, dir := range directories {
		directoryData[i] = DirectoryData{
			BlobID:        dir.BlobID,
			DirectoryName: truncateString(dir.DirectoryName, 30),
			DirectoryHash: dir.DirectoryHash,
			FileCount:     dir.FileCount,
			TotalSize:     formatBytes(dir.TotalSize),
			BlockHeight:   dir.BlockHeight,
			FileTypes:     dir.FileTypes,
		}
	}

	// Create pagination data
	pagination := &Pagination{
		BaseURL:     "/directories",
		CurrentPage: page,
		TotalPages:  totalPages,
		HasPrev:     page > 1,
		HasNext:     page < totalPages,
		PrevPage:    page - 1,
		NextPage:    page + 1,
	}

	data := DirectoriesTemplateData{
		BaseTemplateData: BaseTemplateData{
			Title:      "Directories",
			GatewayUrl: s.gatewayUrl,
		},
		Directories:      directoryData,
		TotalDirectories: totalDirectories,
		Pagination:       pagination,
	}

	if err := RenderDirectories(w, data); err != nil {
		http.Error(w, "Template error: "+err.Error(), http.StatusInternalServerError)
	}
}

func (s *Server) handleDirectoryDetails(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	blobId := vars["blobId"]

	directory, err := s.indexDB.GetDirectoryByID(blobId)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Directory not found", http.StatusNotFound)
		} else {
			http.Error(w, "Error fetching directory: "+err.Error(), http.StatusInternalServerError)
		}
		return
	}

	directoryData := &DirectoryData{
		BlobID:        directory.BlobID,
		DirectoryName: directory.DirectoryName,
		DirectoryHash: directory.DirectoryHash,
		FileCount:     directory.FileCount,
		TotalSize:     formatBytes(directory.TotalSize),
		BlockHeight:   directory.BlockHeight,
		FileTypes:     directory.FileTypes,
	}

	data := DirectoryDetailsTemplateData{
		BaseTemplateData: BaseTemplateData{
			Title:      directory.DirectoryName,
			GatewayUrl: s.gatewayUrl,
		},
		Directory: directoryData,
	}

	if err := RenderDirectoryDetails(w, data); err != nil {
		http.Error(w, "Template error: "+err.Error(), http.StatusInternalServerError)
	}
}

func (s *Server) handleSearch(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")

	var files []FileData
	hasSearched := query != ""

	if hasSearched {
		filters := state.FileFilters{}

		searchFiles, err := s.indexDB.SearchFiles(query, filters, 50, 0)
		if err == nil {
			files = make([]FileData, len(searchFiles))
			for i, file := range searchFiles {
				files[i] = FileData{
					BlobID:      file.BlobID,
					FileName:    file.FileName,
					MimeType:    file.MimeType,
					FileSize:    formatBytes(file.FileSize),
					BlockHeight: file.BlockHeight,
					ChunkCount:  file.ChunkCount,
				}
			}
		}
	}

	data := SearchTemplateData{
		BaseTemplateData: BaseTemplateData{
			Title:      "Search",
			GatewayUrl: s.gatewayUrl,
		},
		Query:       query,
		Files:       files,
		HasSearched: hasSearched,
	}

	if err := RenderSearch(w, data); err != nil {
		http.Error(w, "Template error: "+err.Error(), http.StatusInternalServerError)
	}
}

func (s *Server) apiGetBlocks(w http.ResponseWriter, r *http.Request) {
	limit := 50
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}

	blocks, err := s.indexDB.GetBlocks(limit, 0)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	blockData := make([]map[string]any, len(blocks))
	for i, block := range blocks {
		blockData[i] = map[string]any{
			"height":            block.Height,
			"hash":              truncateString(block.Hash, 20),
			"full_hash":         block.Hash,
			"timestamp":         block.Timestamp.Format("2006-01-02 15:04:05"),
			"celestia_height":   block.CelestiaHeight,
			"total_files":       block.TotalFiles,
			"total_directories": block.TotalDirectories,
			"total_chunks":      block.TotalChunks,
			"storage_used":      formatBytes(block.StorageUsed),
			"storage_bytes":     block.StorageUsed,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"blocks": blockData,
		"total":  len(blockData),
	})
}

func (s *Server) apiGetAnalytics(w http.ResponseWriter, r *http.Request) {
	analytics, err := s.indexDB.GetStorageAnalytics()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(analytics)
}

func formatBytes(bytes any) string {
	var b uint64
	switch v := bytes.(type) {
	case uint64:
		b = v
	case string:
		return v // Already formatted
	default:
		return "0 B"
	}

	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := uint64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}

func truncateString(s string, length int) string {
	if len(s) <= length {
		return s
	}
	half := length / 2
	return s[:half] + "..." + s[len(s)-half:]
}
