package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgconn"

	"github.com/jackc/pgx/v4"

	"github.com/go-gulfstream/gulfstream/pkg/event"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v4/pgxpool"
)

type Journal struct {
	pool       *pgxpool.Pool
	eventCodec event.Encoding
}

func NewJournal(
	pool *pgxpool.Pool,
	opts ...JournalOption,
) Journal {
	journal := Journal{pool: pool}
	for _, opt := range opts {
		opt(&journal)
	}
	return journal
}

type JournalOption func(*Journal)

func WithJournalCodec(c event.Encoding) JournalOption {
	return func(j *Journal) {
		j.eventCodec = c
	}
}

func (j Journal) Append(ctx context.Context, events []*event.Event, expectedVersion int) (err error) {
	if len(events) == 0 {
		return
	}

	tx, txnExists := ctx.Value(pkey).(pgx.Tx)
	if !txnExists {
		tx, err = j.pool.Begin(ctx)
		if err != nil {
			return err
		}
		ctx = context.WithValue(ctx, pkey, tx)
		defer func() {
			if err != nil {
				_ = tx.Rollback(ctx)
			} else {
				_ = tx.Commit(ctx)
			}
		}()
	}

	var (
		streamID   = events[0].StreamID().String()
		streamName = events[0].StreamName()
		version    = events[len(events)-1].Version()
		res        pgconn.CommandTag
	)

	if expectedVersion == 0 {
		res, err = tx.Exec(ctx, insertVersionSQL, streamID, streamName, version)
	} else {
		res, err = tx.Exec(ctx, updateVersionSQL, version, streamID, streamName, expectedVersion)
	}
	verErr := fmt.Errorf("storage/postgres: journal.Append version mismatch: stream=%s, id=%s, ver=%d, expectedVer=%d",
		streamName, streamID, version, expectedVersion)
	if err != nil {
		if perr, ok := err.(*pgconn.PgError); ok && perr.Code == "23505" {
			return verErr
		}
		return err
	}
	if res.RowsAffected() == 0 {
		return verErr
	}
	for _, e := range events {
		rawData, err := j.encodeEvent(e)
		if err != nil {
			return err
		}
		if _, err = tx.Exec(ctx, insertEventSQL,
			e.StreamID().String(),
			e.StreamName(),
			e.Name(),
			e.Version(),
			e.Unix(),
			rawData,
		); err != nil {
			return err
		}
	}
	return
}

func (j Journal) Load(ctx context.Context, streamName string, streamID uuid.UUID) ([]*event.Event, error) {
	rows, err := query(ctx, j.pool, selectEventsSQL, streamName, streamID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	events := make([]*event.Event, 0, 32)
	for rows.Next() {
		if rows.Err() != nil {
			return nil, rows.Err()
		}
		var rawdata []byte
		if err := rows.Scan(&rawdata); err != nil {
			return nil, err
		}
		e, err := j.decodeEvent(rawdata)
		if err != nil {
			return nil, err
		}
		events = append(events, e)
	}
	return events, rows.Err()
}

func (j Journal) decodeEvent(data []byte) (*event.Event, error) {
	if j.eventCodec != nil {
		return j.eventCodec.Decode(data)
	} else {
		return event.Decode(data)
	}
}

func (j Journal) encodeEvent(e *event.Event) ([]byte, error) {
	if j.eventCodec != nil {
		return j.eventCodec.Encode(e)
	} else {
		return event.Encode(e)
	}
}
