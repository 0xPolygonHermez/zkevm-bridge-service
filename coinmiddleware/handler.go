package coinmiddleware

import (
	"context"
	"encoding/json"
	"time"

	"github.com/0xPolygonHermez/zkevm-bridge-service/bridgectrl/pb"
	"github.com/0xPolygonHermez/zkevm-bridge-service/redisstorage"
	"github.com/0xPolygonHermez/zkevm-node/log"
	"github.com/IBM/sarama"
	"github.com/pkg/errors"
)

const (
	maxRetries   = 5
	retryBackoff = 3 * time.Second
)

// MessageHandler implements sarama.ConsumerGroupHandler, handles the messages from kafka and populate the Redis storage
type MessageHandler struct {
	storage redisstorage.RedisStorage
}

func NewMessageHandler(redisStorage redisstorage.RedisStorage) sarama.ConsumerGroupHandler {
	return &MessageHandler{storage: redisStorage}
}

func (h *MessageHandler) Setup(sarama.ConsumerGroupSession) error {
	return nil
}

func (h *MessageHandler) Cleanup(sarama.ConsumerGroupSession) error {
	return nil
}

func (h *MessageHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for {
		select {
		case message, ok := <-claim.Messages():
			if !ok {
				log.Info("message channel was closed")
				return nil
			}
			log.Infof("message received topic[%v] partition[%v] offset[%v]", message.Topic, message.Partition, message.Offset)

			// Retry for 5 times, if still fails, ignore this message
			for i := 0; i < maxRetries; i++ {
				err := h.handleMessage(message)
				if err == nil {
					break
				}
				log.Errorf("handle kafka message error[%v] retryCnt[%v]", err, i)
				time.Sleep(retryBackoff)
			}
			session.MarkMessage(message, "")
		case <-session.Context().Done():
			return nil
		}
	}
}

func (h *MessageHandler) handleMessage(message *sarama.ConsumerMessage) error {
	body := &MessageBody{}
	err := json.Unmarshal(message.Value, body)
	if err != nil {
		return errors.Wrap(err, "unmarshal message body error")
	}

	if body.Data == nil {
		return errors.New("message data is nil")
	}
	pbPriceList := h.convertToPbPriceList(body.Data.PriceList)
	return h.storage.SetCoinPrice(context.Background(), pbPriceList)
}

func (h *MessageHandler) convertToPbPriceList(priceList []*PriceInfo) []*pb.SymbolPrice {
	var result []*pb.SymbolPrice
	for _, price := range priceList {
		result = append(result, &pb.SymbolPrice{
			Symbol:  price.Symbol,
			Price:   price.Price,
			Time:    uint64(price.Timestamp),
			Address: price.TokenAddress,
			ChainId: uint64(price.ChainID),
		})
	}
	return result
}
