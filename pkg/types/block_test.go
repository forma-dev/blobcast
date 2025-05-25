package types

import (
	"encoding/hex"
	"testing"
	"time"

	"github.com/forma-dev/blobcast/pkg/crypto"
)

// Helper function to create hash from hex string
func hashFromHex(hexStr string) crypto.Hash {
	bytes, _ := hex.DecodeString(hexStr)
	var hash crypto.Hash
	copy(hash[:], bytes)
	return hash
}

func TestBlockHeaderBytesRoundTrip(t *testing.T) {
	// Create a test block header with known values
	originalHeader := &BlockHeader{
		Version:             1,
		ChainID:             "blobcast-1",
		Height:              12345,
		CelestiaBlockHeight: 67890,
		Timestamp:           time.Unix(1640995200, 0), // 2022-01-01 00:00:00 UTC
		ParentHash:          hashFromHex("0102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f20"),
		DirsRoot:            hashFromHex("2122232425262728292a2b2c2d2e2f303132333435363738393a3b3c3d3e3f40"),
		FilesRoot:           hashFromHex("4142434445464748494a4b4c4d4e4f505152535455565758595a5b5c5d5e5f60"),
		ChunksRoot:          hashFromHex("6162636465666768696a6b6c6d6e6f707172737475767778797a7b7c7d7e7f80"),
		StateRoot:           hashFromHex("8182838485868788898a8b8c8d8e8f909192939495969798999a9b9c9d9e9fa0"),
	}

	// Serialize to bytes
	headerBytes := originalHeader.Bytes()

	// Deserialize back to header
	deserializedHeader := BlockHeaderFromBytes(headerBytes)

	// Compare all fields
	if deserializedHeader.Version != originalHeader.Version {
		t.Errorf("Version mismatch: got %d, want %d", deserializedHeader.Version, originalHeader.Version)
	}

	if deserializedHeader.ChainID != originalHeader.ChainID {
		t.Errorf("ChainID mismatch: got %s, want %s", deserializedHeader.ChainID, originalHeader.ChainID)
	}

	if deserializedHeader.Height != originalHeader.Height {
		t.Errorf("Height mismatch: got %d, want %d", deserializedHeader.Height, originalHeader.Height)
	}

	if deserializedHeader.CelestiaBlockHeight != originalHeader.CelestiaBlockHeight {
		t.Errorf("CelestiaBlockHeight mismatch: got %d, want %d", deserializedHeader.CelestiaBlockHeight, originalHeader.CelestiaBlockHeight)
	}

	if !deserializedHeader.Timestamp.Equal(originalHeader.Timestamp) {
		t.Errorf("Timestamp mismatch: got %v, want %v", deserializedHeader.Timestamp, originalHeader.Timestamp)
	}

	if !deserializedHeader.ParentHash.Equal(originalHeader.ParentHash) {
		t.Errorf("ParentHash mismatch: got %x, want %x", deserializedHeader.ParentHash, originalHeader.ParentHash)
	}

	if !deserializedHeader.DirsRoot.Equal(originalHeader.DirsRoot) {
		t.Errorf("DirsRoot mismatch: got %x, want %x", deserializedHeader.DirsRoot, originalHeader.DirsRoot)
	}

	if !deserializedHeader.FilesRoot.Equal(originalHeader.FilesRoot) {
		t.Errorf("FilesRoot mismatch: got %x, want %x", deserializedHeader.FilesRoot, originalHeader.FilesRoot)
	}

	if !deserializedHeader.ChunksRoot.Equal(originalHeader.ChunksRoot) {
		t.Errorf("ChunksRoot mismatch: got %x, want %x", deserializedHeader.ChunksRoot, originalHeader.ChunksRoot)
	}

	if !deserializedHeader.StateRoot.Equal(originalHeader.StateRoot) {
		t.Errorf("StateRoot mismatch: got %x, want %x", deserializedHeader.StateRoot, originalHeader.StateRoot)
	}
}

