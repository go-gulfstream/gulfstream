package storage

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	storageredis "github.com/go-gulfstream/gulfstream/pkg/storage/redis"
	"github.com/go-gulfstream/gulfstream/pkg/stream"

	"github.com/go-redis/redis/v8"

	"github.com/go-gulfstream/gulfstream/tests"
	"github.com/stretchr/testify/suite"
)

func TestStorage_Redis(t *testing.T) {
	tests.SkipIfNotIntegration(t)

	conn := redis.NewClient(&redis.Options{
		Addr: tests.RedisAddr,
		DB:   0,
	})
	defer conn.Close()
	conn.FlushAll(context.Background())

	suite.Run(t, &RedisSuite{rdb: conn})
}

type RedisSuite struct {
	suite.Suite
	rdb     *redis.Client
	ctx     context.Context
	storage stream.Storage
}

func (s *RedisSuite) SetupTest() {
	s.ctx = context.Background()
	s.storage = storageredis.New(s.rdb, streamName, blankStream)
}

func (s *RedisSuite) TearDownTest() {
	s.rdb.FlushDB(s.ctx)
}

func (s *RedisSuite) TestPersist() {
	testStream := blankStream()
	testStream.Mutate("someEvent", nil)
	assert.Nil(s.T(), s.storage.Persist(s.ctx, testStream))
	iter := s.rdb.Scan(s.ctx, 0, "", 10).Iterator()
	var match int
	for iter.Next(s.ctx) {
		if strings.HasPrefix(iter.Val(), "gs.v.test") {
			match++
		}
		if strings.HasPrefix(iter.Val(), "gs.s.test") {
			match++
		}
	}
	assert.Equal(s.T(), 2, match)
	assert.Error(s.T(), s.storage.Persist(s.ctx, testStream))
}

func (s *RedisSuite) TestPersistWithConcurrency() {
	testStream := blankStream()
	testStream.Mutate("event", nil)
	assert.NoError(s.T(), s.storage.Persist(s.ctx, testStream))

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				someStream, err := s.storage.Load(s.ctx, testStream.ID())
				assert.NoError(s.T(), err)
				someStream.Mutate("someEvent", nil)
				per := s.storage.Persist(s.ctx, someStream)
				if per == nil {
					break
				}
				time.Sleep(50 * time.Millisecond)
			}
		}()
	}
	wg.Wait()

	fss, err := s.storage.Load(s.ctx, testStream.ID())
	assert.NoError(s.T(), err)
	assert.Equal(s.T(), 11, fss.Version())
}

func (s *RedisSuite) TestLoad() {
	testStream := blankStream()
	testStream.Mutate("someEvent", nil)
	assert.Nil(s.T(), s.storage.Persist(s.ctx, testStream))
	other, err := s.storage.Load(s.ctx, testStream.ID())
	assert.NoError(s.T(), err)
	assert.Equal(s.T(), testStream.ID(), other.ID())
}
