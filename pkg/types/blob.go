package types

import (
	"fmt"
	"strings"

	"github.com/multiformats/go-base36"
	"google.golang.org/protobuf/proto"

	"github.com/forma-dev/blobcast/pkg/crypto"
	"github.com/forma-dev/blobcast/pkg/crypto/merkle"
	pbPrimitivesV1 "github.com/forma-dev/blobcast/pkg/proto/blobcast/primitives/v1"
)

type BlobIdentifier struct {
	Height     uint64
	Commitment []byte
}

func BlobIdentifierFromString(encoded string) (*BlobIdentifier, error) {
	if !strings.HasPrefix(encoded, "bc") {
		return nil, fmt.Errorf("invalid identifier: %s", encoded)
	}

	serialized, err := base36.DecodeString(strings.TrimPrefix(encoded, "bc"))
	if err != nil {
		return nil, fmt.Errorf("error decoding base36 data: %w", err)
	}

	// Unmarshal proto
	protoManifest := &pbPrimitivesV1.BlobIdentifier{}
	if err := proto.Unmarshal(serialized, protoManifest); err != nil {
		return nil, fmt.Errorf("error unmarshalling manifest identifier: %w", err)
	}

	return &BlobIdentifier{
		Height:     protoManifest.Height,
		Commitment: protoManifest.Commitment,
	}, nil
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
		Commitment: proto.Commitment,
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
	protoManifest := &pbPrimitivesV1.BlobIdentifier{
		Height:     b.Height,
		Commitment: b.Commitment,
	}

	serialized, err := proto.Marshal(protoManifest)
	if err != nil {
		return nil
	}

	return serialized
}

func (b *BlobIdentifier) Proto() *pbPrimitivesV1.BlobIdentifier {
	return &pbPrimitivesV1.BlobIdentifier{
		Height:     b.Height,
		Commitment: b.Commitment,
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
