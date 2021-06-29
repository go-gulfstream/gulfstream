package stream

import (
	"context"

	"github.com/google/uuid"
)

type Storage interface {
	BlankStream() *Stream
	Persist(ctx context.Context, s *Stream) error
	Load(ctx context.Context, streamName string, streamID uuid.UUID) (*Stream, error)
	MarkUnpublished(ctx context.Context, s *Stream) error
}

var _ Storage = (*StorageWithJournal)(nil)

type StorageWithJournal struct {
	storage Storage
	journal Journal
	txn     TxnFunc
}

func NewStorageWithJournal(storage Storage, journal Journal, txn TxnFunc) Storage {
	return StorageWithJournal{
		storage: storage,
		journal: journal,
		txn:     txn,
	}
}

func (sj StorageWithJournal) BlankStream() *Stream {
	return sj.storage.BlankStream()
}

func (sj StorageWithJournal) Persist(ctx context.Context, s *Stream) error {
	return sj.txn(ctx, func(txnCtx context.Context) error {
		if err := sj.Persist(txnCtx, s); err != nil {
			return err
		}
		return sj.journal.Append(txnCtx, s.Changes(), s.PreviousVersion())
	})
}

func (sj StorageWithJournal) Load(ctx context.Context, streamName string, streamID uuid.UUID) (*Stream, error) {
	return sj.storage.Load(ctx, streamName, streamID)
}

func (sj StorageWithJournal) MarkUnpublished(ctx context.Context, s *Stream) error {
	return sj.storage.MarkUnpublished(ctx, s)
}

type TxnFunc func(ctx context.Context, fn func(txnCtx context.Context) error) error
