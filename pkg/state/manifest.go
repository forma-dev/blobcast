package state

import (
	"fmt"
	"sync"

	"github.com/cockroachdb/pebble"
	"google.golang.org/protobuf/proto"

	"github.com/forma-dev/blobcast/pkg/types"

	pbStorageV1 "github.com/forma-dev/blobcast/pkg/proto/blobcast/storage/v1"
)

var (
	manifestState      *ManifestState
	manifestStateMutex sync.Mutex
)

type ManifestState struct {
	db         *pebble.DB
	dirPrefix  []byte
	filePrefix []byte
}

func GetManifestState() (*ManifestState, error) {
	manifestStateMutex.Lock()
	defer manifestStateMutex.Unlock()

	if manifestState != nil {
		return manifestState, nil
	}

	manifestStateLocal, err := openManifestState()
	if err != nil {
		return nil, err
	}

	manifestState = manifestStateLocal
	return manifestStateLocal, nil
}

func openManifestState() (*ManifestState, error) {
	dbPath, err := getStateDbPath(ManifestStateDb)
	if err != nil {
		return nil, err
	}

	db, err := pebble.Open(dbPath, nil)
	if err != nil {
		return nil, err
	}

	return &ManifestState{
		db:         db,
		dirPrefix:  []byte("dir:"),
		filePrefix: []byte("file:"),
	}, nil
}

func (s *ManifestState) GetDirectoryManifest(id *types.BlobIdentifier) (*pbStorageV1.DirectoryManifest, bool, error) {
	key := prefixKey(id.Bytes(), s.dirPrefix)

	data, closer, err := s.db.Get(key)
	if err != nil {
		if err == pebble.ErrNotFound {
			return nil, false, nil
		}
		return nil, false, err
	}
	defer closer.Close()

	// Copy the data since the slice becomes invalid after closer.Close()
	dataCopy := make([]byte, len(data))
	copy(dataCopy, data)

	manifest := &pbStorageV1.DirectoryManifest{}
	if err := proto.Unmarshal(dataCopy, manifest); err != nil {
		return nil, false, fmt.Errorf("unmarshal cached directory manifest: %w", err)
	}

	return manifest, true, nil
}

func (s *ManifestState) PutDirectoryManifest(id *types.BlobIdentifier, manifest *pbStorageV1.DirectoryManifest) error {
	data, err := proto.Marshal(manifest)
	if err != nil {
		return fmt.Errorf("marshal directory manifest: %w", err)
	}

	key := prefixKey(id.Bytes(), s.dirPrefix)
	return s.db.Set(key, data, pebble.Sync)
}

func (s *ManifestState) GetFileManifest(id *types.BlobIdentifier) (*pbStorageV1.FileManifest, bool, error) {
	key := prefixKey(id.Bytes(), s.filePrefix)

	data, closer, err := s.db.Get(key)
	if err != nil {
		if err == pebble.ErrNotFound {
			return nil, false, nil
		}
		return nil, false, err
	}
	defer closer.Close()

	// Copy the data since the slice becomes invalid after closer.Close()
	dataCopy := make([]byte, len(data))
	copy(dataCopy, data)

	manifest := &pbStorageV1.FileManifest{}
	if err := proto.Unmarshal(dataCopy, manifest); err != nil {
		return nil, false, fmt.Errorf("unmarshal cached file manifest: %w", err)
	}

	return manifest, true, nil
}

func (s *ManifestState) PutFileManifest(id *types.BlobIdentifier, manifest *pbStorageV1.FileManifest) error {
	data, err := proto.Marshal(manifest)
	if err != nil {
		return fmt.Errorf("marshal file manifest: %w", err)
	}

	key := prefixKey(id.Bytes(), s.filePrefix)
	return s.db.Set(key, data, pebble.Sync)
}
