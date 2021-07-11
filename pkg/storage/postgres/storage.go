package storeagepostgres

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgconn"

	"github.com/go-gulfstream/gulfstream/pkg/event"

	"github.com/go-gulfstream/gulfstream/pkg/stream"
	"github.com/google/uuid"

	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
)

type Storage struct {
	pool           *pgxpool.Pool
	blankStream    func() *stream.Stream
	eventCodec     event.Encoding
	streamName     string
	journalEnabled bool
}

var _ stream.Storage = (*Storage)(nil)

func New(
	pool *pgxpool.Pool,
	streamName string,
	blankStream func() *stream.Stream,
	opts ...StorageOption,
) Storage {
	storage := Storage{
		pool:        pool,
		blankStream: blankStream,
		streamName:  streamName,
	}
	for _, opt := range opts {
		opt(&storage)
	}
	return storage
}

func (s Storage) StreamName() string {
	return s.streamName
}

func (s Storage) NewStream() *stream.Stream {
	return s.blankStream()
}

func (s Storage) Persist(ctx context.Context, ss *stream.Stream) (err error) {
	if ss == nil || ss.PreviousVersion() == ss.Version() {
		return
	}
	if strings.Compare(s.streamName, ss.Name()) != 0 {
		return fmt.Errorf("storage/postgres: different stream names got %s, expected %s",
			ss.Name(), s.streamName)
	}
	rawData, err := ss.MarshalBinary()
	if err != nil {
		return err
	}

	tx, txnExists := ctx.Value(pkey).(pgx.Tx)
	if !txnExists {
		tx, err = s.pool.Begin(ctx)
		if err != nil {
			return err
		}
		ctx = context.WithValue(ctx, pkey, tx)
		defer func() {
			if err != nil {
				err = tx.Rollback(ctx)
			} else {
				err = tx.Commit(ctx)
			}
		}()
	}

	for _, e := range ss.Changes() {
		eventData, err := s.encodeEvent(e)
		if err != nil {
			return err
		}
		if err = s.appendEventToJournal(ctx, e, eventData); err != nil {
			return err
		}
		if err = s.appendEventToOutbox(ctx, e, eventData); err != nil {
			return err
		}
	}

	if err := s.updateStreamVersion(ctx, ss); err != nil {
		return err
	}

	if ss.PreviousVersion() == 0 {
		err = exec(ctx, s.pool, insertStateSQL, ss.Name(), ss.ID().String(), ss.Version(), rawData)
	} else {
		err = exec(ctx, s.pool, updateStateSQL, ss.Version(), rawData, ss.Name(), ss.ID().String())
	}
	if err != nil {
		return
	}
	return exec(ctx, s.pool, deleteOutboxSQL, ss.Name(), ss.ID(), ss.Version())
}

func (s Storage) updateStreamVersion(ctx context.Context, ss *stream.Stream) (err error) {
	if ss.PreviousVersion() == 0 {
		err = exec(ctx, s.pool, insertVersionSQL, ss.Name(), ss.ID(), ss.Version())
	} else {
		err = exec(ctx, s.pool, updateVersionSQL, ss.Version(), ss.Name(), ss.ID(), ss.PreviousVersion())
	}
	if err != nil {
		if pgErr, ok := err.(*pgconn.PgError); ok && pgErr.Code == "23505" {
			return fmt.Errorf("storage/postgres: mismatch stream version: stream=%s, id=%s, ver=%d, expectedVer=%d",
				ss.Name(), ss.ID(), ss.Version(), ss.PreviousVersion())
		}
		return err
	}
	return
}

func (s Storage) appendEventToJournal(ctx context.Context, e *event.Event, data []byte) (err error) {
	if !s.journalEnabled {
		return
	}
	return exec(ctx, s.pool, insertEventSQL,
		e.StreamID().String(),
		e.StreamName(),
		e.Name(),
		e.Version(),
		e.Unix(),
		data,
	)
}

func (s Storage) appendEventToOutbox(ctx context.Context, e *event.Event, data []byte) error {
	if err := exec(ctx, s.pool, insertOutboxSQL,
		e.StreamID().String(),
		e.StreamName(),
		e.Version(),
		data,
	); err != nil {
		return err
	}
	return nil
}

func (s Storage) Load(ctx context.Context, streamID uuid.UUID) (*stream.Stream, error) {
	row := queryRow(ctx, s.pool, selectStateSQL, s.streamName, streamID.String())
	var rawData []byte
	if err := row.Scan(&rawData); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("storage/postgres: storage.Load(%s,%s) not found",
				s.streamName, streamID)
		}
		return nil, err
	}
	blankStream := s.blankStream()
	if err := blankStream.UnmarshalBinary(rawData); err != nil {
		return nil, err
	}
	return blankStream, nil
}

func (s Storage) Drop(ctx context.Context, streamID uuid.UUID) error {
	if s.journalEnabled {
		return nil
	}
	// TODO:
	return nil
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

func WithCodec(c event.Encoding) StorageOption {
	return func(s *Storage) {
		s.eventCodec = c
	}
}

func WithJournal() StorageOption {
	return func(s *Storage) {
		s.journalEnabled = true
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
	if res.RowsAffected() == 0 && !res.Delete() {
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
