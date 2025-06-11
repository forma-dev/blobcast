package state

import (
	"fmt"
	"sync"

	"github.com/cockroachdb/pebble"
	"github.com/forma-dev/blobcast/pkg/crypto"
	"github.com/forma-dev/blobcast/pkg/crypto/mmr"
	"github.com/forma-dev/blobcast/pkg/types"
	"github.com/forma-dev/blobcast/pkg/util"
	"google.golang.org/protobuf/proto"

	pbStorageV1 "github.com/forma-dev/blobcast/pkg/proto/blobcast/storage/v1"
)

type ChainStateTransaction struct {
	batch *pebble.Batch
	state *ChainState

	pendingChunks        map[HashKey][]byte
	pendingChunkHashes   map[HashKey]crypto.Hash
	pendingFileManifests map[*types.BlobIdentifier]*pbStorageV1.FileManifest

	mutex     sync.Mutex
	committed bool
	aborted   bool
}

func NewChainStateTransaction(state *ChainState) *ChainStateTransaction {
	return &ChainStateTransaction{
		state:                state,
		batch:                state.db.NewBatch(),
		pendingChunks:        make(map[HashKey][]byte),
		pendingChunkHashes:   make(map[HashKey]crypto.Hash),
		pendingFileManifests: make(map[*types.BlobIdentifier]*pbStorageV1.FileManifest),
	}
}

func (tx *ChainStateTransaction) SetFinalizedHeight(height uint64) error {
	return tx.batch.Set([]byte(keyFinalizedHeight), util.BytesFromUint64(height), nil)
}

func (tx *ChainStateTransaction) PutStateMMR(height uint64, mmr *mmr.MMR) error {
	if height == 0 {
		return nil
	}

	key := prefixHeightKey(height, tx.state.stateMMRPrefix)
	return tx.batch.Set(key, mmr.Snapshot(), nil)
}

func (tx *ChainStateTransaction) PutStateMMRRaw(height uint64, raw []byte) error {
	if height == 0 {
		return nil
	}

	key := prefixHeightKey(height, tx.state.stateMMRPrefix)
	return tx.batch.Set(key, raw, nil)
}

func (tx *ChainStateTransaction) PutBlock(height uint64, block *types.Block) error {
	key := prefixHeightKey(height, tx.state.blockPrefix)
	if err := tx.batch.Set(key, block.Bytes(), nil); err != nil {
		return err
	}

	hash := block.Hash()
	hashKey := prefixKey(hash.Bytes(), tx.state.blockHashPrefix)
	return tx.batch.Set(hashKey, util.BytesFromUint64(height), nil)
}

func (tx *ChainStateTransaction) PutChunk(key HashKey, value []byte, hash crypto.Hash) error {
	if err := tx.batch.Set(prefixKey(key[:], tx.state.chunkPrefix), value, nil); err != nil {
		return err
	}

	tx.pendingChunks[key] = value

	if err := tx.batch.Set(prefixKey(key[:], tx.state.chunkHashPrefix), hash.Bytes(), nil); err != nil {
		return err
	}

	tx.pendingChunkHashes[key] = hash
	return nil
}

func (tx *ChainStateTransaction) GetChunk(key HashKey) ([]byte, bool, error) {
	value, exists := tx.pendingChunks[key]
	if exists {
		return value, true, nil
	}
	return tx.state.GetChunk(key)
}

func (tx *ChainStateTransaction) GetChunkHash(key HashKey) (crypto.Hash, bool, error) {
	hash, exists := tx.pendingChunkHashes[key]
	if exists {
		return hash, true, nil
	}
	return tx.state.GetChunkHash(key)
}

func (tx *ChainStateTransaction) PutFileManifest(id *types.BlobIdentifier, manifest *pbStorageV1.FileManifest) error {
	data, err := proto.Marshal(manifest)
	if err != nil {
		return fmt.Errorf("marshal file manifest: %w", err)
	}

	tx.pendingFileManifests[id] = manifest
	key := prefixKey(id.Bytes(), tx.state.fileManifestPrefix)
	return tx.batch.Set(key, data, nil)
}

func (tx *ChainStateTransaction) GetFileManifest(id *types.BlobIdentifier) (*pbStorageV1.FileManifest, bool, error) {
	value, exists := tx.pendingFileManifests[id]
	if exists {
		return value, true, nil
	}
	return tx.state.GetFileManifest(id)
}

func (tx *ChainStateTransaction) PutDirectoryManifest(id *types.BlobIdentifier, manifest *pbStorageV1.DirectoryManifest) error {
	data, err := proto.Marshal(manifest)
	if err != nil {
		return fmt.Errorf("marshal directory manifest: %w", err)
	}

	key := prefixKey(id.Bytes(), tx.state.dirManifestPrefix)
	return tx.batch.Set(key, data, nil)
}

func (tx *ChainStateTransaction) Commit() error {
	tx.mutex.Lock()
	defer tx.mutex.Unlock()
	return tx.commit()
}

func (tx *ChainStateTransaction) commit() error {
	if tx.committed || tx.aborted {
		return fmt.Errorf("transaction already finished")
	}

	if err := tx.batch.Commit(pebble.Sync); err != nil {
		tx.abort()
		return fmt.Errorf("error committing transaction: %v", err)
	}

	tx.pendingChunks = nil
	tx.pendingChunkHashes = nil
	tx.pendingFileManifests = nil
	tx.committed = true
	tx.state.pendingTransaction = nil
	return nil
}

func (tx *ChainStateTransaction) Abort() error {
	tx.mutex.Lock()
	defer tx.mutex.Unlock()
	return tx.abort()
}

func (tx *ChainStateTransaction) abort() error {
	if tx.committed || tx.aborted {
		return fmt.Errorf("transaction already finished")
	}
	tx.pendingChunks = nil
	tx.pendingChunkHashes = nil
	tx.pendingFileManifests = nil
	tx.aborted = true
	tx.batch.Close()
	tx.state.pendingTransaction = nil
	return nil
}
