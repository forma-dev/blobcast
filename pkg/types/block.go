package types

import (
	"time"

	"github.com/forma-dev/blobcast/pkg/crypto"
	"github.com/forma-dev/blobcast/pkg/util"
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

	hash := crypto.HashBytes(bh.Bytes())
	blockHeaderHashCache[bh.Height] = hash
	return hash
}

func (b *Block) Hash() crypto.Hash {
	return b.Header.Hash()
}

func (b *Block) Height() uint64 {
	return b.Header.Height
}

func (bh *BlockHeader) Bytes() []byte {
	var headerBytes []byte
	headerBytes = append(headerBytes, util.BytesFromUint32(bh.Version)...)
	headerBytes = append(headerBytes, util.BytesFromShortString(bh.ChainID)...)
	headerBytes = append(headerBytes, util.BytesFromUint64(bh.Height)...)
	headerBytes = append(headerBytes, util.BytesFromUint64(bh.CelestiaBlockHeight)...)
	headerBytes = append(headerBytes, util.BytesFromTime(bh.Timestamp)...)
	headerBytes = append(headerBytes, bh.ParentHash.Bytes()...)
	headerBytes = append(headerBytes, bh.DirsRoot.Bytes()...)
	headerBytes = append(headerBytes, bh.FilesRoot.Bytes()...)
	headerBytes = append(headerBytes, bh.ChunksRoot.Bytes()...)
	headerBytes = append(headerBytes, bh.StateRoot.Bytes()...)
	return headerBytes
}

func (bb *BlockBody) Bytes() []byte {
	totalItems := len(bb.Dirs) + len(bb.Files) + len(bb.Chunks)
	bodyBytes := make([]byte, 0, 12+totalItems*40)

	// Serialize counts (3 * 4 bytes = 12 bytes header)
	bodyBytes = append(bodyBytes, util.BytesFromUint32(uint32(len(bb.Dirs)))...)
	bodyBytes = append(bodyBytes, util.BytesFromUint32(uint32(len(bb.Files)))...)
	bodyBytes = append(bodyBytes, util.BytesFromUint32(uint32(len(bb.Chunks)))...)

	for _, dir := range bb.Dirs {
		bodyBytes = append(bodyBytes, dir.Bytes()...)
	}
	for _, file := range bb.Files {
		bodyBytes = append(bodyBytes, file.Bytes()...)
	}
	for _, chunk := range bb.Chunks {
		bodyBytes = append(bodyBytes, chunk.Bytes()...)
	}

	return bodyBytes
}

func (b *Block) Bytes() []byte {
	var blockBytes []byte
	headerBytes := b.Header.Bytes()
	blockBytes = append(blockBytes, util.BytesFromUint32(uint32(len(headerBytes)))...)
	blockBytes = append(blockBytes, headerBytes...)
	blockBytes = append(blockBytes, b.Body.Bytes()...)
	return blockBytes
}

func BlockFromBytes(b []byte) *Block {
	headerLength := util.Uint32FromBytes(b[:4])
	header := BlockHeaderFromBytes(b[4 : 4+headerLength])
	body := BlockBodyFromBytes(b[4+headerLength:])
	return &Block{
		Header: header,
		Body:   body,
	}
}

func BlockHeaderFromBytes(b []byte) *BlockHeader {
	offset := 0

	// Read version
	version := util.Uint32FromBytes(b[offset : offset+4])
	offset += 4

	// Read chain ID (length-prefixed)
	chainIdLength := int(b[offset])
	offset += 1
	chainID := util.ShortStringFromBytes(b[offset-1 : offset+chainIdLength])
	offset += chainIdLength

	// Read height
	height := util.Uint64FromBytes(b[offset : offset+8])
	offset += 8

	// Read celestia block height
	celestiaBlockHeight := util.Uint64FromBytes(b[offset : offset+8])
	offset += 8

	// Read timestamp
	timestamp := time.Unix(int64(util.Uint64FromBytes(b[offset:offset+8])), 0)
	offset += 8

	// Read hashes (each 32 bytes)
	parentHash := crypto.Hash(b[offset : offset+32])
	offset += 32
	dirsRoot := crypto.Hash(b[offset : offset+32])
	offset += 32
	filesRoot := crypto.Hash(b[offset : offset+32])
	offset += 32
	chunksRoot := crypto.Hash(b[offset : offset+32])
	offset += 32
	stateRoot := crypto.Hash(b[offset : offset+32])

	return &BlockHeader{
		Version:             version,
		ChainID:             chainID,
		Height:              height,
		CelestiaBlockHeight: celestiaBlockHeight,
		Timestamp:           timestamp,
		ParentHash:          parentHash,
		DirsRoot:            dirsRoot,
		FilesRoot:           filesRoot,
		ChunksRoot:          chunksRoot,
		StateRoot:           stateRoot,
	}
}

func BlockBodyFromBytes(b []byte) *BlockBody {
	offset := 0

	dirsCount := util.Uint32FromBytes(b[offset : offset+4])
	offset += 4
	filesCount := util.Uint32FromBytes(b[offset : offset+4])
	offset += 4
	chunksCount := util.Uint32FromBytes(b[offset : offset+4])
	offset += 4

	dirs := make([]*BlobIdentifier, dirsCount)
	for i := range int(dirsCount) {
		dirs[i] = BlobIdentifierFromBytes(b[offset : offset+40])
		offset += 40
	}

	files := make([]*BlobIdentifier, filesCount)
	for i := range int(filesCount) {
		files[i] = BlobIdentifierFromBytes(b[offset : offset+40])
		offset += 40
	}

	chunks := make([]*BlobIdentifier, chunksCount)
	for i := range int(chunksCount) {
		chunks[i] = BlobIdentifierFromBytes(b[offset : offset+40])
		offset += 40
	}

	return &BlockBody{
		Dirs:   dirs,
		Files:  files,
		Chunks: chunks,
	}
}

func (b *Block) Proto() *pbRollupV1.Block {
	return &pbRollupV1.Block{
		Header: b.Header.Proto(),
		Body:   b.Body.Proto(),
		Hash:   b.Header.Hash().Bytes(),
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
