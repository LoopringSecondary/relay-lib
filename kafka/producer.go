package kafka

import (
	"encoding/json"
	"fmt"
	"github.com/Shopify/sarama"
)

type MessageProducer struct {
	pd sarama.SyncProducer
}

func (md *MessageProducer) Initialize(brokerList []string) (err error) {
	config := sarama.NewConfig()
	config.Producer.RequiredAcks = sarama.WaitForAll // Wait for all in-sync replicas to ack the message
	config.Producer.Retry.Max = 10                   // Retry up to 10 times to produce the message
	config.Producer.Return.Successes = true

	producer, err := sarama.NewSyncProducer(brokerList, config)
	if err == nil {
		md.pd = producer
	}
	return err
}

func (md *MessageProducer) SendMessage(topic string, data interface{}, key string) (partition int32, offset int64, sendErr error) {
	if data == nil {
		return -1, -1, fmt.Errorf("message to send is null")
	}

	bytes, err := json.Marshal(data)
	if err != nil {
		fmt.Errorf("Failed to Marshal Msg")
	}
	return md.pd.SendMessage(&sarama.ProducerMessage{
		Topic: topic,
		Value: sarama.ByteEncoder(bytes),
		Key:   sarama.StringEncoder(key),
	})
}

func (md *MessageProducer) Close() error {
	return md.pd.Close()
}
