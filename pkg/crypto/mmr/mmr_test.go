package mmr

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"testing"

	"github.com/forma-dev/blobcast/pkg/crypto"
)

func S(s string) []byte {
	return []byte(s)
}

func H(idx uint64, s string) crypto.Hash {
	idxBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(idxBytes, idx-1)
	return crypto.HashBytes(idxBytes, crypto.HashBytes(S(s)).Bytes())
}

func hashPair(idx uint64, left, right crypto.Hash) crypto.Hash {
	idxBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(idxBytes, idx-1)
	return crypto.HashBytes(idxBytes, left.Bytes(), right.Bytes())
}

func hashPeaks(peaks ...crypto.Hash) crypto.Hash {
	peaksBytes := make([][]byte, len(peaks))
	for i, peak := range peaks {
		peaksBytes[i] = peak.Bytes()
	}
	return crypto.HashBytes(peaksBytes...)
}

func hashRoot(size uint64, root crypto.Hash) crypto.Hash {
	sizeBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(sizeBytes, size)
	return crypto.HashBytes(sizeBytes, root.Bytes())
}

func TestMMR_AddLeaf_And_Root(t *testing.T) {
	tests := []struct {
		name              string
		leaves            [][]byte
		expectedRoot      crypto.Hash
		expectedNumLeaves uint64
		expectedSize      uint64
	}{
		{
			name:              "single leaf",
			leaves:            [][]byte{S("L1")},
			expectedRoot:      hashRoot(1, H(1, "L1")),
			expectedNumLeaves: 1,
			expectedSize:      1,
		},
		{
			name:              "two leaves",
			leaves:            [][]byte{S("L1"), S("L2")},
			expectedRoot:      hashRoot(3, hashPair(3, H(1, "L1"), H(2, "L2"))),
			expectedNumLeaves: 2,
			expectedSize:      3,
		},
		{
			name:   "three leaves",
			leaves: [][]byte{S("L1"), S("L2"), S("L3")},
			expectedRoot: hashRoot(4,
				hashPeaks(
					hashPair(3, H(1, "L1"), H(2, "L2")),
					H(4, "L3"),
				),
			),
			expectedNumLeaves: 3,
			expectedSize:      4,
		},
		{
			name:   "four leaves",
			leaves: [][]byte{S("L1"), S("L2"), S("L3"), S("L4")},
			expectedRoot: hashRoot(7,
				hashPair(7,
					hashPair(3, H(1, "L1"), H(2, "L2")),
					hashPair(6, H(4, "L3"), H(5, "L4")),
				),
			),
			expectedNumLeaves: 4,
			expectedSize:      7,
		},
		{
			name:   "five leaves",
			leaves: [][]byte{S("L1"), S("L2"), S("L3"), S("L4"), S("L5")},
			expectedRoot: hashRoot(8,
				hashPeaks(
					hashPair(7,
						hashPair(3, H(1, "L1"), H(2, "L2")),
						hashPair(6, H(4, "L3"), H(5, "L4")),
					),
					H(8, "L5"),
				),
			),
			expectedNumLeaves: 5,
			expectedSize:      8,
		},
		{
			name:   "six leaves",
			leaves: [][]byte{S("L1"), S("L2"), S("L3"), S("L4"), S("L5"), S("L6")},
			expectedRoot: hashRoot(10,
				hashPeaks(
					hashPair(7,
						hashPair(3, H(1, "L1"), H(2, "L2")),
						hashPair(6, H(4, "L3"), H(5, "L4")),
					),
					hashPair(10, H(8, "L5"), H(9, "L6")),
				),
			),
			expectedNumLeaves: 6,
			expectedSize:      10,
		},
		{
			name:   "seven leaves",
			leaves: [][]byte{S("L1"), S("L2"), S("L3"), S("L4"), S("L5"), S("L6"), S("L7")},
			expectedRoot: hashRoot(11,
				hashPeaks(
					hashPair(7,
						hashPair(3, H(1, "L1"), H(2, "L2")),
						hashPair(6, H(4, "L3"), H(5, "L4")),
					),
					hashPeaks(
						hashPair(10, H(8, "L5"), H(9, "L6")),
						H(11, "L7"),
					),
				),
			),
			expectedNumLeaves: 7,
			expectedSize:      11,
		},
		{
			name:   "eight leaves",
			leaves: [][]byte{S("L1"), S("L2"), S("L3"), S("L4"), S("L5"), S("L6"), S("L7"), S("L8")},
			expectedRoot: hashRoot(15,
				hashPair(15,
					hashPair(7,
						hashPair(3, H(1, "L1"), H(2, "L2")),
						hashPair(6, H(4, "L3"), H(5, "L4")),
					),
					hashPair(14,
						hashPair(10, H(8, "L5"), H(9, "L6")),
						hashPair(13, H(11, "L7"), H(12, "L8")),
					),
				),
			),
			expectedNumLeaves: 8,
			expectedSize:      15,
		},
		{
			name:   "nine leaves",
			leaves: [][]byte{S("L1"), S("L2"), S("L3"), S("L4"), S("L5"), S("L6"), S("L7"), S("L8"), S("L9")},
			expectedRoot: hashRoot(16,
				hashPeaks(
					hashPair(15,
						hashPair(7,
							hashPair(3, H(1, "L1"), H(2, "L2")),
							hashPair(6, H(4, "L3"), H(5, "L4")),
						),
						hashPair(14,
							hashPair(10, H(8, "L5"), H(9, "L6")),
							hashPair(13, H(11, "L7"), H(12, "L8")),
						),
					),
					H(16, "L9"),
				),
			),
			expectedNumLeaves: 9,
			expectedSize:      16,
		},
		{
			name:   "ten leaves",
			leaves: [][]byte{S("L1"), S("L2"), S("L3"), S("L4"), S("L5"), S("L6"), S("L7"), S("L8"), S("L9"), S("L10")},
			expectedRoot: hashRoot(18,
				hashPeaks(
					hashPair(15,
						hashPair(7,
							hashPair(3, H(1, "L1"), H(2, "L2")),
							hashPair(6, H(4, "L3"), H(5, "L4")),
						),
						hashPair(14,
							hashPair(10, H(8, "L5"), H(9, "L6")),
							hashPair(13, H(11, "L7"), H(12, "L8")),
						),
					),
					hashPair(18, H(16, "L9"), H(17, "L10")),
				),
			),
			expectedNumLeaves: 10,
			expectedSize:      18,
		},
		{
			name:   "eleven leaves",
			leaves: [][]byte{S("L1"), S("L2"), S("L3"), S("L4"), S("L5"), S("L6"), S("L7"), S("L8"), S("L9"), S("L10"), S("L11")},
			expectedRoot: hashRoot(19,
				hashPeaks(
					hashPair(15,
						hashPair(7,
							hashPair(3, H(1, "L1"), H(2, "L2")),
							hashPair(6, H(4, "L3"), H(5, "L4")),
						),
						hashPair(14,
							hashPair(10, H(8, "L5"), H(9, "L6")),
							hashPair(13, H(11, "L7"), H(12, "L8")),
						),
					),
					hashPeaks(
						hashPair(18, H(16, "L9"), H(17, "L10")),
						H(19, "L11"),
					),
				),
			),
			expectedNumLeaves: 11,
			expectedSize:      19,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mmr := NewMMR()
			var currentRoot crypto.Hash
			for i, leaf := range tc.leaves {
				currentRoot = mmr.AddLeaf(leaf)
				if mmr.numLeaves != uint64(i+1) {
					t.Errorf("Test %s: After adding leaf %d, expected numLeaves %d, got %d", tc.name, i+1, i+1, mmr.numLeaves)
				}
			}

			if !currentRoot.Equal(tc.expectedRoot) {
				t.Errorf("Test %s: AddLeaf expected root %s, got %s", tc.name, hex.EncodeToString(tc.expectedRoot[:]), hex.EncodeToString(currentRoot[:]))
			}

			finalRoot := mmr.Root()
			if !finalRoot.Equal(tc.expectedRoot) {
				t.Errorf("Test %s: Final Root() expected root %s, got %s", tc.name, hex.EncodeToString(tc.expectedRoot[:]), hex.EncodeToString(finalRoot[:]))
			}
			if mmr.size != tc.expectedSize {
				t.Errorf("Test %s: Final size expected %d, got %d", tc.name, tc.expectedSize, mmr.size)
			}
			if mmr.numLeaves != tc.expectedNumLeaves {
				t.Errorf("Test %s: Final numLeaves expected %d, got %d", tc.name, tc.expectedNumLeaves, mmr.numLeaves)
			}
		})
	}
}

