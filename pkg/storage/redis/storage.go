package redis

import (
	"context"
	"fmt"
	"strconv"
	"strings"

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
	rds         redis.UniversalClient
	blankStream func() *stream.Stream
}

func New(
	rds redis.UniversalClient,
	streamName string,
	blankStream func() *stream.Stream,
) Storage {
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
	if strings.Compare(s.streamName, ss.Name()) != 0 {
		return fmt.Errorf("storage/redis: different stream names got %s, expected %s",
			ss.Name(), s.streamName)
	}
	rawData, err := ss.MarshalBinary()
	if err != nil {
		return err
	}
	versionKey := toKey(ss.Name(), ss.ID().String(), versionPrefix)
	strVer := s.rds.Get(ctx, versionKey).Val()
	var currentVersion int
	if len(strVer) > 0 {
		currentVersion, err = strconv.Atoi(strVer)
		if err != nil {
			return err
		}
	}
	if currentVersion > 0 && currentVersion <= ss.Version() {
		return fmt.Errorf("storage/redis: stream %s already exists", ss)
	}
	if currentVersion != ss.PreviousVersion() {
		return fmt.Errorf("storage/redis: mismatch stream version. got v%d, expected v%d",
			currentVersion, ss.PreviousVersion())
	}
	pipe := s.rds.TxPipeline()
	defer pipe.Close()
	pipe.Set(ctx, toKey(ss.Name(), ss.ID().String(), versionPrefix), ss.Version(), -1)
	pipe.Set(ctx, toKey(ss.Name(), ss.ID().String(), streamPrefix), rawData, -1)
	_, err = pipe.Exec(ctx)
	if err != nil && err != redis.Nil {
		return err
	}
	if err == redis.Nil {
		err = nil
	}
	return
}

func (s Storage) Load(ctx context.Context, streamID uuid.UUID) (*stream.Stream, error) {
	key := toKey(s.streamName, streamID.String(), streamPrefix)
	res := s.rds.Get(ctx, key)
	if res.Err() != nil && res.Err() != redis.Nil {
		return nil, res.Err()
	}
	data, err := res.Bytes()
	if err != nil && err != redis.Nil {
		return nil, err
	}
	blankStream := s.blankStream()
	if err := blankStream.UnmarshalBinary(data); err != nil {
		return nil, err
	}
	return blankStream, nil
}

func toKey(name string, id string, prefix string) string {
	return "gs." + prefix + "." + name + id
}
