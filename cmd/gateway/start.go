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
	"time"

	"github.com/forma-dev/blobcast/cmd"
	"github.com/forma-dev/blobcast/pkg/net/middleware"
	"github.com/forma-dev/blobcast/pkg/types"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"

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
	// initialize storage client
	keepaliveParams := keepalive.ClientParameters{
		Time:                15 * time.Minute,
		Timeout:             60 * time.Second,
		PermitWithoutStream: true,
	}
	conn, err := grpc.NewClient(
		flagNodeGRPC,
		grpc.WithKeepaliveParams(keepaliveParams),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(1024*1024*1024)), // 1GB for now
	)
	if err != nil {
		return fmt.Errorf("error creating storage client: %v", err)
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

// directoryHandler resolves the manifest, fetches the directory manifest
// from Celestia and renders a minimal HTML view.
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
		Id: manifestIdentifier.Proto(),
	})

	if err != nil {
		// check if this is a file manifest
		_, err := storageClient.GetFileManifest(context.Background(), &pbStorageapisV1.GetFileManifestRequest{
			Id: manifestIdentifier.Proto(),
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
				Commitment: f.Id.Commitment,
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
	}

	// Calculate parent path if we have a subPath
	if subPath != "" {
		if idx := strings.LastIndex(subPath, "/"); idx >= 0 {
			data.ParentPath = subPath[:idx]
		} else {
			data.ParentPath = ""
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

	// Build directories data
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

	// Build files data
	for _, name := range files {
		filePath := name
		if subPath != "" {
			filePath = subPath + "/" + name
		}

		// Find the file in the directory manifest to get its file manifest
		var fileSize string
		var mimeType string
		var fileManifestIdentifier *types.BlobIdentifier

		for _, f := range dirManifest.Files {
			if f.RelativePath == filePath || (subPath == "" && f.RelativePath == name) {
				fileManifestIdentifier = &types.BlobIdentifier{
					Height:     f.Id.Height,
					Commitment: f.Id.Commitment,
				}

				fileManifestResponse, err := storageClient.GetFileManifest(context.Background(), &pbStorageapisV1.GetFileManifestRequest{
					Id: fileManifestIdentifier.Proto(),
				})
				if err == nil {
					fileSize = formatFileSize(fileManifestResponse.Manifest.FileSize)
					mimeType = fileManifestResponse.Manifest.MimeType
					if mimeType == "" {
						mimeType = "application/octet-stream"
					}
				}
				break
			}
		}

		if fileManifestIdentifier != nil {
			data.Files = append(data.Files, FileItem{
				Name:         name,
				Path:         filePath,
				Icon:         getFileIcon(name),
				Size:         fileSize,
				MimeType:     mimeType,
				ManifestID:   fileManifestIdentifier.ID(),
				ManifestLink: fileManifestIdentifier.ID()[:10] + "..." + fileManifestIdentifier.ID()[len(fileManifestIdentifier.ID())-10:],
			})
		}
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
		Id: fileManifestIdentifier.Proto(),
	})
	if err != nil {
		http.Error(w, fmt.Sprintf("error fetching file manifest: %v", err), http.StatusInternalServerError)
		return
	}

	// Get the file data
	fileDataResponse, err := storageClient.GetFileData(context.Background(), &pbStorageapisV1.GetFileDataRequest{
		Id: fileManifestIdentifier.Proto(),
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

// Get a more specific icon based on the file extension
func getFileIcon(fileName string) string {
	ext := strings.ToLower(filepath.Ext(fileName))
	switch ext {
	case ".jpg", ".jpeg", ".png", ".gif", ".webp", ".svg", ".bmp":
		return "🖼️"
	case ".mp4", ".webm", ".mov", ".avi", ".mkv":
		return "🎬"
	case ".mp3", ".wav", ".ogg", ".flac":
		return "🎵"
	case ".pdf":
		return "📑"
	case ".doc", ".docx", ".txt", ".md":
		return "📝"
	case ".xls", ".xlsx", ".csv":
		return "📊"
	case ".zip", ".tar", ".gz", ".rar":
		return "🗜️"
	case ".html", ".htm":
		return "🌐"
	case ".js", ".py", ".go", ".java", ".c", ".cpp", ".ts", ".rs", ".sol":
		return "📜"
	case ".json", ".yaml", ".yml", ".toml", ".xml":
		return "🗃️"
	default:
		return "📄"
	}
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