func TestMMR_Root_Empty(t *testing.T) {
	mmr := NewMMR()
	root := mmr.Root()
	if !root.IsZero() {
		t.Errorf("Expected zero root for empty MMR, got %s", hex.EncodeToString(root[:]))
	}
}

func TestMMR_AddLeaf_RootConsistency(t *testing.T) {
	mmr := NewMMR()
	leaves := [][]byte{S("A"), S("B"), S("C"), S("D"), S("E")}

	for _, leaf := range leaves {
		rootFromAdd := mmr.AddLeaf(leaf)
		rootFromGet := mmr.Root()
		if !rootFromAdd.Equal(rootFromGet) {
			t.Errorf("Root from AddLeaf (%s) does not match root from Root() (%s) after adding leaf %s",
				hex.EncodeToString(rootFromAdd[:]),
				hex.EncodeToString(rootFromGet[:]),
				hex.EncodeToString(leaf[:]))
		}
	}
}

// TestZero checks the zero helper function
func TestZero(t *testing.T) {
	var z crypto.Hash
	if !z.IsZero() {
		t.Error("Expected zero(Hash{}) to be true")
	}
	var nz crypto.Hash
	nz[0] = 1
	if nz.IsZero() {
		t.Error("Expected zero(Hash{1,...}) to be false")
	}
}

