package kafka_test

import (
	"github.com/Loopring/relay-lib/kafka"
	"strings"
	"fmt"
	"time"
	"testing"
)

type TestData struct {
	Msg       string
	Timestamp string
}

func TestProducer(t *testing.T) {
	brokers := strings.Split("127.0.0.1:9092", ",")
	producerWrapped := &kafka.MessageProducer{}
	err := producerWrapped.Initialize(brokers)

	if err != nil {
		fmt.Printf("Failed init producerWrapped %s", err.Error())
	}

	for i := 0; i < 10; i++ {
		t := time.Now()
		data := &TestData{"msg", t.Format(time.RFC3339)}
		partition, offset, sendRes := producerWrapped.SendMessage("test", data, "1")
		if sendRes != nil {
			fmt.Errorf("failed to sendmsg : %s", err.Error())
		} else {
			fmt.Printf("Your data is stored with unique identifier important/%d/%d\n", partition, offset)
		}
		time.Sleep(time.Second * 3)
	}

	defer func() {
		producerWrapped.Close()
	}()

}