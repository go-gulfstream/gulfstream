package redis

import (
	"context"
	"fmt"

	"github.com/go-redis/redis/v8"

	"github.com/go-gulfstream/gulfstream/pkg/stream"
	"github.com/google/uuid"
)

const (
	versionPrefix = "v"
	streamPrefix  = "s"
)

var _ stream.Storage = (*Storage)(nil)

type Storage struct {
	streamName  string
	rds         *redis.Client
	blankStream func() *stream.Stream
}

func New(rds *redis.Client, streamName string, blankStream func() *stream.Stream) Storage {
	return Storage{
		rds:         rds,
		streamName:  streamName,
		blankStream: blankStream,
	}
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
	rawData, err := ss.MarshalBinary()
	if err != nil {
		return err
	}
	pipe := s.rds.TxPipeline()
	res := pipe.Get(ctx, toKey(ss, versionPrefix))
	currentVersion, err := res.Int()
	if err != nil && err != redis.Nil {
		return err
	}
	if currentVersion != ss.PreviousVersion() {
		return fmt.Errorf("storage/redis: mismatch stream version. got v%d, expected v%d",
			currentVersion, ss.PreviousVersion())
	}
	pipe.Set(ctx, toKey(ss, versionPrefix), ss.Version(), -1)
	pipe.Set(ctx, toKey(ss, streamPrefix), rawData, -1)
	if _, err = pipe.Exec(ctx); err != nil && err != redis.Nil {
		return err
	}
	return
}

func (s Storage) Load(ctx context.Context, streamID uuid.UUID) (*stream.Stream, error) {
	return nil, nil
}

func toKey(ss *stream.Stream, prefix string) string {
	return "gs." + prefix + "." + ss.Name() + ss.ID().String()
}