func TestBlockHeaderBytesWithDifferentChainIDs(t *testing.T) {
	testCases := []struct {
		name    string
		chainID string
	}{
		{"short chain ID", "test"},
		{"medium chain ID", "blobcast-1"},
		{"long chain ID", "blobcast-mainnet"},
		{"empty chain ID", ""},
		{"single char", "x"},
		{"max reasonable length", "blobcast-testnet-v2"},
	}

	baseHeader := &BlockHeader{
		Version:             1,
		Height:              100,
		CelestiaBlockHeight: 200,
		Timestamp:           time.Unix(1640995200, 0),
		ParentHash:          hashFromHex("0102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f20"),
		DirsRoot:            crypto.Hash{},
		FilesRoot:           crypto.Hash{},
		ChunksRoot:          crypto.Hash{},
		StateRoot:           hashFromHex("704eaff9741fcf70c4a10bbe81c4e19f74e03e8044592e23eb80afd26ddb1a13"),
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			header := *baseHeader
			header.ChainID = tc.chainID

			// Serialize and deserialize
			headerBytes := header.Bytes()
			deserializedHeader := BlockHeaderFromBytes(headerBytes)

			if deserializedHeader.ChainID != tc.chainID {
				t.Errorf("ChainID mismatch for %s: got %s, want %s", tc.name, deserializedHeader.ChainID, tc.chainID)
			}

			// Verify other fields are preserved
			if deserializedHeader.Version != header.Version {
				t.Errorf("Version not preserved for %s", tc.name)
			}
			if deserializedHeader.Height != header.Height {
				t.Errorf("Height not preserved for %s", tc.name)
			}
			if deserializedHeader.CelestiaBlockHeight != header.CelestiaBlockHeight {
				t.Errorf("CelestiaBlockHeight not preserved for %s", tc.name)
			}
			if !deserializedHeader.Timestamp.Equal(header.Timestamp) {
				t.Errorf("Timestamp not preserved for %s", tc.name)
			}
			if !deserializedHeader.ParentHash.Equal(header.ParentHash) {
				t.Errorf("ParentHash not preserved for %s", tc.name)
			}
			if !deserializedHeader.DirsRoot.Equal(header.DirsRoot) {
				t.Errorf("DirsRoot not preserved for %s", tc.name)
			}
			if !deserializedHeader.FilesRoot.Equal(header.FilesRoot) {
				t.Errorf("FilesRoot not preserved for %s", tc.name)
			}
			if !deserializedHeader.ChunksRoot.Equal(header.ChunksRoot) {
				t.Errorf("ChunksRoot not preserved for %s", tc.name)
			}
			if !deserializedHeader.StateRoot.Equal(header.StateRoot) {
				t.Errorf("StateRoot not preserved for %s", tc.name)
			}
		})
	}
}

func TestBlockHeaderBytesWithZeroHashes(t *testing.T) {
	// Test with all zero hashes (common for genesis blocks)
	header := &BlockHeader{
		Version:             1,
		ChainID:             "blobcast-1",
		Height:              0,
		CelestiaBlockHeight: 0,
		Timestamp:           time.Unix(0, 0),
		ParentHash:          crypto.Hash{}, // All zeros
		DirsRoot:            crypto.Hash{}, // All zeros
		FilesRoot:           crypto.Hash{}, // All zeros
		ChunksRoot:          crypto.Hash{}, // All zeros
		StateRoot:           crypto.Hash{}, // All zeros
	}

	headerBytes := header.Bytes()
	deserializedHeader := BlockHeaderFromBytes(headerBytes)

	// Verify zero hashes are preserved
	if !deserializedHeader.ParentHash.IsZero() {
		t.Errorf("ParentHash should be zero but got %x", deserializedHeader.ParentHash)
	}
	if !deserializedHeader.DirsRoot.IsZero() {
		t.Errorf("DirsRoot should be zero but got %x", deserializedHeader.DirsRoot)
	}
	if !deserializedHeader.FilesRoot.IsZero() {
		t.Errorf("FilesRoot should be zero but got %x", deserializedHeader.FilesRoot)
	}
	if !deserializedHeader.ChunksRoot.IsZero() {
		t.Errorf("ChunksRoot should be zero but got %x", deserializedHeader.ChunksRoot)
	}
	if !deserializedHeader.StateRoot.IsZero() {
		t.Errorf("StateRoot should be zero but got %x", deserializedHeader.StateRoot)
	}
}

func TestBlockHeaderBytesWithMaxValues(t *testing.T) {
	// Test with maximum values
	maxHash := hashFromHex("ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff")

	header := &BlockHeader{
		Version:             0xffffffff,
		ChainID:             "max-test-chain-id",
		Height:              0xffffffffffffffff,
		CelestiaBlockHeight: 0xffffffffffffffff,
		Timestamp:           time.Unix(0x7fffffffffffffff, 0), // Max int64
		ParentHash:          maxHash,
		DirsRoot:            maxHash,
		FilesRoot:           maxHash,
		ChunksRoot:          maxHash,
		StateRoot:           maxHash,
	}

	headerBytes := header.Bytes()
	deserializedHeader := BlockHeaderFromBytes(headerBytes)

	if deserializedHeader.Version != header.Version {
		t.Errorf("Version mismatch with max values: got %d, want %d", deserializedHeader.Version, header.Version)
	}
	if deserializedHeader.Height != header.Height {
		t.Errorf("Height mismatch with max values: got %d, want %d", deserializedHeader.Height, header.Height)
	}
	if deserializedHeader.CelestiaBlockHeight != header.CelestiaBlockHeight {
		t.Errorf("CelestiaBlockHeight mismatch with max values: got %d, want %d", deserializedHeader.CelestiaBlockHeight, header.CelestiaBlockHeight)
	}
}

