package celestia

import (
	"context"
	"crypto/sha256"
	"fmt"
	"sync/atomic"

	"github.com/celestiaorg/celestia-node/blob"
	"github.com/celestiaorg/celestia-node/header"
	"github.com/celestiaorg/go-square/v2/share"
)

type NoopDA struct {
	cfg       DAConfig
	namespace share.Namespace
	nextH     uint64
}

func NewNoopDA() (*NoopDA, error) {
	return &NoopDA{
		cfg:       DAConfig{},
		namespace: share.Namespace{},
		nextH:     1,
	}, nil
}

func (d *NoopDA) Namespace() share.Namespace { return d.namespace }
func (d *NoopDA) Cfg() DAConfig              { return d.cfg }

func (d *NoopDA) fakeCommit(data []byte) []byte {
	sum := sha256.Sum256(data)
	return sum[:]
}
func (d *NoopDA) fakeHeight() uint64 {
	return atomic.AddUint64(&d.nextH, 1)
}

func (d *NoopDA) Store(_ context.Context, msg []byte) ([]byte, uint64, error) {
	h := d.fakeHeight()
	fmt.Printf("[DRY-RUN] would Store %d bytes (fake height %d)\n", len(msg), h)
	return d.fakeCommit(msg), h, nil
}

func (d *NoopDA) StoreBatch(_ context.Context, msgs [][]byte) ([][]byte, uint64, error) {
	h := d.fakeHeight()
	var commits [][]byte
	for i, m := range msgs {
		fmt.Printf("[DRY-RUN]   └─ blob %d: %d bytes\n", i+1, len(m))
		commits = append(commits, d.fakeCommit(m))
	}
	return commits, h, nil
}

func (d *NoopDA) Has(_ context.Context, height uint64, _ []byte) (bool, error) {
	return false, fmt.Errorf("dry-run: no data at fake height %d", height)
}

func (d *NoopDA) Read(_ context.Context, height uint64, _ []byte) ([]byte, error) {
	return nil, fmt.Errorf("dry-run: no data at fake height %d", height)
}

func (d *NoopDA) GetBlobs(_ context.Context, height uint64) ([]*blob.Blob, error) {
	return nil, fmt.Errorf("dry-run: no data at fake height %d", height)
}

func (d *NoopDA) LatestHeight(_ context.Context) (uint64, error) {
	return d.nextH, nil
}

func (d *NoopDA) GetHeader(_ context.Context, height uint64) (*header.ExtendedHeader, error) {
	return nil, fmt.Errorf("dry-run: no data at fake height %d", height)
}

func (d *NoopDA) SubscribeToNewHeaders(_ context.Context) (<-chan CompactHeader, error) {
	return nil, fmt.Errorf("dry-run: no data at fake height %d", d.nextH)
}
