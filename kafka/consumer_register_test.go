package kafka_test

import (
	"fmt"
	"time"
	"github.com/Loopring/relay-lib/kafka"
	"testing"
)

func TestConsumer(t *testing.T) {
	address := "127.0.0.1:9092"
	register := &kafka.ConsumerRegister{}
	register.Initialize(address)
	err := register.RegisterTopicAndHandler("test", "group1", TestData{}, func(data interface{}) error {
		dataValue := data.(*TestData)
		fmt.Printf("Msg : %s, Timestamp : %s \n", dataValue.Msg, dataValue.Timestamp)
		 return  nil
		 })
	if err != nil {
		fmt.Errorf("Failed register")
		println(err)
	}
	time.Sleep(1000 * time.Second)

	defer func() {
		register.Close()
	}()
}