func TestMMR_InclusionProofs(t *testing.T) {
	mmr := NewMMR()

	totalLeaves := uint64(10000)

	proofs := make([]*InclusionProof, totalLeaves)
	leaves := make([][]byte, totalLeaves)

	for i := uint64(0); i < totalLeaves; i++ {
		leaves[i] = S(fmt.Sprintf("leaf%d", i))

		mmr.AddLeaf(leaves[i])

		proof, err := mmr.GenerateInclusionProof(i)
		if err != nil {
			t.Fatalf("Failed to generate inclusion proof for leaf %d: %v", i, err)
		}

		proofs[i] = proof

		expectedHash := mmr.hashWithPosition(leafIndextoMMRIndex(i), crypto.HashBytes(leaves[i]).Bytes())
		if !proof.LeafHash.Equal(expectedHash) {
			t.Errorf("Leaf hash mismatch for leaf %d: got %x, want %x",
				i, proof.LeafHash, expectedHash)
		}

		// Verify the proof
		if !mmr.VerifyInclusionProof(proof) {
			t.Errorf("Inclusion proof verification failed for leaf %d", i)
		}
	}

	mmr.AddLeaf(S("new leaf"))

	// verify all historical proofs are still valid
	for j := uint64(0); j < totalLeaves; j++ {
		if !mmr.VerifyInclusionProof(proofs[j]) {
			t.Errorf("Historical inclusion proof verification failed for leaf %d", j)
		}
	}
}

