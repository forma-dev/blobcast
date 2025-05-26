package merkle

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"testing"

	"github.com/forma-dev/blobcast/pkg/crypto"
)

// leafHash hashes a leaf with its 8-byte big-endian index prefix, mirroring the
// logic in CalculateMerkleRoot.
func leafHash(idx int, data []byte) crypto.Hash {
	var pref [8]byte
	binary.LittleEndian.PutUint64(pref[:], uint64(idx))
	return crypto.HashBytes(pref[:], data)
}

// hashPair mirrors CalculateMerkleRoot's internal node hashing.
func hashPair(left, right crypto.Hash) crypto.Hash {
	return crypto.HashBytes(left.Bytes(), right.Bytes())
}

func TestCalculateMerkleRoot_Empty(t *testing.T) {
	if !CalculateMerkleRoot(nil).IsZero() {
		t.Error("expected error for empty leaf slice, got nil")
	}
}

func TestCalculateMerkleRoot_KnownCases(t *testing.T) {
	leaves := map[string][]byte{
		"L1": []byte("L1"),
		"L2": []byte("L2"),
		"L3": []byte("L3"),
		"L4": []byte("L4"),
		"L5": []byte("L5"),
	}

	leafHashes := map[string]crypto.Hash{
		"L1": leafHash(0, leaves["L1"]),
		"L2": leafHash(1, leaves["L2"]),
		"L3": leafHash(2, leaves["L3"]),
		"L4": leafHash(3, leaves["L4"]),
		"L5": leafHash(4, leaves["L5"]),
	}

	tests := []struct {
		name         string
		leaves       [][]byte
		expectedRoot crypto.Hash
	}{
		{
			name:         "single",
			leaves:       [][]byte{leaves["L1"]},
			expectedRoot: leafHashes["L1"],
		},
		{
			name:         "two",
			leaves:       [][]byte{leaves["L1"], leaves["L2"]},
			expectedRoot: hashPair(leafHashes["L1"], leafHashes["L2"]),
		},
		{
			name:   "three",
			leaves: [][]byte{leaves["L1"], leaves["L2"], leaves["L3"]},
			expectedRoot: hashPair(
				hashPair(leafHashes["L1"], leafHashes["L2"]),
				hashPair(leafHashes["L3"], leafHashes["L3"]),
			),
		},
		{
			name:   "four",
			leaves: [][]byte{leaves["L1"], leaves["L2"], leaves["L3"], leaves["L4"]},
			expectedRoot: hashPair(
				hashPair(leafHashes["L1"], leafHashes["L2"]),
				hashPair(leafHashes["L3"], leafHashes["L4"]),
			),
		},
		{
			name:   "five",
			leaves: [][]byte{leaves["L1"], leaves["L2"], leaves["L3"], leaves["L4"], leaves["L5"]},
			expectedRoot: hashPair(
				hashPair(
					hashPair(leafHashes["L1"], leafHashes["L2"]),
					hashPair(leafHashes["L3"], leafHashes["L4"]),
				),
				hashPair(
					hashPair(leafHashes["L5"], leafHashes["L5"]),
					hashPair(leafHashes["L5"], leafHashes["L5"]),
				),
			),
		},
	}

	for _, tc := range tests {
		tc := tc // capture range variable
		t.Run(tc.name, func(t *testing.T) {
			got := CalculateMerkleRoot(tc.leaves)
			if !bytes.Equal(got[:], tc.expectedRoot[:]) {
				t.Errorf("root mismatch.\nwant: %s\n got: %s", tc.expectedRoot, got)
			}
		})
	}
}

func TestCalculateMerkleRoot_OrderSensitivity(t *testing.T) {
	leaves1 := [][]byte{[]byte("A"), []byte("B"), []byte("C")}
	leaves2 := [][]byte{[]byte("C"), []byte("B"), []byte("A")}

	root1 := CalculateMerkleRoot(leaves1)
	root2 := CalculateMerkleRoot(leaves2)

	if bytes.Equal(root1[:], root2[:]) {
		t.Errorf("expected different roots for differently ordered leaves, got equal %s", hex.EncodeToString(root1[:]))
	}
}