func TestBlockHeaderHashConsistency(t *testing.T) {
	// Test that the hash is consistent across serialization/deserialization
	header := &BlockHeader{
		Version:             1,
		ChainID:             "blobcast-1",
		Height:              12345,
		CelestiaBlockHeight: 67890,
		Timestamp:           time.Unix(1640995200, 0),
		ParentHash:          hashFromHex("0102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f20"),
		DirsRoot:            hashFromHex("2122232425262728292a2b2c2d2e2f303132333435363738393a3b3c3d3e3f40"),
		FilesRoot:           hashFromHex("4142434445464748494a4b4c4d4e4f505152535455565758595a5b5c5d5e5f60"),
		ChunksRoot:          hashFromHex("6162636465666768696a6b6c6d6e6f707172737475767778797a7b7c7d7e7f80"),
		StateRoot:           hashFromHex("8182838485868788898a8b8c8d8e8f909192939495969798999a9b9c9d9e9fa0"),
	}

	// Get original hash
	originalHash := header.Hash()

	// Serialize and deserialize
	headerBytes := header.Bytes()
	deserializedHeader := BlockHeaderFromBytes(headerBytes)

	// Get hash of deserialized header
	deserializedHash := deserializedHeader.Hash()

	// Hashes should be identical
	if !originalHash.Equal(deserializedHash) {
		t.Errorf("Hash mismatch after serialization/deserialization: got %x, want %x", deserializedHash, originalHash)
	}
}

func TestBlockHeaderBytesLength(t *testing.T) {
	// Test that serialized bytes have expected length
	header := &BlockHeader{
		Version:             1,
		ChainID:             "blobcast-1",
		Height:              12345,
		CelestiaBlockHeight: 67890,
		Timestamp:           time.Unix(1640995200, 0),
		ParentHash:          crypto.Hash{},
		DirsRoot:            crypto.Hash{},
		FilesRoot:           crypto.Hash{},
		ChunksRoot:          crypto.Hash{},
		StateRoot:           crypto.Hash{},
	}

	headerBytes := header.Bytes()

	// Expected length calculation:
	// Version: 4 bytes
	// ChainID length: 1 byte + len("blobcast-1") = 1 + 10 = 11 bytes
	// Height: 8 bytes
	// CelestiaBlockHeight: 8 bytes
	// Timestamp: 8 bytes
	// ParentHash: 32 bytes
	// DirsRoot: 32 bytes
	// FilesRoot: 32 bytes
	// ChunksRoot: 32 bytes
	// StateRoot: 32 bytes
	// Total: 4 + 11 + 8 + 8 + 8 + 32 + 32 + 32 + 32 + 32 = 199 bytes

	expectedLength := 4 + 1 + len("blobcast-1") + 8 + 8 + 8 + 32 + 32 + 32 + 32 + 32
	if len(headerBytes) != expectedLength {
		t.Errorf("Unexpected header bytes length: got %d, want %d", len(headerBytes), expectedLength)
	}
}

// Helper function to create test BlobIdentifier
func createTestBlobIdentifier(height uint64, hashSuffix byte) *BlobIdentifier {
	hash := crypto.Hash{}
	for i := range hash {
		hash[i] = hashSuffix
	}
	return &BlobIdentifier{
		Height:     height,
		Commitment: hash,
	}
}