func TestMMR_InclusionProof_InvalidIndex(t *testing.T) {
	mmr := NewMMR()
	mmr.AddLeaf(S("leaf0"))

	// Try to generate proof for non-existent leaf
	_, err := mmr.GenerateInclusionProof(1)
	if err == nil {
		t.Error("Expected error for out-of-range leaf index")
	}
}

func TestMMR_Invariants(t *testing.T) {
	mmr := NewMMR()
	for i := uint64(0); i < 10000; i++ {
		mmr.AddLeaf(S(fmt.Sprintf("leaf%d", i)))
		if mmr.size != uint64(len(mmr.nodes)) {
			t.Errorf("Expected len(nodes) to be %d, got %d", mmr.size, len(mmr.nodes))
		}

		if mmr.numLeaves != uint64(len(mmr.leafIndexes)) {
			t.Errorf("Expected len(leafIndexes) to be %d, got %d", mmr.numLeaves, len(mmr.leafIndexes))
		}
	}
}

func TestMMR_SnapshotWithNodes(t *testing.T) {
	mmr := NewMMR()

	// Add some leaves
	leaves := [][]byte{S("A"), S("B"), S("C"), S("D"), S("E"), S("F"), S("G"), S("H"), S("I"), S("J")}
	for _, leaf := range leaves {
		mmr.AddLeaf(leaf)
	}

	// Create snapshot
	snapshot := mmr.Snapshot()

	// Restore from snapshot
	mmr2, err := NewMMRFromSnapshot(snapshot)
	if err != nil {
		t.Fatalf("Failed to restore from snapshot: %v", err)
	}

	// Verify restored MMR has same properties
	if mmr2.Size() != mmr.Size() {
		t.Errorf("Size mismatch: got %d, want %d",
			mmr2.Size(), mmr.Size())
	}

	if mmr2.NumPeaks() != mmr.NumPeaks() {
		t.Errorf("NumPeaks mismatch: got %d, want %d",
			mmr2.NumPeaks(), mmr.NumPeaks())
	}

	if mmr2.NumLeaves() != mmr.NumLeaves() {
		t.Errorf("NumLeaves mismatch: got %d, want %d",
			mmr2.NumLeaves(), mmr.NumLeaves())
	}

	if !mmr2.Root().Equal(mmr.Root()) {
		t.Errorf("Root mismatch: got %x, want %x",
			mmr2.Root(), mmr.Root())
	}

	// Verify all nodes are preserved
	for i := uint64(0); i < mmr.Size(); i++ {
		hash1, exists1 := mmr.nodes[i]
		hash2, exists2 := mmr2.nodes[i]

		if exists1 != exists2 {
			t.Errorf("Node existence mismatch at position %d", i)
		}

		if exists1 && !hash1.Equal(hash2) {
			t.Errorf("Node mismatch at position %d: got %x, want %x",
				i, hash2, hash1)
		}
	}

	// Verify all leaf indexes are preserved
	for hash, index := range mmr.leafIndexes {
		index2, exists := mmr2.leafIndexes[hash]
		if !exists {
			t.Errorf("Leaf index mismatch for hash %x", hash)
		}

		if exists && index != index2 {
			t.Errorf("Leaf index mismatch for hash %x: got %d, want %d", hash, index2, index)
		}
	}

	// Verify inclusion proofs work on restored MMR
	for i := uint64(0); i < mmr.NumLeaves(); i++ {
		proof, err := mmr.GenerateInclusionProof(i)
		if err != nil {
			t.Fatalf("Failed to generate proof on restored MMR for leaf %d: %v", i, err)
		}

		if !mmr2.VerifyInclusionProof(proof) {
			t.Errorf("Inclusion proof verification failed on restored MMR for leaf %d", i)
		}
	}
}

