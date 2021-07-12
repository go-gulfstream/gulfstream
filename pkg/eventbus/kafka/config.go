package eventbuskafka

import (
	"github.com/Shopify/sarama"
	"github.com/google/uuid"
)

func DefaultConfig() *sarama.Config {
	conf := sarama.NewConfig()
	conf.ClientID = uuid.New().String()
	conf.Version = sarama.V2_7_0_0
	conf.Producer.Return.Successes = true
	conf.Producer.Return.Errors = true
	conf.Producer.MaxMessageBytes = 1e6
	conf.Producer.Retry.Max = 30
	conf.Producer.RequiredAcks = sarama.WaitForAll
	conf.Consumer.Offsets.Initial = sarama.OffsetNewest
	conf.Consumer.Group.Rebalance.Strategy = sarama.BalanceStrategyRoundRobin
	return conf
}