func TestBlockBodyBytesRoundTrip(t *testing.T) {
	// Create test block body with various blob identifiers
	originalBody := &BlockBody{
		Dirs: []*BlobIdentifier{
			createTestBlobIdentifier(100, 0x01),
			createTestBlobIdentifier(101, 0x02),
		},
		Files: []*BlobIdentifier{
			createTestBlobIdentifier(200, 0x03),
			createTestBlobIdentifier(201, 0x04),
			createTestBlobIdentifier(202, 0x05),
		},
		Chunks: []*BlobIdentifier{
			createTestBlobIdentifier(300, 0x06),
		},
	}

	// Serialize to bytes
	bodyBytes := originalBody.Bytes()

	// Deserialize back to body
	deserializedBody := BlockBodyFromBytes(bodyBytes)

	// Compare dirs
	if len(deserializedBody.Dirs) != len(originalBody.Dirs) {
		t.Errorf("Dirs count mismatch: got %d, want %d", len(deserializedBody.Dirs), len(originalBody.Dirs))
	}
	for i, dir := range originalBody.Dirs {
		if deserializedBody.Dirs[i].Height != dir.Height {
			t.Errorf("Dir[%d] height mismatch: got %d, want %d", i, deserializedBody.Dirs[i].Height, dir.Height)
		}
		if !deserializedBody.Dirs[i].Commitment.Equal(dir.Commitment) {
			t.Errorf("Dir[%d] commitment mismatch: got %x, want %x", i, deserializedBody.Dirs[i].Commitment, dir.Commitment)
		}
	}

	// Compare files
	if len(deserializedBody.Files) != len(originalBody.Files) {
		t.Errorf("Files count mismatch: got %d, want %d", len(deserializedBody.Files), len(originalBody.Files))
	}
	for i, file := range originalBody.Files {
		if deserializedBody.Files[i].Height != file.Height {
			t.Errorf("File[%d] height mismatch: got %d, want %d", i, deserializedBody.Files[i].Height, file.Height)
		}
		if !deserializedBody.Files[i].Commitment.Equal(file.Commitment) {
			t.Errorf("File[%d] commitment mismatch: got %x, want %x", i, deserializedBody.Files[i].Commitment, file.Commitment)
		}
	}

	// Compare chunks
	if len(deserializedBody.Chunks) != len(originalBody.Chunks) {
		t.Errorf("Chunks count mismatch: got %d, want %d", len(deserializedBody.Chunks), len(originalBody.Chunks))
	}
	for i, chunk := range originalBody.Chunks {
		if deserializedBody.Chunks[i].Height != chunk.Height {
			t.Errorf("Chunk[%d] height mismatch: got %d, want %d", i, deserializedBody.Chunks[i].Height, chunk.Height)
		}
		if !deserializedBody.Chunks[i].Commitment.Equal(chunk.Commitment) {
			t.Errorf("Chunk[%d] commitment mismatch: got %x, want %x", i, deserializedBody.Chunks[i].Commitment, chunk.Commitment)
		}
	}
}

func TestBlockBodyBytesEmpty(t *testing.T) {
	// Test with empty block body
	emptyBody := &BlockBody{
		Dirs:   []*BlobIdentifier{},
		Files:  []*BlobIdentifier{},
		Chunks: []*BlobIdentifier{},
	}

	bodyBytes := emptyBody.Bytes()
	deserializedBody := BlockBodyFromBytes(bodyBytes)

	if len(deserializedBody.Dirs) != 0 {
		t.Errorf("Expected empty dirs, got %d", len(deserializedBody.Dirs))
	}
	if len(deserializedBody.Files) != 0 {
		t.Errorf("Expected empty files, got %d", len(deserializedBody.Files))
	}
	if len(deserializedBody.Chunks) != 0 {
		t.Errorf("Expected empty chunks, got %d", len(deserializedBody.Chunks))
	}
}

func TestBlockBodyBytesNilSlices(t *testing.T) {
	// Test with nil slices
	nilBody := &BlockBody{
		Dirs:   nil,
		Files:  nil,
		Chunks: nil,
	}

	bodyBytes := nilBody.Bytes()
	deserializedBody := BlockBodyFromBytes(bodyBytes)

	if len(deserializedBody.Dirs) != 0 {
		t.Errorf("Expected empty dirs from nil, got %d", len(deserializedBody.Dirs))
	}
	if len(deserializedBody.Files) != 0 {
		t.Errorf("Expected empty files from nil, got %d", len(deserializedBody.Files))
	}
	if len(deserializedBody.Chunks) != 0 {
		t.Errorf("Expected empty chunks from nil, got %d", len(deserializedBody.Chunks))
	}
}

func TestBlockBodyBytesLargeData(t *testing.T) {
	// Test with larger amounts of data
	dirs := make([]*BlobIdentifier, 100)
	files := make([]*BlobIdentifier, 200)
	chunks := make([]*BlobIdentifier, 300)

	for i := range dirs {
		dirs[i] = createTestBlobIdentifier(uint64(i), byte(i%256))
	}
	for i := range files {
		files[i] = createTestBlobIdentifier(uint64(i+1000), byte((i+100)%256))
	}
	for i := range chunks {
		chunks[i] = createTestBlobIdentifier(uint64(i+2000), byte((i+200)%256))
	}

	largeBody := &BlockBody{
		Dirs:   dirs,
		Files:  files,
		Chunks: chunks,
	}

	bodyBytes := largeBody.Bytes()
	deserializedBody := BlockBodyFromBytes(bodyBytes)

	// Verify counts
	if len(deserializedBody.Dirs) != 100 {
		t.Errorf("Dirs count mismatch: got %d, want 100", len(deserializedBody.Dirs))
	}
	if len(deserializedBody.Files) != 200 {
		t.Errorf("Files count mismatch: got %d, want 200", len(deserializedBody.Files))
	}
	if len(deserializedBody.Chunks) != 300 {
		t.Errorf("Chunks count mismatch: got %d, want 300", len(deserializedBody.Chunks))
	}

	// Spot check a few items
	if deserializedBody.Dirs[50].Height != 50 {
		t.Errorf("Dir[50] height mismatch: got %d, want 50", deserializedBody.Dirs[50].Height)
	}
	if deserializedBody.Files[100].Height != 1100 {
		t.Errorf("File[100] height mismatch: got %d, want 1100", deserializedBody.Files[100].Height)
	}
	if deserializedBody.Chunks[200].Height != 2200 {
		t.Errorf("Chunk[200] height mismatch: got %d, want 2200", deserializedBody.Chunks[200].Height)
	}
}

