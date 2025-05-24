package rest

import (
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

func (s *Server) setupRoutes() {
	// Block endpoints
	s.router.HandleFunc("/api/v1/blocks/latest", s.getLatestBlock).Methods("GET")
	s.router.HandleFunc("/api/v1/blocks/{heightOrHash}", s.getBlockByHeightOrHash).Methods("GET")

	// Manifest endpoints
	s.router.HandleFunc("/api/v1/manifests/file/{id}", s.getFileManifest).Methods("GET")
	s.router.HandleFunc("/api/v1/manifests/directory/{id}", s.getDirectoryManifest).Methods("GET")

	// File data endpoint
	// s.router.HandleFunc("/api/v1/files/{id}/data", s.getFileData).Methods("GET") // todo

	// Chain info endpoints
	s.router.HandleFunc("/api/v1/chain/info", s.getChainInfo).Methods("GET")

	// Health check
	s.router.HandleFunc("/api/v1/health", s.healthCheck).Methods("GET")
}

func (s *Server) Router() *mux.Router {
	return s.router
}