func TestMMR_InclusionProof_CorrectSiblings(t *testing.T) {
	tests := []struct {
		name                   string
		numLeaves              int
		proofLeafIndex         uint64
		expectedSiblingIndices []uint64 // MMR positions of expected siblings
	}{
		{
			name:                   "2 leaves - proof for leaf 0",
			numLeaves:              2,
			proofLeafIndex:         0,
			expectedSiblingIndices: []uint64{1}, // Position 1 (leaf 1)
		},
		{
			name:                   "3 leaves - proof for leaf 0",
			numLeaves:              3,
			proofLeafIndex:         0,
			expectedSiblingIndices: []uint64{1}, // Position 1 (leaf 1)
		},
		{
			name:                   "3 leaves - proof for leaf 1",
			numLeaves:              3,
			proofLeafIndex:         1,
			expectedSiblingIndices: []uint64{0}, // Position 0 (leaf 0)
		},
		{
			name:                   "3 leaves - proof for leaf 2",
			numLeaves:              3,
			proofLeafIndex:         2,
			expectedSiblingIndices: []uint64{}, // Leaf 2 is already a peak
		},
		{
			name:                   "4 leaves - proof for leaf 0",
			numLeaves:              4,
			proofLeafIndex:         0,
			expectedSiblingIndices: []uint64{1, 5}, // Position 1 (leaf 1), Position 5 (parent of leaves 2,3)
		},
		{
			name:                   "4 leaves - proof for leaf 3",
			numLeaves:              4,
			proofLeafIndex:         3,
			expectedSiblingIndices: []uint64{3, 2}, // Position 3 (leaf 2), Position 2 (parent of leaves 0,1)
		},
		{
			name:                   "7 leaves - proof for leaf 4",
			numLeaves:              7,
			proofLeafIndex:         4,
			expectedSiblingIndices: []uint64{8}, // Position 8 (leaf 5)
		},
		{
			name:                   "8 leaves - proof for leaf 1",
			numLeaves:              8,
			proofLeafIndex:         1,
			expectedSiblingIndices: []uint64{0, 5, 13}, // leaf 0, parent(2,3), parent(parent(4,5),parent(6,7))
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mmr := NewMMR()

			// Add leaves
			for i := 0; i < tc.numLeaves; i++ {
				mmr.AddLeaf([]byte(fmt.Sprintf("leaf%d", i)))
			}

			// Generate proof
			proof, err := mmr.GenerateInclusionProof(tc.proofLeafIndex)
			if err != nil {
				t.Fatalf("Failed to generate proof: %v", err)
			}

			// Verify we got the expected number of siblings
			if len(proof.Siblings) != len(tc.expectedSiblingIndices) {
				t.Errorf("Expected %d siblings, got %d", len(tc.expectedSiblingIndices), len(proof.Siblings))
			}

			// Verify each sibling matches the expected hash from the MMR
			for i, expectedPos := range tc.expectedSiblingIndices {
				if i >= len(proof.Siblings) {
					t.Errorf("Missing sibling %d (expected at position %d)", i, expectedPos)
					continue
				}

				expectedHash, exists := mmr.nodes[expectedPos]
				if !exists {
					t.Errorf("Expected sibling at position %d doesn't exist in MMR", expectedPos)
					continue
				}

				if !proof.Siblings[i].Equal(expectedHash) {
					t.Errorf("Sibling %d mismatch: got %x, want %x (from position %d)",
						i, proof.Siblings[i][:8], expectedHash[:8], expectedPos)
				}
			}

			// Also verify the proof path is correct by reconstruction
			currentPos := leafIndextoMMRIndex(tc.proofLeafIndex)
			for i := range proof.Siblings {
				parentPos, siblingPos := family(currentPos)

				// Verify this sibling came from the expected position
				if i < len(tc.expectedSiblingIndices) && siblingPos != tc.expectedSiblingIndices[i] {
					t.Errorf("Sibling %d position mismatch: got %d, want %d",
						i, siblingPos, tc.expectedSiblingIndices[i])
				}

				currentPos = parentPos
			}
		})
	}
}
