package rest

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/forma-dev/blobcast/pkg/types"
	"github.com/gorilla/mux"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pbRollupV1 "github.com/forma-dev/blobcast/pkg/proto/blobcast/rollup/v1"
	pbRollupapisV1 "github.com/forma-dev/blobcast/pkg/proto/blobcast/rollupapis/v1"
	pbStorageV1 "github.com/forma-dev/blobcast/pkg/proto/blobcast/storage/v1"
	pbStorageapisV1 "github.com/forma-dev/blobcast/pkg/proto/blobcast/storageapis/v1"
)

type ChainInfoResponse struct {
	ChainID              string `json:"chain_id"`
	FinalizedHeight      uint64 `json:"finalized_height"`
	CelestiaHeightOffset uint64 `json:"celestia_height_offset"`
}

type BlockResponse struct {
	ChainID             string   `json:"chain_id"`
	Height              uint64   `json:"height"`
	CelestiaBlockHeight uint64   `json:"celestia_block_height"`
	Timestamp           int64    `json:"timestamp"`
	Hash                string   `json:"hash"`
	ParentHash          string   `json:"parent_hash"`
	DirsRoot            string   `json:"dirs_root"`
	FilesRoot           string   `json:"files_root"`
	ChunksRoot          string   `json:"chunks_root"`
	StateRoot           string   `json:"state_root"`
	Chunks              []string `json:"chunks"`
	Files               []string `json:"files"`
	Dirs                []string `json:"dirs"`
}

type FileManifestResponse struct {
	FileName             string   `json:"file_name"`
	MimeType             string   `json:"mime_type"`
	FileSize             uint64   `json:"file_size"`
	FileHash             string   `json:"file_hash"`
	CompressionAlgorithm string   `json:"compression_algorithm"`
	ChunkIDs             []string `json:"chunk_ids"`
}

type DirectoryManifestResponse struct {
	DirectoryName string                  `json:"directory_name"`
	DirectoryHash string                  `json:"directory_hash"`
	Files         []FileReferenceResponse `json:"files"`
}

type FileReferenceResponse struct {
	ID           string `json:"id"`
	RelativePath string `json:"relative_path"`
}