func TestBlockBodyBytesLength(t *testing.T) {
	// Test that serialized bytes have expected length
	body := &BlockBody{
		Dirs: []*BlobIdentifier{
			createTestBlobIdentifier(100, 0x01),
			createTestBlobIdentifier(101, 0x02),
		},
		Files: []*BlobIdentifier{
			createTestBlobIdentifier(200, 0x03),
		},
		Chunks: []*BlobIdentifier{
			createTestBlobIdentifier(300, 0x04),
			createTestBlobIdentifier(301, 0x05),
			createTestBlobIdentifier(302, 0x06),
		},
	}

	bodyBytes := body.Bytes()

	// Expected length calculation:
	// Header: 3 * 4 bytes (counts) = 12 bytes
	// Dirs: 2 * 40 bytes = 80 bytes
	// Files: 1 * 40 bytes = 40 bytes
	// Chunks: 3 * 40 bytes = 120 bytes
	// Total: 12 + 80 + 40 + 120 = 252 bytes

	expectedLength := 12 + (2+1+3)*40
	if len(bodyBytes) != expectedLength {
		t.Errorf("Unexpected body bytes length: got %d, want %d", len(bodyBytes), expectedLength)
	}
}

func TestBlockBodyBytesOnlyDirs(t *testing.T) {
	// Test with only dirs populated
	body := &BlockBody{
		Dirs: []*BlobIdentifier{
			createTestBlobIdentifier(100, 0x01),
			createTestBlobIdentifier(101, 0x02),
		},
		Files:  []*BlobIdentifier{},
		Chunks: []*BlobIdentifier{},
	}

	bodyBytes := body.Bytes()
	deserializedBody := BlockBodyFromBytes(bodyBytes)

	if len(deserializedBody.Dirs) != 2 {
		t.Errorf("Dirs count mismatch: got %d, want 2", len(deserializedBody.Dirs))
	}
	if len(deserializedBody.Files) != 0 {
		t.Errorf("Files count mismatch: got %d, want 0", len(deserializedBody.Files))
	}
	if len(deserializedBody.Chunks) != 0 {
		t.Errorf("Chunks count mismatch: got %d, want 0", len(deserializedBody.Chunks))
	}
}

func TestBlockBodyBytesOnlyFiles(t *testing.T) {
	// Test with only files populated
	body := &BlockBody{
		Dirs: []*BlobIdentifier{},
		Files: []*BlobIdentifier{
			createTestBlobIdentifier(200, 0x03),
		},
		Chunks: []*BlobIdentifier{},
	}

	bodyBytes := body.Bytes()
	deserializedBody := BlockBodyFromBytes(bodyBytes)

	if len(deserializedBody.Dirs) != 0 {
		t.Errorf("Dirs count mismatch: got %d, want 0", len(deserializedBody.Dirs))
	}
	if len(deserializedBody.Files) != 1 {
		t.Errorf("Files count mismatch: got %d, want 1", len(deserializedBody.Files))
	}
	if len(deserializedBody.Chunks) != 0 {
		t.Errorf("Chunks count mismatch: got %d, want 0", len(deserializedBody.Chunks))
	}
}

func TestBlockBodyBytesOnlyChunks(t *testing.T) {
	// Test with only chunks populated
	body := &BlockBody{
		Dirs:  []*BlobIdentifier{},
		Files: []*BlobIdentifier{},
		Chunks: []*BlobIdentifier{
			createTestBlobIdentifier(300, 0x04),
			createTestBlobIdentifier(301, 0x05),
		},
	}

	bodyBytes := body.Bytes()
	deserializedBody := BlockBodyFromBytes(bodyBytes)

	if len(deserializedBody.Dirs) != 0 {
		t.Errorf("Dirs count mismatch: got %d, want 0", len(deserializedBody.Dirs))
	}
	if len(deserializedBody.Files) != 0 {
		t.Errorf("Files count mismatch: got %d, want 0", len(deserializedBody.Files))
	}
	if len(deserializedBody.Chunks) != 2 {
		t.Errorf("Chunks count mismatch: got %d, want 2", len(deserializedBody.Chunks))
	}
}

