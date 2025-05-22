package celestia

import (
	"encoding/hex"

	"github.com/celestiaorg/go-square/v2/share"
)

// Convert hex namespace ID to blob namespace
func HexToNamespaceV0(namespaceId string) (share.Namespace, error) {
	nsBytes, err := hex.DecodeString(namespaceId)
	if err != nil {
		return share.Namespace{}, err
	}

	return BytesToNamespaceV0(nsBytes)
}

func BytesToNamespaceV0(bytes []byte) (share.Namespace, error) {
	if len(bytes) == 28 {
		bytes = bytes[18:]
	}

	return share.NewV0Namespace(bytes)
}
