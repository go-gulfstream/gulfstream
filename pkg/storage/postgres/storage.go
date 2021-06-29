package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgconn"

	"github.com/go-gulfstream/gulfstream/pkg/event"

	"github.com/go-gulfstream/gulfstream/pkg/stream"
	"github.com/google/uuid"

	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
)

type Storage struct {
	pool        *pgxpool.Pool
	blankStream func() *stream.Stream
	eventCodec  event.Encoding
}

func NewStorage(
	pool *pgxpool.Pool,
	blankStream func() *stream.Stream,
	opts ...StorageOption,
) Storage {
	storage := Storage{
		pool:        pool,
		blankStream: blankStream,
	}
	for _, opt := range opts {
		opt(&storage)
	}
	return storage
}

func (s Storage) Txn(ctx context.Context, fn func(txnCtx context.Context) error) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	txnCtx := context.WithValue(ctx, pkey, tx)
	if err := fn(txnCtx); err != nil {
		return tx.Rollback(ctx)
	}
	return tx.Commit(ctx)
}

func (s Storage) BlankStream() *stream.Stream {
	return s.blankStream()
}

func (s Storage) Persist(ctx context.Context, ss *stream.Stream) (err error) {
	if ss == nil || ss.PreviousVersion() == ss.Version() {
		return
	}
	rawData, err := ss.MarshalBinary()
	if err != nil {
		return err
	}
	if ss.PreviousVersion() == 0 {
		err = exec(ctx, s.pool, insertStateSQL, ss.Name(), ss.ID().String(), ss.Version(), rawData)
	} else {
		err = exec(ctx, s.pool, updateStateSQL, ss.Version(), rawData, ss.Name(), ss.ID().String())
	}
	return
}

func (s Storage) Load(ctx context.Context, streamName string, streamID uuid.UUID) (*stream.Stream, error) {
	row := queryRow(ctx, s.pool, selectStateSQL, streamName, streamID.String())
	var rawData []byte
	if err := row.Scan(&rawData); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("storage/postgres: storage.Load(%s,%s) not found",
				streamName, streamID)
		}
		return nil, err
	}
	blankStream := s.blankStream()
	if err := blankStream.UnmarshalBinary(rawData); err != nil {
		return nil, err
	}
	return blankStream, nil
}

func (s Storage) MarkUnpublished(ctx context.Context, ss *stream.Stream) (err error) {
	if ss == nil || ss.PreviousVersion() == ss.Version() {
		return
	}
	return exec(ctx, s.pool, insertUnpublishedSQL, ss.Name(), ss.ID().String(), ss.Version())
}

func (s Storage) decodeEvent(data []byte) (*event.Event, error) {
	if s.eventCodec != nil {
		return s.eventCodec.Decode(data)
	} else {
		return event.Decode(data)
	}
}

func (s Storage) encodeEvent(e *event.Event) ([]byte, error) {
	if s.eventCodec != nil {
		return s.eventCodec.Encode(e)
	} else {
		return event.Encode(e)
	}
}

type StorageOption func(*Storage)

func WithStorageCodec(c event.Encoding) StorageOption {
	return func(s *Storage) {
		s.eventCodec = c
	}
}

type txn int

const pkey txn = 9

func exec(ctx context.Context, pool *pgxpool.Pool, sql string, arguments ...interface{}) (err error) {
	tx, txnExists := ctx.Value(pkey).(pgx.Tx)
	var res pgconn.CommandTag
	if txnExists {
		res, err = tx.Exec(ctx, sql, arguments...)
	} else {
		res, err = pool.Exec(ctx, sql, arguments...)
	}
	if err != nil {
		return err
	}
	if res.RowsAffected() == 0 {
		return fmt.Errorf("storage/postgres: no affected rows")
	}
	return
}

func queryRow(ctx context.Context, pool *pgxpool.Pool, sql string, arguments ...interface{}) (row pgx.Row) {
	tx, txnExists := ctx.Value(pkey).(pgx.Tx)
	if txnExists {
		row = tx.QueryRow(ctx, sql, arguments...)
	} else {
		row = pool.QueryRow(ctx, sql, arguments...)
	}
	return
}

func query(ctx context.Context, pool *pgxpool.Pool, sql string, arguments ...interface{}) (rows pgx.Rows, err error) {
	tx, txnExists := ctx.Value(pkey).(pgx.Tx)
	if txnExists {
		rows, err = tx.Query(ctx, sql, arguments...)
	} else {
		rows, err = pool.Query(ctx, sql, arguments...)
	}
	return
}