func TestBlockBytesRoundTrip(t *testing.T) {
	// Create a complete test block
	originalBlock := &Block{
		Header: &BlockHeader{
			Version:             1,
			ChainID:             "blobcast-1",
			Height:              12345,
			CelestiaBlockHeight: 67890,
			Timestamp:           time.Unix(1640995200, 0),
			ParentHash:          hashFromHex("0102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f20"),
			DirsRoot:            hashFromHex("2122232425262728292a2b2c2d2e2f303132333435363738393a3b3c3d3e3f40"),
			FilesRoot:           hashFromHex("4142434445464748494a4b4c4d4e4f505152535455565758595a5b5c5d5e5f60"),
			ChunksRoot:          hashFromHex("6162636465666768696a6b6c6d6e6f707172737475767778797a7b7c7d7e7f80"),
			StateRoot:           hashFromHex("8182838485868788898a8b8c8d8e8f909192939495969798999a9b9c9d9e9fa0"),
		},
		Body: &BlockBody{
			Dirs: []*BlobIdentifier{
				createTestBlobIdentifier(100, 0x01),
				createTestBlobIdentifier(101, 0x02),
			},
			Files: []*BlobIdentifier{
				createTestBlobIdentifier(200, 0x03),
			},
			Chunks: []*BlobIdentifier{
				createTestBlobIdentifier(300, 0x04),
				createTestBlobIdentifier(301, 0x05),
			},
		},
	}

	// Serialize to bytes
	blockBytes := originalBlock.Bytes()

	// Deserialize back to block
	deserializedBlock := BlockFromBytes(blockBytes)

	// Compare header fields
	if deserializedBlock.Header.Version != originalBlock.Header.Version {
		t.Errorf("Header Version mismatch: got %d, want %d", deserializedBlock.Header.Version, originalBlock.Header.Version)
	}
	if deserializedBlock.Header.ChainID != originalBlock.Header.ChainID {
		t.Errorf("Header ChainID mismatch: got %s, want %s", deserializedBlock.Header.ChainID, originalBlock.Header.ChainID)
	}
	if deserializedBlock.Header.Height != originalBlock.Header.Height {
		t.Errorf("Header Height mismatch: got %d, want %d", deserializedBlock.Header.Height, originalBlock.Header.Height)
	}
	if !deserializedBlock.Header.ParentHash.Equal(originalBlock.Header.ParentHash) {
		t.Errorf("Header ParentHash mismatch: got %x, want %x", deserializedBlock.Header.ParentHash, originalBlock.Header.ParentHash)
	}

	// Compare body counts
	if len(deserializedBlock.Body.Dirs) != len(originalBlock.Body.Dirs) {
		t.Errorf("Body Dirs count mismatch: got %d, want %d", len(deserializedBlock.Body.Dirs), len(originalBlock.Body.Dirs))
	}
	if len(deserializedBlock.Body.Files) != len(originalBlock.Body.Files) {
		t.Errorf("Body Files count mismatch: got %d, want %d", len(deserializedBlock.Body.Files), len(originalBlock.Body.Files))
	}
	if len(deserializedBlock.Body.Chunks) != len(originalBlock.Body.Chunks) {
		t.Errorf("Body Chunks count mismatch: got %d, want %d", len(deserializedBlock.Body.Chunks), len(originalBlock.Body.Chunks))
	}

	// Spot check some body items
	if len(deserializedBlock.Body.Dirs) > 0 && deserializedBlock.Body.Dirs[0].Height != 100 {
		t.Errorf("First dir height mismatch: got %d, want 100", deserializedBlock.Body.Dirs[0].Height)
	}
	if len(deserializedBlock.Body.Files) > 0 && deserializedBlock.Body.Files[0].Height != 200 {
		t.Errorf("First file height mismatch: got %d, want 200", deserializedBlock.Body.Files[0].Height)
	}
}

