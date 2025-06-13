package celestia

import (
	"context"
	"encoding/hex"
	"errors"
	"log/slog"
	"time"

	"github.com/celestiaorg/celestia-node/api/rpc/client"
	"github.com/celestiaorg/celestia-node/blob"
	"github.com/celestiaorg/celestia-node/header"
	"github.com/celestiaorg/celestia-node/state"
	"github.com/celestiaorg/go-square/v2/share"
	"github.com/forma-dev/blobcast/pkg/util"
)

// Client for interacting with Celestia
type CelestiaDA struct {
	cfg       DAConfig
	client    *client.Client
	namespace share.Namespace
}

// Create a new Celestia DA client
func NewCelestiaDA(cfg DAConfig) (*CelestiaDA, error) {
	daClient, err := client.NewClient(context.Background(), cfg.Rpc, cfg.AuthToken)
	if err != nil {
		return nil, err
	}

	if cfg.NamespaceId == "" {
		return nil, errors.New("namespace id cannot be blank")
	}

	namespace, err := HexToNamespaceV0(cfg.NamespaceId)
	if err != nil {
		return nil, err
	}

	return &CelestiaDA{
		cfg:       cfg,
		client:    daClient,
		namespace: namespace,
	}, nil
}

func (c *CelestiaDA) Store(ctx context.Context, message []byte) ([]byte, uint64, error) {
	dataBlob, err := blob.NewBlobV0(c.namespace, message)
	if err != nil {
		return nil, 0, err
	}

	height, err := c.submitBlobs(ctx, []*blob.Blob{dataBlob})
	if err != nil {
		return nil, height, err
	}

	return dataBlob.Commitment, height, nil
}

func (c *CelestiaDA) StoreBatch(ctx context.Context, messages [][]byte) ([][]byte, uint64, error) {
	if len(messages) == 0 {
		return nil, 0, errors.New("no messages to store")
	}

	blobs := make([]*blob.Blob, 0, len(messages))
	commitments := make([][]byte, 0, len(messages))

	for _, message := range messages {
		dataBlob, err := blob.NewBlobV0(c.namespace, message)
		if err != nil {
			return nil, 0, err
		}
		blobs = append(blobs, dataBlob)
		commitments = append(commitments, dataBlob.Commitment)
	}

	height, err := c.submitBlobs(ctx, blobs)
	if err != nil {
		return nil, height, err
	}

	return commitments, height, nil
}

func (c *CelestiaDA) Has(ctx context.Context, height uint64, commitment []byte) (bool, error) {
	logAttrs := []slog.Attr{
		slog.Uint64("height", height),
		slog.String("commitment", hex.EncodeToString(commitment)),
	}

	slog.LogAttrs(ctx, slog.LevelDebug, "Checking if data exists in Celestia", logAttrs...)

	var included bool
	err := util.Retry(ctx, util.DefaultRetryConfig(), func() error {
		proof, proofErr := c.client.Blob.GetProof(ctx, height, c.namespace, commitment)
		if proofErr != nil {
			slog.Error("Error getting proof from Celestia", "error", proofErr)
			return proofErr
		}
		var includedErr error
		included, includedErr = c.client.Blob.Included(ctx, height, c.namespace, proof, commitment)
		if includedErr != nil {
			slog.Error("Error checking if data exists in Celestia", "error", includedErr)
		}
		return includedErr
	})

	return included, err
}

func (c *CelestiaDA) Read(ctx context.Context, height uint64, commitment []byte) ([]byte, error) {
	logAttrs := []slog.Attr{
		slog.Uint64("height", height),
		slog.String("commitment", hex.EncodeToString(commitment)),
	}

	slog.LogAttrs(ctx, slog.LevelDebug, "Fetching data from Celestia", logAttrs...)

	var blob *blob.Blob
	err := util.Retry(ctx, util.DefaultRetryConfig(), func() error {
		var opErr error
		blob, opErr = c.client.Blob.Get(ctx, height, c.namespace, commitment)
		if opErr != nil {
			slog.Error("Error fetching data from Celestia", "error", opErr)
		}
		return opErr
	})

	if err != nil {
		logAttrs = append(logAttrs, slog.String("error", err.Error()))
		slog.LogAttrs(ctx, slog.LevelError, "Error fetching data from Celestia", logAttrs...)
		return nil, err
	}

	slog.LogAttrs(ctx, slog.LevelDebug, "Successfully fetched data from Celestia", logAttrs...)

	return blob.Data(), nil
}

func (c *CelestiaDA) GetBlobs(ctx context.Context, height uint64) ([]*blob.Blob, error) {
	var blobs []*blob.Blob
	err := util.Retry(ctx, util.DefaultRetryConfig(), func() error {
		var opErr error
		blobs, opErr = c.client.Blob.GetAll(ctx, height, []share.Namespace{c.namespace})
		if opErr != nil {
			slog.Error("Error fetching blobs from Celestia", "error", opErr)
		}
		return opErr
	})
	return blobs, err
}

func (c *CelestiaDA) LatestHeight(ctx context.Context) (uint64, error) {
	state, err := c.client.Header.SyncState(ctx)
	if err != nil {
		return 0, err
	}
	return state.Height, nil
}

func (c *CelestiaDA) GetHeader(ctx context.Context, height uint64) (*header.ExtendedHeader, error) {
	var header *header.ExtendedHeader
	err := util.Retry(ctx, util.DefaultRetryConfig(), func() error {
		var opErr error
		header, opErr = c.client.Header.GetByHeight(ctx, height)
		if opErr != nil {
			slog.Error("Error fetching header from Celestia", "error", opErr)
		}
		return opErr
	})
	return header, err
}

func (c *CelestiaDA) SubscribeToNewHeaders(ctx context.Context) (<-chan CompactHeader, error) {
	compactHeaderChan := make(chan CompactHeader)

	headerChan, err := c.client.Header.Subscribe(ctx)
	if err != nil {
		return nil, err
	}

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case header := <-headerChan:
				compactHeaderChan <- CompactHeader{
					Height: header.Height(),
					Time:   header.Time(),
				}
			}
		}
	}()

	return compactHeaderChan, nil
}

func (c *CelestiaDA) Namespace() share.Namespace {
	return c.namespace
}

func (c *CelestiaDA) Cfg() DAConfig {
	return c.cfg
}

func (c *CelestiaDA) submitBlobs(ctx context.Context, dataBlobs []*blob.Blob) (uint64, error) {
	submitCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	var height uint64
	options := state.NewTxConfig()
	err := util.Retry(submitCtx, util.DefaultRetryConfig(), func() error {
		var opErr error
		height, opErr = c.client.Blob.Submit(submitCtx, dataBlobs, options)
		if opErr != nil {
			slog.Error("Error submitting blob", "error", opErr)
		}
		return opErr
	})

	if err != nil {
		return 0, err
	}

	if height == 0 {
		return 0, errors.New("unexpected response code, height is 0")
	}

	return height, nil
}
