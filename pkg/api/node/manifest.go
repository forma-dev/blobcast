package node

import (
	"context"
	"fmt"

	"github.com/forma-dev/blobcast/pkg/state"
	"github.com/forma-dev/blobcast/pkg/types"

	pbStorageV1 "github.com/forma-dev/blobcast/pkg/proto/blobcast/storage/v1"
)

func GetDirectoryManifest(ctx context.Context, id *types.BlobIdentifier) (*pbStorageV1.DirectoryManifest, error) {
	chainState, err := state.GetChainState()
	if err != nil {
		return nil, fmt.Errorf("error getting chain state: %v", err)
	}

	manifest, found, err := chainState.GetDirectoryManifest(id)
	if err != nil {
		return nil, fmt.Errorf("error getting directory manifest: %v", err)
	}

	if !found {
		return nil, fmt.Errorf("directory manifest not found")
	}

	return manifest, nil
}

func GetFileManifest(ctx context.Context, id *types.BlobIdentifier) (*pbStorageV1.FileManifest, error) {
	chainState, err := state.GetChainState()
	if err != nil {
		return nil, fmt.Errorf("error getting chain state: %v", err)
	}

	manifest, found, err := chainState.GetFileManifest(id)
	if err != nil {
		return nil, fmt.Errorf("error getting file manifest: %v", err)
	}

	if !found {
		return nil, fmt.Errorf("file manifest not found")
	}

	return manifest, nil
}
