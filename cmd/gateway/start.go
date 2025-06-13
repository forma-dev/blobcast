package gateway

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/forma-dev/blobcast/cmd"
	"github.com/forma-dev/blobcast/pkg/crypto"
	"github.com/forma-dev/blobcast/pkg/net/middleware"
	"github.com/forma-dev/blobcast/pkg/types"
	"github.com/forma-dev/blobcast/pkg/util"
	"github.com/spf13/cobra"

	pbStorageV1 "github.com/forma-dev/blobcast/pkg/proto/blobcast/storage/v1"
	pbStorageapisV1 "github.com/forma-dev/blobcast/pkg/proto/blobcast/storageapis/v1"
)

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start a blobcast gateway server",
	RunE:  runStart,
}

func init() {
	gatewayCmd.AddCommand(startCmd)
	startCmd.Flags().StringVarP(&flagAddr, "addr", "a", "127.0.0.1", "Address to listen on")
	startCmd.Flags().StringVarP(&flagPort, "port", "p", "8080", "Port to listen on")
	startCmd.Flags().
		StringVar(&flagNodeGRPC, "node-grpc", cmd.GetEnvWithDefault("BLOBCAST_NODE_GRPC", "127.0.0.1:50051"), "gRPC address for a blobcast full node")
}

func runStart(command *cobra.Command, args []string) error {
	conn, err := util.NewGRPCClient(flagNodeGRPC)
	if err != nil {
		return fmt.Errorf("error creating gRPC client: %v", err)
	}
	defer conn.Close()

	storageClient := pbStorageapisV1.NewStorageServiceClient(conn)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		directoryHandler(w, r, storageClient)
	})

	// Apply logging middleware
	loggedHandler := middleware.LogRequestMiddleware(handler)

	addr := flagAddr + ":" + flagPort
	slog.Info("Blobcast explorer listening", "addr", addr)
	return http.ListenAndServe(addr, loggedHandler)
}

