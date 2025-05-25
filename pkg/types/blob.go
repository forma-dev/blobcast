package types

import (
	"fmt"
	"strings"

	"github.com/multiformats/go-base36"

	"github.com/forma-dev/blobcast/pkg/crypto"
	"github.com/forma-dev/blobcast/pkg/crypto/merkle"
	pbPrimitivesV1 "github.com/forma-dev/blobcast/pkg/proto/blobcast/primitives/v1"
	"github.com/forma-dev/blobcast/pkg/util"
)

type BlobIdentifier struct {
	Height     uint64
	Commitment crypto.Hash
}

func BlobIdentifierFromString(encoded string) (*BlobIdentifier, error) {
	if !strings.HasPrefix(encoded, "bc") {
		return nil, fmt.Errorf("invalid identifier: %s", encoded)
	}

	serialized, err := base36.DecodeString(strings.TrimPrefix(encoded, "bc"))
	if err != nil {
		return nil, fmt.Errorf("error decoding base36 data: %w", err)
	}

	return BlobIdentifierFromBytes(serialized), nil
}

func BlobIdentifierFromURL(url string) (*BlobIdentifier, error) {
	if !strings.HasPrefix(url, "blobcast://") {
		return nil, fmt.Errorf("invalid URL: %s", url)
	}
	return BlobIdentifierFromString(strings.TrimPrefix(url, "blobcast://"))
}

func BlobIdentifierFromProto(proto *pbPrimitivesV1.BlobIdentifier) *BlobIdentifier {
	return &BlobIdentifier{
		Height:     proto.Height,
		Commitment: crypto.Hash(proto.Commitment),
	}
}

func BlobIdentifierFromBytes(b []byte) *BlobIdentifier {
	return &BlobIdentifier{
		Height:     util.Uint64FromBytes(b[:8]),
		Commitment: crypto.Hash(b[8:]),
	}
}

func (b *BlobIdentifier) URL() string {
	return fmt.Sprintf("blobcast://%s", b.String())
}

func (b *BlobIdentifier) ID() string {
	serialized := b.Bytes()
	if serialized == nil {
		return ""
	}
	return "bc" + base36.EncodeToStringLc(serialized)
}

func (b *BlobIdentifier) Bytes() []byte {
	return append(util.BytesFromUint64(b.Height), b.Commitment.Bytes()...)
}

func (b *BlobIdentifier) Proto() *pbPrimitivesV1.BlobIdentifier {
	return &pbPrimitivesV1.BlobIdentifier{
		Height:     b.Height,
		Commitment: b.Commitment.Bytes(),
	}
}

func (b *BlobIdentifier) String() string {
	return b.ID()
}

func BlobIdentifiersMerkleRoot(blobs []*BlobIdentifier) crypto.Hash {
	bytes := make([][]byte, len(blobs))
	for i, blob := range blobs {
		bytes[i] = blob.Bytes()
	}
	return merkle.CalculateMerkleRoot(bytes)
}
