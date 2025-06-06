package rest

import (
	_ "embed"
	"net/http"

	scalargo "github.com/bdpiprava/scalar-go"
	"github.com/gorilla/mux"

	pbRollupapisV1 "github.com/forma-dev/blobcast/pkg/proto/blobcast/rollupapis/v1"
	pbStorageapisV1 "github.com/forma-dev/blobcast/pkg/proto/blobcast/storageapis/v1"
)

type Server struct {
	storageClient pbStorageapisV1.StorageServiceClient
	rollupClient  pbRollupapisV1.RollupServiceClient
	router        *mux.Router
}

func NewServer(storageClient pbStorageapisV1.StorageServiceClient, rollupClient pbRollupapisV1.RollupServiceClient) *Server {
	s := &Server{
		storageClient: storageClient,
		rollupClient:  rollupClient,
		router:        mux.NewRouter(),
	}
	s.setupRoutes()
	return s
}

//go:embed openapi.yaml
var openapiSpec []byte

func (s *Server) setupRoutes() {
	// Block endpoints
	s.router.HandleFunc("/v1/blocks/latest", s.getLatestBlock).Methods("GET")
	s.router.HandleFunc("/v1/blocks/{heightOrHash}", s.getBlockByHeightOrHash).Methods("GET")

	// Manifest endpoints
	s.router.HandleFunc("/v1/manifests/file/{id}", s.getFileManifest).Methods("GET")
	s.router.HandleFunc("/v1/manifests/directory/{id}", s.getDirectoryManifest).Methods("GET")

	// File data endpoint
	// s.router.HandleFunc("/v1/files/{id}/data", s.getFileData).Methods("GET") // todo

	// Chain info endpoints
	s.router.HandleFunc("/v1/chain/info", s.getChainInfo).Methods("GET")

	// Health check
	s.router.HandleFunc("/v1/health", s.healthCheck).Methods("GET")

	// OpenAPI spec
	s.router.HandleFunc("/openapi.yaml", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/yaml")
		w.Write(openapiSpec)
	}).Methods("GET")

	// Api docs
	s.router.HandleFunc("/docs", func(w http.ResponseWriter, r *http.Request) {
		content, err := scalargo.NewV2(
			scalargo.WithSpecURL("/openapi.yaml"),
			scalargo.WithDarkMode(),
		)

		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Write([]byte(content))
	}).Methods("GET")
}

func (s *Server) Router() *mux.Router {
	return s.router
}
