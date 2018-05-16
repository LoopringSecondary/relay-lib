/*

  Copyright 2017 Loopring Project Ltd (Loopring Foundation).

  Licensed under the Apache License, Version 2.0 (the "License");
  you may not use this file except in compliance with the License.
  You may obtain a copy of the License at

  http://www.apache.org/licenses/LICENSE-2.0

  Unless required by applicable law or agreed to in writing, software
  distributed under the License is distributed on an "AS IS" BASIS,
  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
  See the License for the specific language governing permissions and
  limitations under the License.

*/

package kafka_test

import (
	"fmt"
	"github.com/Loopring/relay-lib/kafka"
	"strings"
	"testing"
	"time"
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