func directoryHandler(w http.ResponseWriter, r *http.Request, storageClient pbStorageapisV1.StorageServiceClient) {
	// Handle static files
	if strings.HasPrefix(r.URL.Path, "/static/") {
		ServeStatic(w, r)
		return
	}

	rawPath := strings.TrimPrefix(r.URL.Path, "/")
	if rawPath == "" {
		if err := RenderHome(w); err != nil {
			slog.Error("Template rendering error", "error", err)
			http.Error(w, "Template error", http.StatusInternalServerError)
		}
		return
	}

	parts := strings.SplitN(rawPath, "/", 2)
	manifestID := parts[0]
	subPath := ""
	if len(parts) > 1 {
		subPath = strings.TrimPrefix(parts[1], "/")
	}

	manifestIdentifier, err := types.BlobIdentifierFromString(manifestID)
	if err != nil {
		http.Error(w, "invalid manifest id", http.StatusBadRequest)
		return
	}

	dirManifestResponse, err := storageClient.GetDirectoryManifest(context.Background(), &pbStorageapisV1.GetDirectoryManifestRequest{
		Id: manifestIdentifier.ID(),
	})

	if err != nil {
		// check if this is a file manifest
		_, err := storageClient.GetFileManifest(context.Background(), &pbStorageapisV1.GetFileManifestRequest{
			Id: manifestIdentifier.ID(),
		})
		if err != nil {
			http.Error(w, "Not found", http.StatusNotFound)
			return
		}

		// serve the file
		serveFile(w, r, storageClient, manifestIdentifier)
		return
	}

	dirManifest := dirManifestResponse.Manifest

	// Check if the subPath is a file in the manifest
	for _, f := range dirManifest.Files {
		if f.RelativePath == subPath {
			fileManifestIdentifier := &types.BlobIdentifier{
				Height:     f.Id.Height,
				Commitment: crypto.Hash(f.Id.Commitment),
			}
			serveFile(w, r, storageClient, fileManifestIdentifier)
			return
		}
	}

	// Build map of immediate children for current subPath
	children := make(map[string]bool) // name -> isDirectory

	for _, f := range dirManifest.Files {
		rel := f.RelativePath
		if subPath != "" {
			if !strings.HasPrefix(rel, subPath+"/") {
				continue
			}
			rel = strings.TrimPrefix(rel, subPath+"/")
		}

		if !strings.Contains(rel, "/") {
			children[rel] = false
		} else {
			first := strings.SplitN(rel, "/", 2)[0]
			children[first] = true
		}
	}

	// Format manifest ID for display
	displayID := manifestID
	if len(manifestID) > 20 {
		displayID = manifestID[:10] + "..." + manifestID[len(manifestID)-10:]
	}

	// Parse pagination parameters
	page := 1
	if p := r.URL.Query().Get("page"); p != "" {
		if parsed, err := strconv.Atoi(p); err == nil && parsed > 0 {
			page = parsed
		}
	}

	// Create separate slices for directories and files
	var dirs, files []string
	for name, isDir := range children {
		if isDir {
			dirs = append(dirs, name)
		} else {
			files = append(files, name)
		}
	}

	// Sort both slices alphabetically
	sort.Strings(dirs)
	sort.Strings(files)

	// Calculate pagination for files (100 files per page)
	filesPerPage := 100
	totalFiles := len(files)
	totalPages := (totalFiles + filesPerPage - 1) / filesPerPage
	if totalPages == 0 {
		totalPages = 1
	}

	// Ensure page is within bounds
	if page > totalPages {
		page = totalPages
	}

	// Calculate file slice boundaries
	startIndex := (page - 1) * filesPerPage
	endIndex := min(startIndex+filesPerPage, totalFiles)

	// Slice files for current page
	var pagedFiles []string
	if startIndex < totalFiles {
		pagedFiles = files[startIndex:endIndex]
	}

	// batch get file manifests only for current page
	fileManifestIds := make([]string, 0, len(pagedFiles))
	filePathToManifestId := make(map[string]string)
	for _, name := range pagedFiles {
		filePath := name
		if subPath != "" {
			filePath = subPath + "/" + name
		}

		// Find the matching file in dirManifest.Files for this paged file
		for _, f := range dirManifest.Files {
			if f.RelativePath == filePath || (subPath == "" && f.RelativePath == name) {
				manifestId := types.BlobIdentifierFromProto(f.Id).ID()
				fileManifestIds = append(fileManifestIds, manifestId)
				filePathToManifestId[filePath] = manifestId
				break
			}
		}
	}

	var fileManifests map[string]*pbStorageV1.FileManifest
	if len(fileManifestIds) > 0 {
		fileManifestsResp, err := storageClient.BatchGetFileManifest(context.Background(), &pbStorageapisV1.BatchGetFileManifestRequest{
			Ids: fileManifestIds,
		})
		if err == nil {
			fileManifests = make(map[string]*pbStorageV1.FileManifest)
			for _, f := range fileManifestsResp.Manifests {
				fileManifests[f.Id] = f.Manifest
			}
		}
	}

	// Build template data
	data := DirectoryTemplateData{
		BaseTemplateData: BaseTemplateData{
			Title: "Blobcast - " + manifestID,
		},
		ManifestID:  manifestID,
		DisplayID:   displayID,
		SubPath:     subPath,
		HasParent:   subPath != "",
		Directories: make([]DirectoryItem, 0),
		Files:       make([]FileItem, 0),
		TotalFiles:  totalFiles,
	}

	// Calculate parent path if we have a subPath
	if subPath != "" {
		if idx := strings.LastIndex(subPath, "/"); idx >= 0 {
			data.ParentPath = subPath[:idx]
		} else {
			data.ParentPath = ""
		}
	}

	// Create pagination data
	var pagination *Pagination
	if totalPages > 1 {
		baseURL := "/" + manifestID
		if subPath != "" {
			baseURL += "/" + subPath
		}

		pagination = &Pagination{
			BaseURL:     baseURL,
			CurrentPage: page,
			TotalPages:  totalPages,
			HasPrev:     page > 1,
			HasNext:     page < totalPages,
			PrevPage:    page - 1,
			NextPage:    page + 1,
		}
	}
	data.Pagination = pagination

	// Copy directory and file data to template
	for _, name := range dirs {
		path := name
		if subPath != "" {
			path = subPath + "/" + name
		}
		data.Directories = append(data.Directories, DirectoryItem{
			Name: name,
			Path: path,
		})
	}

	// Build files data for current page
	for _, name := range pagedFiles {
		filePath := name
		if subPath != "" {
			filePath = subPath + "/" + name
		}

		manifestId := filePathToManifestId[filePath]
		if manifestId == "" {
			continue
		}

		var fileSize string
		var mimeType string

		fileManifest := fileManifests[manifestId]
		if fileManifest != nil {
			fileSize = formatFileSize(fileManifest.FileSize)
			mimeType = fileManifest.MimeType
			if mimeType == "" {
				mimeType = "application/octet-stream"
			}
		}

		data.Files = append(data.Files, FileItem{
			Name:         name,
			Path:         filePath,
			Size:         fileSize,
			MimeType:     mimeType,
			ManifestID:   manifestId,
			ManifestLink: manifestId[:10] + "..." + manifestId[len(manifestId)-10:],
		})
	}

	// Render template instead of manual HTML generation
	if err := RenderDirectory(w, data); err != nil {
		slog.Error("Template rendering error", "error", err)
		http.Error(w, "Template error", http.StatusInternalServerError)
		return
	}
}

