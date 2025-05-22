package sync

import (
	"context"
	"fmt"

	"github.com/forma-dev/blobcast/pkg/celestia"
	"github.com/forma-dev/blobcast/pkg/state"
	"github.com/forma-dev/blobcast/pkg/types"
	"google.golang.org/protobuf/proto"

	pbRollupV1 "github.com/forma-dev/blobcast/pkg/proto/blobcast/rollup/v1"
	pbStorageV1 "github.com/forma-dev/blobcast/pkg/proto/blobcast/storage/v1"
)

func GetDirectoryManifest(ctx context.Context, da celestia.BlobStore, id *types.BlobIdentifier) (*pbStorageV1.DirectoryManifest, error) {
	manifestState, err := state.GetManifestState()
	if err != nil {
		return nil, fmt.Errorf("error getting manifest state: %v", err)
	}

	manifest, found, err := manifestState.GetDirectoryManifest(id)
	if err != nil {
		return nil, fmt.Errorf("error getting directory manifest: %v", err)
	}

	if found {
		return manifest, nil
	}

	// Fetch from Celestia if not in state
	directoryManifestData, err := da.Read(ctx, id.Height, id.Commitment)
	if err != nil {
		return nil, fmt.Errorf("error reading directory manifest from Celestia: %v", err)
	}

	var directoryManifest pbStorageV1.DirectoryManifest
	err = proto.Unmarshal(directoryManifestData, &directoryManifest)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling directory manifest: %v", err)
	}

	return &directoryManifest, nil
}

func GetFileManifest(ctx context.Context, da celestia.BlobStore, id *types.BlobIdentifier) (*pbStorageV1.FileManifest, error) {
	manifestState, err := state.GetManifestState()
	if err != nil {
		return nil, fmt.Errorf("error getting manifest state: %v", err)
	}

	manifest, found, err := manifestState.GetFileManifest(id)
	if err != nil {
		return nil, fmt.Errorf("error getting file manifest: %v", err)
	}

	if found {
		return manifest, nil
	}

	// Fetch from Celestia if not in state
	fileManifestData, err := da.Read(ctx, id.Height, id.Commitment)
	if err != nil {
		return nil, fmt.Errorf("error reading file manifest from Celestia: %v", err)
	}

	var fileManifest pbStorageV1.FileManifest
	err = proto.Unmarshal(fileManifestData, &fileManifest)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling file manifest: %v", err)
	}

	return &fileManifest, nil
}

func PutDirectoryManifest(ctx context.Context, da celestia.BlobStore, directoryManifest *pbStorageV1.DirectoryManifest) (*types.BlobIdentifier, error) {
	blobcastEnvelope := &pbRollupV1.BlobcastEnvelope{
		Payload: &pbRollupV1.BlobcastEnvelope_DirectoryManifest{
			DirectoryManifest: directoryManifest,
		},
	}

	blobcastEnvelopeData, err := proto.Marshal(blobcastEnvelope)
	if err != nil {
		return nil, fmt.Errorf("error marshalling directory manifest into blobcast envelope: %v", err)
	}

	dirManifestCommitment, dirManifestHeight, err := da.Store(ctx, blobcastEnvelopeData)
	if err != nil {
		return nil, fmt.Errorf("error submitting directory manifest to Celestia: %v", err)
	}

	return &types.BlobIdentifier{
		Height:     dirManifestHeight,
		Commitment: dirManifestCommitment,
	}, nil
}

func PutFileManifest(ctx context.Context, da celestia.BlobStore, fileManifest *pbStorageV1.FileManifest) (*types.BlobIdentifier, error) {
	blobcastEnvelope := &pbRollupV1.BlobcastEnvelope{
		Payload: &pbRollupV1.BlobcastEnvelope_FileManifest{
			FileManifest: fileManifest,
		},
	}

	blobcastEnvelopeData, err := proto.Marshal(blobcastEnvelope)
	if err != nil {
		return nil, fmt.Errorf("error marshalling file manifest into blobcast envelope: %v", err)
	}

	fileManifestCommitment, fileManifestHeight, err := da.Store(ctx, blobcastEnvelopeData)
	if err != nil {
		return nil, fmt.Errorf("error submitting file manifest to Celestia: %v", err)
	}

	return &types.BlobIdentifier{
		Height:     fileManifestHeight,
		Commitment: fileManifestCommitment,
	}, nil
}
