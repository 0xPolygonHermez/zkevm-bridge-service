package messagepush

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/0xPolygonHermez/zkevm-bridge-service/bridgectrl/pb"
	"github.com/0xPolygonHermez/zkevm-bridge-service/utils"
	"github.com/0xPolygonHermez/zkevm-node/log"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/encoding/protojson"
)

const (
	fakeMessageLimit = 100
)

type fakeProducer struct {
	defaultTopic   string
	defaultPushKey string
	messages       map[string][]string // Map from topic name to list of messages
}

func newFakeProducer(cfg Config) KafkaProducer {
	return &fakeProducer{
		defaultTopic:   cfg.Topic,
		defaultPushKey: cfg.PushKey,
		messages:       make(map[string][]string),
	}
}

func (p *fakeProducer) Produce(msg interface{}, optFns ...produceOptFunc) error {
	opts := &produceOptions{
		topic:   p.defaultTopic,
		pushKey: p.defaultPushKey,
	}
	for _, f := range optFns {
		f(opts)
	}

	msgString, err := convertMsgToString(msg)
	if err != nil {
		return err
	}

	p.messages[opts.topic] = append(p.messages[opts.topic], msgString)
	// Keep the latest 100 messages only
	if len(p.messages[opts.topic]) > fakeMessageLimit {
		p.messages[opts.topic] = p.messages[opts.topic][1:]
	}
	log.Debugf("Produced to fake producer: topic[%v] msg[%v]", opts.topic, msgString)
	return nil
}

func (p *fakeProducer) PushTransactionUpdate(tx *pb.Transaction, optFns ...produceOptFunc) error {
	if tx == nil {
		return nil
	}
	// chain id convert to ok inner chain id
	if tx.FromChainId != 0 {
		tx.FromChainId = uint32(utils.GetInnerChainIdByStandardId(uint64(tx.FromChainId)))
	}
	if tx.ToChainId != 0 {
		tx.ToChainId = uint32(utils.GetInnerChainIdByStandardId(uint64(tx.ToChainId)))
	}
	var b []byte
	var err error
	if tx.Status == uint32(pb.TransactionStatus_TX_CREATED) {
		b, err = protojson.MarshalOptions{EmitUnpopulated: true}.Marshal(tx)
	} else {
		//b, err = json.Marshal([]*pb.Transaction{tx})
		b, err = json.Marshal(tx)
	}
	if err != nil {
		return errors.Wrap(err, "json marshal error")
	}

	msg := &PushMessage{
		BizCode:       BizCodeBridgeOrder,
		WalletAddress: tx.GetDestAddr(),
		RequestID:     utils.GenerateTraceID(),
		PushContent:   fmt.Sprintf("[%v]", string(b)),
		Time:          time.Now().UnixMilli(),
	}

	return p.Produce(msg, optFns...)
}

func (p *fakeProducer) Close() error {
	return nil
}

// GetFakeMessages returns latest 100 messages from the topic name
func (p *fakeProducer) GetFakeMessages(topic string) []string {
	allMsg := p.messages[topic][0:]
	p.messages[topic] = []string{}
	return allMsg
}