func TestBlockBytesEmpty(t *testing.T) {
	// Test with empty block (genesis-like)
	emptyBlock := &Block{
		Header: &BlockHeader{
			Version:             1,
			ChainID:             "blobcast-1",
			Height:              0,
			CelestiaBlockHeight: 0,
			Timestamp:           time.Unix(0, 0),
			ParentHash:          crypto.Hash{},
			DirsRoot:            crypto.Hash{},
			FilesRoot:           crypto.Hash{},
			ChunksRoot:          crypto.Hash{},
			StateRoot:           crypto.Hash{},
		},
		Body: &BlockBody{
			Dirs:   []*BlobIdentifier{},
			Files:  []*BlobIdentifier{},
			Chunks: []*BlobIdentifier{},
		},
	}

	blockBytes := emptyBlock.Bytes()
	deserializedBlock := BlockFromBytes(blockBytes)

	// Verify header
	if deserializedBlock.Header.Height != 0 {
		t.Errorf("Expected genesis height 0, got %d", deserializedBlock.Header.Height)
	}
	if deserializedBlock.Header.ChainID != "blobcast-1" {
		t.Errorf("ChainID mismatch: got %s, want blobcast-1", deserializedBlock.Header.ChainID)
	}

	// Verify empty body
	if len(deserializedBlock.Body.Dirs) != 0 {
		t.Errorf("Expected empty dirs, got %d", len(deserializedBlock.Body.Dirs))
	}
	if len(deserializedBlock.Body.Files) != 0 {
		t.Errorf("Expected empty files, got %d", len(deserializedBlock.Body.Files))
	}
	if len(deserializedBlock.Body.Chunks) != 0 {
		t.Errorf("Expected empty chunks, got %d", len(deserializedBlock.Body.Chunks))
	}
}

func TestBlockBytesWithDifferentChainIDs(t *testing.T) {
	testCases := []struct {
		name    string
		chainID string
	}{
		{"short chain ID", "test"},
		{"medium chain ID", "blobcast-1"},
		{"long chain ID", "blobcast-mainnet"},
		{"empty chain ID", ""},
		{"single char", "x"},
	}

	baseBlock := &Block{
		Header: &BlockHeader{
			Version:             1,
			Height:              100,
			CelestiaBlockHeight: 200,
			Timestamp:           time.Unix(1640995200, 0),
			ParentHash:          hashFromHex("0102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f20"),
			DirsRoot:            crypto.Hash{},
			FilesRoot:           crypto.Hash{},
			ChunksRoot:          crypto.Hash{},
			StateRoot:           crypto.Hash{},
		},
		Body: &BlockBody{
			Dirs:   []*BlobIdentifier{createTestBlobIdentifier(100, 0x01)},
			Files:  []*BlobIdentifier{},
			Chunks: []*BlobIdentifier{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			block := *baseBlock
			block.Header = &BlockHeader{
				Version:             baseBlock.Header.Version,
				ChainID:             tc.chainID, // Use test case chain ID
				Height:              baseBlock.Header.Height,
				CelestiaBlockHeight: baseBlock.Header.CelestiaBlockHeight,
				Timestamp:           baseBlock.Header.Timestamp,
				ParentHash:          baseBlock.Header.ParentHash,
				DirsRoot:            baseBlock.Header.DirsRoot,
				FilesRoot:           baseBlock.Header.FilesRoot,
				ChunksRoot:          baseBlock.Header.ChunksRoot,
				StateRoot:           baseBlock.Header.StateRoot,
			}

			blockBytes := block.Bytes()
			deserializedBlock := BlockFromBytes(blockBytes)

			if deserializedBlock.Header.ChainID != tc.chainID {
				t.Errorf("ChainID mismatch for %s: got %s, want %s", tc.name, deserializedBlock.Header.ChainID, tc.chainID)
			}

			// Verify other fields are preserved
			if deserializedBlock.Header.Height != block.Header.Height {
				t.Errorf("Height not preserved for %s", tc.name)
			}
			if len(deserializedBlock.Body.Dirs) != len(block.Body.Dirs) {
				t.Errorf("Dirs count not preserved for %s", tc.name)
			}
		})
	}
}

func TestBlockBytesLength(t *testing.T) {
	// Test that serialized bytes have expected length
	block := &Block{
		Header: &BlockHeader{
			Version:             1,
			ChainID:             "blobcast-1",
			Height:              12345,
			CelestiaBlockHeight: 67890,
			Timestamp:           time.Unix(1640995200, 0),
			ParentHash:          crypto.Hash{},
			DirsRoot:            crypto.Hash{},
			FilesRoot:           crypto.Hash{},
			ChunksRoot:          crypto.Hash{},
			StateRoot:           crypto.Hash{},
		},
		Body: &BlockBody{
			Dirs: []*BlobIdentifier{
				createTestBlobIdentifier(100, 0x01),
			},
			Files:  []*BlobIdentifier{},
			Chunks: []*BlobIdentifier{},
		},
	}

	blockBytes := block.Bytes()

	// Expected length calculation:
	// Header length prefix: 4 bytes
	// Header: 4 + 1 + 10 + 8 + 8 + 8 + 32*5 = 199 bytes
	// Body: 12 + 1*40 = 52 bytes
	// Total: 4 + 199 + 52 = 255 bytes

	headerLength := 4 + 1 + len("blobcast-1") + 8 + 8 + 8 + 32*5
	bodyLength := 12 + 1*40
	expectedLength := 4 + headerLength + bodyLength

	if len(blockBytes) != expectedLength {
		t.Errorf("Unexpected block bytes length: got %d, want %d", len(blockBytes), expectedLength)
	}
}

