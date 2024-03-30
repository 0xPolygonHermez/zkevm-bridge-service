package coinmiddleware

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"os"

	"github.com/0xPolygonHermez/zkevm-bridge-service/log"
	"github.com/0xPolygonHermez/zkevm-bridge-service/redisstorage"
	"github.com/IBM/sarama"
	"github.com/pkg/errors"
)

// KafkaConsumer provides the interface to consume from coin middleware kafka
type KafkaConsumer interface {
	Start(ctx context.Context)
	Close() error
}

type kafkaConsumerImpl struct {
	topics  []string
	client  sarama.ConsumerGroup
	handler sarama.ConsumerGroupHandler
}

func NewKafkaConsumer(cfg Config, redisStorage redisstorage.RedisStorage) (KafkaConsumer, error) {
	config := sarama.NewConfig()
	config.Consumer.Offsets.Initial = cfg.InitialOffset

	// Enable SASL authentication
	if cfg.Username != "" && cfg.Password != "" && cfg.RootCAPath != "" {
		config.Net.SASL.Enable = true
		config.Net.SASL.User = cfg.Username
		config.Net.SASL.Password = cfg.Password

		// Read the CA cert from file
		rootCA, err := os.ReadFile(cfg.RootCAPath)
		if err != nil {
			return nil, errors.Wrap(err, "Kafka consumer: read root CA cert fail")
		}

		caCertPool := x509.NewCertPool()
		if ok := caCertPool.AppendCertsFromPEM([]byte(rootCA)); !ok {
			return nil, errors.New("NewKafkaConsumer caCertPool.AppendCertsFromPEM")
		}

		config.Net.TLS.Enable = true
		config.Net.TLS.Config = &tls.Config{RootCAs: caCertPool, InsecureSkipVerify: true} // #nosec
	}

	client, err := sarama.NewConsumerGroup(cfg.Brokers, cfg.ConsumerGroupID, config)
	if err != nil {
		return nil, errors.Wrap(err, "kafka consumer group init error")
	}

	return &kafkaConsumerImpl{
		topics:  cfg.Topics,
		client:  client,
		handler: NewMessageHandler(redisStorage),
	}, nil
}

func (c *kafkaConsumerImpl) Start(ctx context.Context) {
	log.Debug("starting kafka consumer")
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	for {
		log.Debugf("start consume")
		err := c.client.Consume(ctx, c.topics, c.handler)
		if err != nil {
			log.Errorf("kafka consumer error: %v", err)
			//if errors.Is(err, sarama.ErrClosedConsumerGroup) {
			//	err = nil
			//}
			//err = errors.Wrap(err, "kafka consumer error")
			//panic(err)
			return
		}
		if err = ctx.Err(); err != nil {
			log.Errorf("kafka consumer ctx error: %v", err)
			//err = errors.Wrap(err, "kafka consumer ctx error")
			//panic(err)
			return
		}
	}
}

func (c *kafkaConsumerImpl) Close() error {
	log.Debug("closing kafka consumer...")
	return c.client.Close()
}