func (s *Server) getChainInfo(w http.ResponseWriter, r *http.Request) {
	chainInfo, err := s.rollupClient.GetChainInfo(context.Background(), &pbRollupapisV1.GetChainInfoRequest{})
	if err != nil {
		http.Error(w, "Failed to get chain info", http.StatusInternalServerError)
		return
	}

	response := ChainInfoResponse{
		ChainID:              chainInfo.ChainId,
		FinalizedHeight:      chainInfo.FinalizedHeight,
		CelestiaHeightOffset: chainInfo.CelestiaHeightOffset,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (s *Server) getBlockByHeightOrHash(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	heightOrHash := vars["heightOrHash"]

	if strings.HasPrefix(heightOrHash, "0x") || len(heightOrHash) == 64 {
		s.getBlockByHash(w, r)
	} else {
		s.getBlockByHeight(w, r)
	}
}

func (s *Server) getBlockByHeight(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	heightStr := vars["heightOrHash"]

	height, err := strconv.ParseUint(heightStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid height", http.StatusBadRequest)
		return
	}

	resp, err := s.rollupClient.GetBlockByHeight(context.Background(), &pbRollupapisV1.GetBlockByHeightRequest{
		Height: height,
	})

	if err != nil {
		if status.Code(err) == codes.NotFound {
			http.Error(w, "Block not found", http.StatusNotFound)
		} else {
			http.Error(w, "Failed to get block", http.StatusInternalServerError)
		}
		return
	}

	response := blockToResponse(resp.Block)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (s *Server) getBlockByHash(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	hashStr := vars["heightOrHash"]

	if strings.HasPrefix(hashStr, "0x") {
		hashStr = hashStr[2:]
	}

	hashBytes, err := hex.DecodeString(hashStr)
	if err != nil {
		http.Error(w, "Invalid hash format", http.StatusBadRequest)
		return
	}

	if len(hashBytes) != 32 {
		http.Error(w, "Hash must be 32 bytes", http.StatusBadRequest)
		return
	}

	resp, err := s.rollupClient.GetBlockByHash(context.Background(), &pbRollupapisV1.GetBlockByHashRequest{
		Hash: hashBytes,
	})
	if err != nil {
		if status.Code(err) == codes.NotFound {
			http.Error(w, "Block not found", http.StatusNotFound)
		} else {
			http.Error(w, "Failed to get block", http.StatusInternalServerError)
		}
		return
	}

	response := blockToResponse(resp.Block)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (s *Server) getLatestBlock(w http.ResponseWriter, r *http.Request) {
	resp, err := s.rollupClient.GetLatestBlock(context.Background(), &pbRollupapisV1.GetLatestBlockRequest{})
	if err != nil {
		if status.Code(err) == codes.NotFound {
			http.Error(w, "No blocks available", http.StatusNotFound)
		} else {
			http.Error(w, "Failed to get latest block", http.StatusInternalServerError)
		}
		return
	}

	response := blockToResponse(resp.Block)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (s *Server) getFileManifest(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr := vars["id"]

	id, err := types.BlobIdentifierFromString(idStr)
	if err != nil {
		http.Error(w, "Invalid manifest ID", http.StatusBadRequest)
		return
	}

	resp, err := s.storageClient.GetFileManifest(context.Background(), &pbStorageapisV1.GetFileManifestRequest{
		Id: id.Proto(),
	})
	if err != nil {
		http.Error(w, "Manifest not found", http.StatusNotFound)
		return
	}

	response := FileManifestResponse{
		FileName:             resp.Manifest.FileName,
		MimeType:             resp.Manifest.MimeType,
		FileSize:             resp.Manifest.FileSize,
		FileHash:             hex.EncodeToString(resp.Manifest.FileHash),
		CompressionAlgorithm: compressionAlgorithmToString(resp.Manifest.CompressionAlgorithm),
		ChunkIDs:             make([]string, len(resp.Manifest.Chunks)),
	}

	for i, chunk := range resp.Manifest.Chunks {
		response.ChunkIDs[i] = types.BlobIdentifierFromProto(chunk.Id).String()
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (s *Server) getDirectoryManifest(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr := vars["id"]

	id, err := types.BlobIdentifierFromString(idStr)
	if err != nil {
		http.Error(w, "Invalid manifest ID", http.StatusBadRequest)
		return
	}

	resp, err := s.storageClient.GetDirectoryManifest(context.Background(), &pbStorageapisV1.GetDirectoryManifestRequest{
		Id: id.Proto(),
	})
	if err != nil {
		http.Error(w, "Manifest not found", http.StatusNotFound)
		return
	}

	response := DirectoryManifestResponse{
		DirectoryName: resp.Manifest.DirectoryName,
		DirectoryHash: hex.EncodeToString(resp.Manifest.DirectoryHash),
		Files:         make([]FileReferenceResponse, len(resp.Manifest.Files)),
	}

	for i, file := range resp.Manifest.Files {
		response.Files[i] = FileReferenceResponse{
			ID:           types.BlobIdentifierFromProto(file.Id).String(),
			RelativePath: file.RelativePath,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (s *Server) healthCheck(w http.ResponseWriter, r *http.Request) {
	response := map[string]string{
		"status": "healthy",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func compressionAlgorithmToString(algorithm pbStorageV1.CompressionAlgorithm) string {
	switch algorithm {
	case pbStorageV1.CompressionAlgorithm_COMPRESSION_ALGORITHM_NONE:
		return "none"
	}
	return "unknown"
}

func blockToResponse(block *pbRollupV1.Block) BlockResponse {
	chunks := make([]string, len(block.Body.Chunks))
	files := make([]string, len(block.Body.Files))
	dirs := make([]string, len(block.Body.Dirs))

	for i, chunk := range block.Body.Chunks {
		chunks[i] = types.BlobIdentifierFromProto(chunk).String()
	}

	for i, file := range block.Body.Files {
		files[i] = types.BlobIdentifierFromProto(file).String()
	}

	for i, dir := range block.Body.Dirs {
		dirs[i] = types.BlobIdentifierFromProto(dir).String()
	}

	bh := types.BlockHeaderFromProto(block.Header)

	return BlockResponse{
		ChainID:             bh.ChainID,
		Height:              bh.Height,
		CelestiaBlockHeight: bh.CelestiaBlockHeight,
		Timestamp:           bh.Timestamp.Unix(),
		Hash:                bh.Hash().String(),
		ParentHash:          bh.ParentHash.String(),
		DirsRoot:            bh.DirsRoot.String(),
		FilesRoot:           bh.FilesRoot.String(),
		ChunksRoot:          bh.ChunksRoot.String(),
		StateRoot:           bh.StateRoot.String(),
		Chunks:              chunks,
		Files:               files,
		Dirs:                dirs,
	}
}