func TestBlockBytesLargeData(t *testing.T) {
	// Test with a block containing lots of data
	dirs := make([]*BlobIdentifier, 50)
	files := make([]*BlobIdentifier, 100)
	chunks := make([]*BlobIdentifier, 150)

	for i := range dirs {
		dirs[i] = createTestBlobIdentifier(uint64(i), byte(i%256))
	}
	for i := range files {
		files[i] = createTestBlobIdentifier(uint64(i+1000), byte((i+100)%256))
	}
	for i := range chunks {
		chunks[i] = createTestBlobIdentifier(uint64(i+2000), byte((i+200)%256))
	}

	largeBlock := &Block{
		Header: &BlockHeader{
			Version:             1,
			ChainID:             "blobcast-1",
			Height:              99999,
			CelestiaBlockHeight: 88888,
			Timestamp:           time.Unix(1640995200, 0),
			ParentHash:          hashFromHex("0102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f20"),
			DirsRoot:            hashFromHex("2122232425262728292a2b2c2d2e2f303132333435363738393a3b3c3d3e3f40"),
			FilesRoot:           hashFromHex("4142434445464748494a4b4c4d4e4f505152535455565758595a5b5c5d5e5f60"),
			ChunksRoot:          hashFromHex("6162636465666768696a6b6c6d6e6f707172737475767778797a7b7c7d7e7f80"),
			StateRoot:           hashFromHex("8182838485868788898a8b8c8d8e8f909192939495969798999a9b9c9d9e9fa0"),
		},
		Body: &BlockBody{
			Dirs:   dirs,
			Files:  files,
			Chunks: chunks,
		},
	}

	blockBytes := largeBlock.Bytes()
	deserializedBlock := BlockFromBytes(blockBytes)

	// Verify counts
	if len(deserializedBlock.Body.Dirs) != 50 {
		t.Errorf("Dirs count mismatch: got %d, want 50", len(deserializedBlock.Body.Dirs))
	}
	if len(deserializedBlock.Body.Files) != 100 {
		t.Errorf("Files count mismatch: got %d, want 100", len(deserializedBlock.Body.Files))
	}
	if len(deserializedBlock.Body.Chunks) != 150 {
		t.Errorf("Chunks count mismatch: got %d, want 150", len(deserializedBlock.Body.Chunks))
	}

	// Verify header is preserved
	if deserializedBlock.Header.Height != 99999 {
		t.Errorf("Header height mismatch: got %d, want 99999", deserializedBlock.Header.Height)
	}
	if deserializedBlock.Header.CelestiaBlockHeight != 88888 {
		t.Errorf("Header celestia height mismatch: got %d, want 88888", deserializedBlock.Header.CelestiaBlockHeight)
	}
}

func TestBlockHashConsistency(t *testing.T) {
	// Test that block hash is consistent across serialization/deserialization
	block := &Block{
		Header: &BlockHeader{
			Version:             1,
			ChainID:             "blobcast-1",
			Height:              12345,
			CelestiaBlockHeight: 67890,
			Timestamp:           time.Unix(1640995200, 0),
			ParentHash:          hashFromHex("0102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f20"),
			DirsRoot:            hashFromHex("2122232425262728292a2b2c2d2e2f303132333435363738393a3b3c3d3e3f40"),
			FilesRoot:           hashFromHex("4142434445464748494a4b4c4d4e4f505152535455565758595a5b5c5d5e5f60"),
			ChunksRoot:          hashFromHex("6162636465666768696a6b6c6d6e6f707172737475767778797a7b7c7d7e7f80"),
			StateRoot:           hashFromHex("8182838485868788898a8b8c8d8e8f909192939495969798999a9b9c9d9e9fa0"),
		},
		Body: &BlockBody{
			Dirs:   []*BlobIdentifier{createTestBlobIdentifier(100, 0x01)},
			Files:  []*BlobIdentifier{createTestBlobIdentifier(200, 0x02)},
			Chunks: []*BlobIdentifier{createTestBlobIdentifier(300, 0x03)},
		},
	}

	// Get original hash
	originalHash := block.Hash()

	// Serialize and deserialize
	blockBytes := block.Bytes()
	deserializedBlock := BlockFromBytes(blockBytes)

	// Get hash of deserialized block
	deserializedHash := deserializedBlock.Hash()

	// Hashes should be identical (since hash is based on header only)
	if !originalHash.Equal(deserializedHash) {
		t.Errorf("Hash mismatch after serialization/deserialization: got %x, want %x", deserializedHash, originalHash)
	}
}
