package sync

import (
	"github.com/forma-dev/blobcast/pkg/types"
	"github.com/google/uuid"
)

// Subscription represents a block header subscription
type Subscription struct {
	ID uuid.UUID
	Ch <-chan *types.BlockHeader
}
