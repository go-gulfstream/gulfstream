package eventbuskafka

import (
	"github.com/Shopify/sarama"
	"github.com/go-gulfstream/gulfstream/pkg/event"
	"github.com/go-gulfstream/gulfstream/pkg/stream"
)

var _ stream.Publisher = (*Publisher)(nil)

type Publisher struct {
	brokers    []string
	conf       *sarama.Config
	eventCodec event.Encoding
	producer   sarama.SyncProducer
}

type PublisherOption func(*Publisher)

func NewPublisher(
	addr []string,
	conf *sarama.Config,
	opts ...PublisherOption,
) *Publisher {
	if conf == nil {
		conf = DefaultConfig()
	}
	publisher := &Publisher{
		brokers: addr,
	}
	for _, opt := range opts {
		opt(publisher)
	}
	return publisher
}

func WithPublisherCodec(codec event.Encoding) PublisherOption {
	return func(p *Publisher) {
		p.eventCodec = codec
	}
}

func (p *Publisher) Connect() (err error) {
	p.producer, err = sarama.NewSyncProducer(p.brokers, p.conf)
	return
}

func (p *Publisher) Publish(events []*event.Event) error {
	if p.producer == nil {
		return nil
	}
	messages := make([]*sarama.ProducerMessage, len(events))
	for i, e := range events {
		data, err := p.encodeEvent(e)
		if err != nil {
			return err
		}
		route := e.StreamName() + e.StreamID().String()
		headers := []sarama.RecordHeader{
			{
				Key:   []byte("_stream"),
				Value: []byte(e.StreamName()),
			},
			{
				Key:   []byte("_event"),
				Value: []byte(e.Name()),
			},
		}
		message := &sarama.ProducerMessage{
			Topic:   e.StreamName(),
			Key:     sarama.StringEncoder(route),
			Value:   sarama.ByteEncoder(data),
			Headers: headers,
		}
		messages[i] = message
	}
	return p.producer.SendMessages(messages)
}

func (p *Publisher) Close() error {
	if p.producer == nil {
		return nil
	}
	return p.producer.Close()
}

func (p *Publisher) encodeEvent(e *event.Event) ([]byte, error) {
	if p.eventCodec != nil {
		return p.eventCodec.Encode(e)
	} else {
		return event.Encode(e)
	}
}
