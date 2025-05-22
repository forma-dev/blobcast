package state

import (
	"encoding/json"
	"errors"
	"sync"

	"github.com/cockroachdb/pebble"
)

var (
	uploadState      *UploadState
	uploadStateMutex sync.Mutex
)

type UploadRecordKey [32]byte

type UploadRecord struct {
	Key        UploadRecordKey `json:"key"`
	Completed  bool            `json:"completed"`
	ManifestID string          `json:"manifest_id,omitempty"`
}

type UploadState struct {
	db *pebble.DB
}

func GetUploadState() (*UploadState, error) {
	uploadStateMutex.Lock()
	defer uploadStateMutex.Unlock()

	if uploadState != nil {
		return uploadState, nil
	}

	uploadStateLocal, err := openUploadState()
	if err != nil {
		return nil, err
	}

	uploadState = uploadStateLocal
	return uploadState, nil
}

func openUploadState() (*UploadState, error) {
	dbPath, err := getStateDbPath(UploadStateDb)
	if err != nil {
		return nil, err
	}

	db, err := pebble.Open(dbPath, nil)
	if err != nil {
		return nil, err
	}

	return &UploadState{
		db: db,
	}, nil
}

func (s *UploadState) GetUploadRecord(key UploadRecordKey) (*UploadRecord, error) {
	data, closer, err := s.db.Get(key[:])
	if err != nil {
		if errors.Is(err, pebble.ErrNotFound) {
			return &UploadRecord{
				Key: key,
			}, nil
		}
		return nil, err
	}
	defer closer.Close()

	var uploadRecord UploadRecord
	if err := json.Unmarshal(data, &uploadRecord); err != nil {
		return nil, err
	}

	return &uploadRecord, nil
}

func (s *UploadState) SaveUploadRecord(uploadRecord *UploadRecord) error {
	key := uploadRecord.Key[:]
	data, err := json.Marshal(uploadRecord)
	if err != nil {
		return err
	}

	return s.db.Set(key, data, nil)
}

func (s *UploadState) DeleteUploadRecord(key UploadRecordKey) error {
	return s.db.Delete(key[:], nil)
}
