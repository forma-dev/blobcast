package merkle

import (
	"encoding/binary"

	"github.com/forma-dev/blobcast/pkg/crypto"
)

type MerkleNode struct {
	Left  *MerkleNode
	Right *MerkleNode
	Hash  crypto.Hash
}

func CalculateMerkleRoot(leaves [][]byte) crypto.Hash {
	if len(leaves) == 0 {
		return crypto.Hash{}
	}

	// hash leaves with a length-prefixed index to avoid ambiguity
	nodes := make([]*MerkleNode, len(leaves))
	for i, leaf := range leaves {
		var idx [8]byte
		binary.BigEndian.PutUint64(idx[:], uint64(i))
		nodes[i] = &MerkleNode{
			Hash: crypto.HashBytes(idx[:], leaf),
		}
	}

	for len(nodes) > 1 {
		var newNodes []*MerkleNode
		for i := 0; i < len(nodes); i += 2 {
			left := nodes[i]
			right := left
			if i+1 < len(nodes) {
				right = nodes[i+1]
			}
			pairHash := crypto.HashBytes(left.Hash.Bytes(), right.Hash.Bytes())
			newNodes = append(newNodes, &MerkleNode{Left: left, Right: right, Hash: pairHash})
		}
		nodes = newNodes
	}

	return nodes[0].Hash
}
