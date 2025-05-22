package celestia

import (
	"context"
	"time"

	"github.com/celestiaorg/celestia-node/blob"
	"github.com/celestiaorg/celestia-node/header"
	"github.com/celestiaorg/go-square/v2/share"
)

type BlobStore interface {
	Store(ctx context.Context, msg []byte) (commitment []byte, height uint64, err error)
	StoreBatch(ctx context.Context, msgs [][]byte) (commitments [][]byte, height uint64, err error)
	Has(ctx context.Context, height uint64, commitment []byte) (bool, error)
	Read(ctx context.Context, height uint64, commitment []byte) ([]byte, error)
	GetBlobs(ctx context.Context, height uint64) ([]*blob.Blob, error)
	LatestHeight(ctx context.Context) (uint64, error)
	GetHeader(ctx context.Context, height uint64) (*header.ExtendedHeader, error)
	SubscribeToNewHeaders(ctx context.Context) (<-chan CompactHeader, error)

	Namespace() share.Namespace
	Cfg() DAConfig
}

type CompactHeader struct {
	Height uint64
	Time   time.Time
}
