package sync

import (
	"context"
	"fmt"

	"github.com/forma-dev/blobcast/pkg/celestia"
	"github.com/forma-dev/blobcast/pkg/crypto"
	"github.com/forma-dev/blobcast/pkg/types"
	"google.golang.org/protobuf/proto"

	pbRollupV1 "github.com/forma-dev/blobcast/pkg/proto/blobcast/rollup/v1"
	pbStorageV1 "github.com/forma-dev/blobcast/pkg/proto/blobcast/storage/v1"
)

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
		Commitment: crypto.Hash(dirManifestCommitment),
	}, nil
}

func PutFileManifest(ctx context.Context, submitter *Submitter, fileManifest *pbStorageV1.FileManifest) (*types.BlobIdentifier, error) {
	blobcastEnvelope := &pbRollupV1.BlobcastEnvelope{
		Payload: &pbRollupV1.BlobcastEnvelope_FileManifest{
			FileManifest: fileManifest,
		},
	}

	blobcastEnvelopeData, err := proto.Marshal(blobcastEnvelope)
	if err != nil {
		return nil, fmt.Errorf("error marshalling file manifest into blobcast envelope: %v", err)
	}

	submissionResult := submitter.SubmitWithPriority(blobcastEnvelopeData)

	var fileId *types.BlobIdentifier
	select {
	case result := <-submissionResult:
		if result.Error != nil {
			return nil, fmt.Errorf("error submitting file manifest: %v", result.Error)
		}
		fileId = result.BlobID
	case <-ctx.Done():
		return nil, fmt.Errorf("context cancelled while waiting for file manifest submission")
	}

	return fileId, nil
}
