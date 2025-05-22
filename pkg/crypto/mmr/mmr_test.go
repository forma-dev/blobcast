package mmr

import (
	"encoding/hex"
	"testing"

	"github.com/forma-dev/blobcast/pkg/crypto"
)

func S(s string) []byte {
	return []byte(s)
}

func H(s string) crypto.Hash {
	return crypto.HashBytes(S(s))
}

func hashPair(left, right crypto.Hash) crypto.Hash {
	return crypto.HashBytes(left.Bytes(), right.Bytes())
}

func TestMMR_AddLeaf_And_Root(t *testing.T) {
	tests := []struct {
		name              string
		leaves            [][]byte
		expectedRoot      crypto.Hash
		expectedNumLeaves uint64
	}{
		{
			name:              "single leaf",
			leaves:            [][]byte{S("L1")},
			expectedRoot:      H("L1"),
			expectedNumLeaves: 1,
		},
		{
			name:              "two leaves",
			leaves:            [][]byte{S("L1"), S("L2")},
			expectedRoot:      hashPair(H("L1"), H("L2")),
			expectedNumLeaves: 2,
		},
		{
			name:              "three leaves", // Forms P1=H(L1,L2), P2=L3. Root=H(P1,L3)
			leaves:            [][]byte{S("L1"), S("L2"), S("L3")},
			expectedRoot:      hashPair(hashPair(H("L1"), H("L2")), H("L3")),
			expectedNumLeaves: 3,
		},
		{
			name:              "four leaves", // Forms P1=H(L1,L2), P2=H(L3,L4). Root=H(P1,P2)
			leaves:            [][]byte{S("L1"), S("L2"), S("L3"), S("L4")},
			expectedRoot:      hashPair(hashPair(H("L1"), H("L2")), hashPair(H("L3"), H("L4"))),
			expectedNumLeaves: 4,
		},
		{
			name:   "five leaves", // P1=H(H(L1,L2),H(L3,L4)), P2=L5. Root=H(P1,P2)
			leaves: [][]byte{S("L1"), S("L2"), S("L3"), S("L4"), S("L5")},
			expectedRoot: hashPair(
				hashPair(hashPair(H("L1"), H("L2")), hashPair(H("L3"), H("L4"))),
				H("L5"),
			),
			expectedNumLeaves: 5,
		},
		{
			name:   "seven leaves", // P1=H(H(L1,L2),H(L3,L4)), P2=H(L5,L6), P3=L7. Root=H(P1,H(P2,P3))
			leaves: [][]byte{S("L1"), S("L2"), S("L3"), S("L4"), S("L5"), S("L6"), S("L7")},
			expectedRoot: hashPair(
				hashPair(hashPair(H("L1"), H("L2")), hashPair(H("L3"), H("L4"))),
				hashPair(
					hashPair(H("L5"), H("L6")),
					H("L7"),
				),
			),
			expectedNumLeaves: 7,
		},
		{
			name:   "eight leaves",
			leaves: [][]byte{S("L1"), S("L2"), S("L3"), S("L4"), S("L5"), S("L6"), S("L7"), S("L8")},
			expectedRoot: hashPair(
				hashPair(
					hashPair(H("L1"), H("L2")),
					hashPair(H("L3"), H("L4")),
				),
				hashPair(
					hashPair(H("L5"), H("L6")),
					hashPair(H("L7"), H("L8")),
				),
			),
			expectedNumLeaves: 8,
		},
		{
			name:   "nine leaves",
			leaves: [][]byte{S("L1"), S("L2"), S("L3"), S("L4"), S("L5"), S("L6"), S("L7"), S("L8"), S("L9")},
			expectedRoot: hashPair(
				hashPair(
					hashPair(
						hashPair(H("L1"), H("L2")),
						hashPair(H("L3"), H("L4")),
					),
					hashPair(
						hashPair(H("L5"), H("L6")),
						hashPair(H("L7"), H("L8")),
					),
				),
				H("L9"),
			),
			expectedNumLeaves: 9,
		},
		{
			name:   "ten leaves",
			leaves: [][]byte{S("L1"), S("L2"), S("L3"), S("L4"), S("L5"), S("L6"), S("L7"), S("L8"), S("L9"), S("L10")},
			expectedRoot: hashPair(
				hashPair(
					hashPair(
						hashPair(H("L1"), H("L2")),
						hashPair(H("L3"), H("L4")),
					),
					hashPair(
						hashPair(H("L5"), H("L6")),
						hashPair(H("L7"), H("L8")),
					),
				),
				hashPair(H("L9"), H("L10")),
			),
			expectedNumLeaves: 10,
		},
		{
			name:   "eleven leaves",
			leaves: [][]byte{S("L1"), S("L2"), S("L3"), S("L4"), S("L5"), S("L6"), S("L7"), S("L8"), S("L9"), S("L10"), S("L11")},
			expectedRoot: hashPair(
				hashPair(
					hashPair(
						hashPair(H("L1"), H("L2")),
						hashPair(H("L3"), H("L4")),
					),
					hashPair(
						hashPair(H("L5"), H("L6")),
						hashPair(H("L7"), H("L8")),
					),
				),
				hashPair(
					hashPair(H("L9"), H("L10")),
					H("L11"),
				),
			),
			expectedNumLeaves: 11,
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

			for height, peak := range mmr.peaks {
				t.Logf("Peak (height %d): %s", height, hex.EncodeToString(peak[:]))
			}

			if !currentRoot.Equal(tc.expectedRoot) {
				t.Errorf("Test %s: AddLeaf expected root %s, got %s", tc.name, hex.EncodeToString(tc.expectedRoot[:]), hex.EncodeToString(currentRoot[:]))
			}

			finalRoot := mmr.Root()
			if !finalRoot.Equal(tc.expectedRoot) {
				t.Errorf("Test %s: Final Root() expected root %s, got %s", tc.name, hex.EncodeToString(tc.expectedRoot[:]), hex.EncodeToString(finalRoot[:]))
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
