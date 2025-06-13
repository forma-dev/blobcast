package sync

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/forma-dev/blobcast/pkg/celestia"
	"github.com/forma-dev/blobcast/pkg/crypto"
	"github.com/forma-dev/blobcast/pkg/types"
)

var (
	submitter      *Submitter
	submitterMutex sync.Mutex
)

type SubmissionItem struct {
	Payload []byte
	Result  chan SubmissionResult
}

type SubmissionResult struct {
	BlobID *types.BlobIdentifier
	Error  error
}

type Submitter struct {
	da            celestia.BlobStore
	queue         chan SubmissionItem
	priorityQueue chan SubmissionItem
	ctx           context.Context
	cancel        context.CancelFunc
	wg            sync.WaitGroup
	batchTimeout  time.Duration
}

func GetSubmitter(da celestia.BlobStore) *Submitter {
	submitterMutex.Lock()
	defer submitterMutex.Unlock()

	if submitter == nil {
		submitter = NewSubmitter(da)
	}

	return submitter
}

func NewSubmitter(da celestia.BlobStore) *Submitter {
	ctx, cancel := context.WithCancel(context.Background())

	s := &Submitter{
		da:            da,
		queue:         make(chan SubmissionItem, 1000),
		priorityQueue: make(chan SubmissionItem, 1000),
		ctx:           ctx,
		cancel:        cancel,
		batchTimeout:  100 * time.Millisecond,
	}

	s.wg.Add(1)
	go s.run()

	return s
}

func (s *Submitter) Submit(payload []byte) chan SubmissionResult {
	result := make(chan SubmissionResult, 1)
	item := SubmissionItem{
		Payload: payload,
		Result:  result,
	}

	select {
	case s.queue <- item:
	case <-s.ctx.Done():
		result <- SubmissionResult{Error: fmt.Errorf("context cancelled")}
		close(result)
	}

	return result
}

func (s *Submitter) SubmitWithPriority(payload []byte) chan SubmissionResult {
	result := make(chan SubmissionResult, 1)
	item := SubmissionItem{
		Payload: payload,
		Result:  result,
	}

	select {
	case s.priorityQueue <- item:
	case <-s.ctx.Done():
		result <- SubmissionResult{Error: fmt.Errorf("context cancelled")}
		close(result)
	}

	return result
}

func (s *Submitter) Close() {
	close(s.queue)
	close(s.priorityQueue)
	s.cancel()
	s.wg.Wait()
}

func (s *Submitter) run() {
	defer s.wg.Done()

	var currentBatch []SubmissionItem
	var currentBatchSize int

	timer := time.NewTimer(s.batchTimeout)
	timer.Stop()

	for {
		select {
		case item, ok := <-s.priorityQueue:
			if !ok {
				s.priorityQueue = nil
				continue
			}
			estimatedItemSize := len(item.Payload) + 100 // todo: estimate size better
			if currentBatchSize+estimatedItemSize > s.da.Cfg().MaxTxSize {
				s.submitBatch(currentBatch)
				currentBatch = []SubmissionItem{item}
				currentBatchSize = estimatedItemSize
			} else {
				currentBatch = append(currentBatch, item)
				currentBatchSize += estimatedItemSize
			}
			timer.Reset(s.batchTimeout)
		case item, ok := <-s.queue:
			if !ok {
				if len(currentBatch) > 0 {
					s.submitBatch(currentBatch)
				}
				return
			}

			estimatedItemSize := len(item.Payload) + 100 // todo: estimate size better
			if currentBatchSize+estimatedItemSize > s.da.Cfg().MaxTxSize {
				s.submitBatch(currentBatch)
				currentBatch = []SubmissionItem{item}
				currentBatchSize = estimatedItemSize
			} else {
				currentBatch = append(currentBatch, item)
				currentBatchSize += estimatedItemSize
			}
			timer.Reset(s.batchTimeout)
		case <-timer.C:
			if len(currentBatch) > 0 {
				s.submitBatch(currentBatch)
				currentBatch = []SubmissionItem{}
				currentBatchSize = 0
			}
		case <-s.ctx.Done():
			if len(currentBatch) > 0 {
				s.submitBatch(currentBatch)
			}
			return
		}
	}
}

func (s *Submitter) submitBatch(batch []SubmissionItem) {
	totalSize := 0
	msgs := make([][]byte, len(batch))
	for i, item := range batch {
		msgs[i] = item.Payload
		totalSize += len(item.Payload)
	}

	slog.Info("submitting batch to celestia", "batch_size", len(batch), "total_bytes", totalSize)

	commitments, height, err := s.da.StoreBatch(s.ctx, msgs)
	if err != nil {
		slog.Error("error submitting batch to celestia", "batch_size", len(batch), "error", err)
		for _, item := range batch {
			item.Result <- SubmissionResult{Error: err}
			close(item.Result)
		}
		return
	}

	slog.Info("sucessfully submitted batch to celestia", "batch_size", len(batch), "total_bytes", totalSize)

	for i, item := range batch {
		item.Result <- SubmissionResult{
			BlobID: &types.BlobIdentifier{
				Height:     height,
				Commitment: crypto.Hash(commitments[i]),
			},
		}
		close(item.Result)
	}
}
