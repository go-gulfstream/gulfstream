package stream_test

import (
	"context"
	"testing"

	"github.com/go-gulfstream/gulfstream/pkg/stream"

	"github.com/stretchr/testify/assert"

	"github.com/go-gulfstream/gulfstream/pkg/event"
	"github.com/google/uuid"

	mockstream "github.com/go-gulfstream/gulfstream/mocks/stream"

	"github.com/golang/mock/gomock"
)

func TestMutator_EventSinkControllerNotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	storage := mockstream.NewMockStorage(ctrl)
	publisher := mockstream.NewMockPublisher(ctrl)

	// silent mode
	mutator := stream.NewMutator(storage, publisher)
	event1 := event.New("event1", "users", uuid.New(), 1, nil)
	assert.Nil(t, mutator.EventSink(context.Background(), event1))

	// strict mode
	mutator = stream.NewMutator(storage, publisher, stream.WithMutatorStrictMode())
	event1 = event.New("event1", "users", uuid.New(), 1, nil)
	err := mutator.EventSink(context.Background(), event1)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "mutator.EventSink controller")
}

func TestMutator_SetBlacklistOfEvents(t *testing.T) {
	ctrl := gomock.NewController(t)
	storage := mockstream.NewMockStorage(ctrl)
	publisher := mockstream.NewMockPublisher(ctrl)

	// silent mode
	mutator := stream.NewMutator(storage, publisher)
	mutator.SetBlacklistOfEvents("event1", "event2")
	event1 := event.New("event1", "users", uuid.New(), 1, nil)
	event2 := event.New("event2", "users", uuid.New(), 2, nil)
	assert.Nil(t, mutator.EventSink(context.Background(), event1))
	assert.Nil(t, mutator.EventSink(context.Background(), event2))

	// strict mode
	mutator = stream.NewMutator(storage, publisher, stream.WithMutatorStrictMode())
	mutator.SetBlacklistOfEvents("event1", "event2")
	event1 = event.New("event1", "users", uuid.New(), 1, nil)
	err := mutator.EventSink(context.Background(), event1)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "stream: mutator.EventSink event cycles")
}

func TestMutator_EventSinkPickStreamError(t *testing.T) {
	ctrl := gomock.NewController(t)
	groupJoinedController := mockstream.NewMockEventController(ctrl)
	storage := mockstream.NewMockStorage(ctrl)
	storage.EXPECT().StreamName().Return("users")
	publisher := mockstream.NewMockPublisher(ctrl)

	// users stream
	usersMutator := stream.NewMutator(storage, publisher, stream.WithMutatorStrictMode())
	usersMutator.AddEventController("groupJoined", groupJoinedController)

	// group stream
	groupJoinedEvent := event.New("groupJoined", "group", uuid.New(), 1, nil)
	groupJoinedController.EXPECT().PickStream(groupJoinedEvent).Return(stream.Picker{
		StreamID: uuid.UUID{},
	})
	err := usersMutator.EventSink(context.Background(), groupJoinedEvent)
	assert.Error(t, err)
}

func TestMutator_EventSinkPickOneStream(t *testing.T) {
	ctrl := gomock.NewController(t)
	storage := mockstream.NewMockStorage(ctrl)
	publisher := mockstream.NewMockPublisher(ctrl)
	state := &userState{}
	userID := uuid.New()
	userStream := stream.New("users", userID, state)
	groupID := uuid.New()

	ctx := context.Background()
	storage.EXPECT().Load(ctx, userID).Return(userStream, nil)
	storage.EXPECT().Persist(ctx, userStream).Return(nil)

	publisher.EXPECT().Publish(gomock.Any()).Return(nil)

	// users stream
	usersMutator := stream.NewMutator(
		storage,
		publisher,
		stream.WithMutatorStrictMode(),
	)
	groupJoinedController := stream.EventControllerFunc(
		func(e *event.Event) stream.Picker {
			return stream.Picker{
				StreamID: e.Payload().(*groupJoinedPayload).UserID,
			}
		}, func(ctx context.Context, s *stream.Stream, groupJoinedEvent *event.Event) error {
			// event-command from other stream make a stream event for current stream
			s.Mutate("userJoined", &userJoinedPayload{
				GroupID: groupJoinedEvent.StreamID(),
			})
			return nil
		})
	usersMutator.AddEventController("groupJoined", groupJoinedController)

	// group stream
	groupJoinedEvent := event.New("groupJoined", "group", groupID, 100,
		&groupJoinedPayload{
			UserID: userID,
		})

	err := usersMutator.EventSink(ctx, groupJoinedEvent)
	assert.Nil(t, err)
	assert.Len(t, userStream.State().(*userState).Groups, 1)
	assert.Equal(t, groupJoinedEvent.StreamID(), userStream.State().(*userState).Groups[0])
}

type groupJoinedPayload struct {
	Name   string
	UserID uuid.UUID
}

type userJoinedPayload struct {
	GroupID uuid.UUID
}

type userState struct {
	Groups []uuid.UUID
}

func (s *userState) Mutate(e *event.Event) {
	switch payload := e.Payload().(type) {
	case *userJoinedPayload:
		s.Groups = append(s.Groups, payload.GroupID)
	}
}

func (s *userState) MarshalBinary() ([]byte, error) {
	return nil, nil
}

func (s *userState) UnmarshalBinary(data []byte) error {
	return nil
}