func serveFile(w http.ResponseWriter, r *http.Request, storageClient pbStorageapisV1.StorageServiceClient, fileManifestIdentifier *types.BlobIdentifier) {
	// Get the file manifest to determine mime type
	fileManifestResponse, err := storageClient.GetFileManifest(context.Background(), &pbStorageapisV1.GetFileManifestRequest{
		Id: fileManifestIdentifier.ID(),
	})
	if err != nil {
		http.Error(w, fmt.Sprintf("error fetching file manifest: %v", err), http.StatusInternalServerError)
		return
	}

	// Get the file data
	fileDataResponse, err := storageClient.GetFileData(context.Background(), &pbStorageapisV1.GetFileDataRequest{
		Id: fileManifestIdentifier.ID(),
	})
	if err != nil {
		http.Error(w, fmt.Sprintf("error fetching file data: %v", err), http.StatusInternalServerError)
		return
	}

	// Set appropriate content type header
	if fileManifestResponse.Manifest.MimeType != "" {
		w.Header().Set("Content-Type", fileManifestResponse.Manifest.MimeType)
	} else {
		w.Header().Set("Content-Type", "application/octet-stream")
	}

	fileSize := len(fileDataResponse.Data)
	// Set content disposition based on the file name
	fileName := filepath.Base(fileManifestResponse.Manifest.FileName)
	if !isMediaContentType(fileManifestResponse.Manifest.MimeType) {
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", fileName))
	}

	// Handle range requests
	rangeHeader := r.Header.Get("Range")
	if rangeHeader != "" {
		// Parse the range header
		if strings.HasPrefix(rangeHeader, "bytes=") {
			rangeHeader = strings.TrimPrefix(rangeHeader, "bytes=")
			rangeParts := strings.Split(rangeHeader, "-")
			if len(rangeParts) == 2 {
				var start, end int
				var err error

				// Parse the start position
				if rangeParts[0] != "" {
					start, err = parseInt(rangeParts[0])
					if err != nil || start >= fileSize {
						http.Error(w, "Invalid range", http.StatusRequestedRangeNotSatisfiable)
						return
					}
				}

				// Parse the end position
				if rangeParts[1] != "" {
					end, err = parseInt(rangeParts[1])
					if err != nil || end >= fileSize {
						end = fileSize - 1
					}
				} else {
					end = fileSize - 1
				}

				// Validate the range
				if start > end || start >= fileSize {
					http.Error(w, "Invalid range", http.StatusRequestedRangeNotSatisfiable)
					return
				}

				// Calculate the length of the range
				contentLength := end - start + 1

				// Set headers for partial content
				w.Header().Set("Content-Length", fmt.Sprintf("%d", contentLength))
				w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, fileSize))
				w.Header().Set("Accept-Ranges", "bytes")
				w.WriteHeader(http.StatusPartialContent)

				// Serve the partial content
				w.Write(fileDataResponse.Data[start : end+1])
				slog.Info("serving partial file", "file", fileManifestResponse.Manifest.FileName, "range", fmt.Sprintf("%d-%d/%d", start, end, fileSize))
				return
			}
		}
		// If we get here, the range format was invalid
		http.Error(w, "Invalid range format", http.StatusRequestedRangeNotSatisfiable)
		return
	}

	// Set content length header for full file
	w.Header().Set("Content-Length", fmt.Sprintf("%d", fileSize))
	w.Header().Set("Accept-Ranges", "bytes")

	slog.Info(
		"serving file",
		"file",
		fileManifestResponse.Manifest.FileName,
		"size",
		fileManifestResponse.Manifest.FileSize,
		"mime_type",
		fileManifestResponse.Manifest.MimeType,
	)

	// Serve the full file content
	w.Write(fileDataResponse.Data)
}

// parseInt parses a string to an integer with error handling
func parseInt(s string) (int, error) {
	return strconv.Atoi(s)
}

// Helper function to determine if a content type is media that should be displayed in browser
func isMediaContentType(contentType string) bool {
	prefix := strings.Split(contentType, "/")[0]
	return prefix == "image" || prefix == "video" || prefix == "audio" ||
		contentType == "text/plain" || contentType == "text/html" ||
		contentType == "application/pdf"
}

// formatFileSize formats a file size in bytes to a human-readable string
func formatFileSize(size uint64) string {
	const unit = 1024
	if size < unit {
		return fmt.Sprintf("%d B", size)
	}
	div, exp := uint64(unit), 0
	for n := size / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(size)/float64(div), "KMGTPE"[exp])
}
