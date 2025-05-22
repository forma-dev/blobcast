package types

import (
	"crypto/sha256"
	"time"

	"github.com/forma-dev/blobcast/pkg/crypto"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	pbPrimitivesV1 "github.com/forma-dev/blobcast/pkg/proto/blobcast/primitives/v1"
	pbRollupV1 "github.com/forma-dev/blobcast/pkg/proto/blobcast/rollup/v1"
)

type Block struct {
	Header *BlockHeader
	Body   *BlockBody
}

type BlockHeader struct {
	Version             uint32
	ChainID             string
	Height              uint64
	CelestiaBlockHeight uint64
	Timestamp           time.Time
	ParentHash          crypto.Hash
	DirsRoot            crypto.Hash
	FilesRoot           crypto.Hash
	ChunksRoot          crypto.Hash
	StateRoot           crypto.Hash
}

type BlockBody struct {
	Dirs   []*BlobIdentifier
	Files  []*BlobIdentifier
	Chunks []*BlobIdentifier
}

var blockHeaderHashCache = make(map[uint64]crypto.Hash)

func NewBlockFromHeader(header *BlockHeader, dirs []*BlobIdentifier, files []*BlobIdentifier, chunks []*BlobIdentifier) *Block {
	block := &Block{
		Header: CopyBlockHeader(header),
		Body: &BlockBody{
			Dirs:   dirs,
			Files:  files,
			Chunks: chunks,
		},
	}

	block.Header.Version = 1

	// compute merkle roots
	block.Header.DirsRoot = BlobIdentifiersMerkleRoot(dirs)
	block.Header.FilesRoot = BlobIdentifiersMerkleRoot(files)
	block.Header.ChunksRoot = BlobIdentifiersMerkleRoot(chunks)

	return block
}

func NewBlock(dirs []*BlobIdentifier, files []*BlobIdentifier, chunks []*BlobIdentifier) *Block {
	return NewBlockFromHeader(&BlockHeader{}, dirs, files, chunks)
}

func (bh *BlockHeader) Hash() crypto.Hash {
	if hash, ok := blockHeaderHashCache[bh.Height]; ok {
		return hash
	}

	bhProto := &pbRollupV1.BlockHeader{
		Version:             bh.Version,
		ChainId:             bh.ChainID,
		Height:              bh.Height,
		CelestiaBlockHeight: bh.CelestiaBlockHeight,
		Timestamp:           timestamppb.New(bh.Timestamp),
		ParentHash:          bh.ParentHash[:],
		DirsRoot:            bh.DirsRoot[:],
		FilesRoot:           bh.FilesRoot[:],
		ChunksRoot:          bh.ChunksRoot[:],
		StateRoot:           bh.StateRoot[:],
	}

	protoBytes, err := proto.Marshal(bhProto)
	if err != nil {
		panic(err)
	}

	hash := sha256.Sum256(protoBytes)
	blockHeaderHashCache[bh.Height] = hash
	return hash
}

func (b *Block) Hash() crypto.Hash {
	return b.Header.Hash()
}

func (b *Block) Height() uint64 {
	return b.Header.Height
}

func (b *Block) Proto() *pbRollupV1.Block {
	return &pbRollupV1.Block{
		Header: b.Header.Proto(),
		Body:   b.Body.Proto(),
	}
}

func (bh *BlockHeader) Proto() *pbRollupV1.BlockHeader {
	return &pbRollupV1.BlockHeader{
		Version:             bh.Version,
		ChainId:             bh.ChainID,
		Height:              bh.Height,
		CelestiaBlockHeight: bh.CelestiaBlockHeight,
		Timestamp:           timestamppb.New(bh.Timestamp),
		ParentHash:          bh.ParentHash[:],
		DirsRoot:            bh.DirsRoot[:],
		FilesRoot:           bh.FilesRoot[:],
		ChunksRoot:          bh.ChunksRoot[:],
		StateRoot:           bh.StateRoot[:],
	}
}

func (bb *BlockBody) Proto() *pbRollupV1.BlockBody {
	dirs := make([]*pbPrimitivesV1.BlobIdentifier, len(bb.Dirs))
	for i, dir := range bb.Dirs {
		dirs[i] = dir.Proto()
	}
	files := make([]*pbPrimitivesV1.BlobIdentifier, len(bb.Files))
	for i, file := range bb.Files {
		files[i] = file.Proto()
	}
	chunks := make([]*pbPrimitivesV1.BlobIdentifier, len(bb.Chunks))
	for i, chunk := range bb.Chunks {
		chunks[i] = chunk.Proto()
	}
	return &pbRollupV1.BlockBody{
		Dirs:   dirs,
		Files:  files,
		Chunks: chunks,
	}
}

func BlockFromProto(pbBlock *pbRollupV1.Block) *Block {
	return &Block{
		Header: BlockHeaderFromProto(pbBlock.Header),
		Body:   BlockBodyFromProto(pbBlock.Body),
	}
}

func BlockHeaderFromProto(pbHeader *pbRollupV1.BlockHeader) *BlockHeader {
	return &BlockHeader{
		Version:             pbHeader.Version,
		ChainID:             pbHeader.ChainId,
		Height:              pbHeader.Height,
		CelestiaBlockHeight: pbHeader.CelestiaBlockHeight,
		Timestamp:           pbHeader.Timestamp.AsTime(),
		ParentHash:          crypto.Hash(pbHeader.ParentHash),
		DirsRoot:            crypto.Hash(pbHeader.DirsRoot),
		FilesRoot:           crypto.Hash(pbHeader.FilesRoot),
		ChunksRoot:          crypto.Hash(pbHeader.ChunksRoot),
		StateRoot:           crypto.Hash(pbHeader.StateRoot),
	}
}

func BlockBodyFromProto(pbBody *pbRollupV1.BlockBody) *BlockBody {
	dirs := make([]*BlobIdentifier, len(pbBody.Dirs))
	for i, dir := range pbBody.Dirs {
		dirs[i] = BlobIdentifierFromProto(dir)
	}
	files := make([]*BlobIdentifier, len(pbBody.Files))
	for i, file := range pbBody.Files {
		files[i] = BlobIdentifierFromProto(file)
	}
	chunks := make([]*BlobIdentifier, len(pbBody.Chunks))
	for i, chunk := range pbBody.Chunks {
		chunks[i] = BlobIdentifierFromProto(chunk)
	}
	return &BlockBody{
		Dirs:   dirs,
		Files:  files,
		Chunks: chunks,
	}
}

func CopyBlockHeader(bh *BlockHeader) *BlockHeader {
	return &BlockHeader{
		Version:             bh.Version,
		ChainID:             bh.ChainID,
		Height:              bh.Height,
		CelestiaBlockHeight: bh.CelestiaBlockHeight,
		Timestamp:           bh.Timestamp,
		ParentHash:          bh.ParentHash,
		DirsRoot:            bh.DirsRoot,
		FilesRoot:           bh.FilesRoot,
		ChunksRoot:          bh.ChunksRoot,
		StateRoot:           bh.StateRoot,
	}
}
