package kafka

import (
	"github.com/bsm/sarama-cluster"
	"fmt"
	"strings"
	"encoding/json"
	"reflect"
)

type ConsumerRegister struct {
	brokers     []string
	conf        *cluster.Config
	consumerMap map[string]map[string]*cluster.Consumer
}

type HandlerFunc func(event interface{})(error)

func (cr *ConsumerRegister) Initialize(address string) {
	config := cluster.NewConfig()
	config.Consumer.Return.Errors = true
	config.Group.Return.Notifications = true
	cr.conf = config
	cr.brokers = strings.Split(address, ",")
	cr.consumerMap = make(map[string]map[string]*cluster.Consumer) //map[topic][groupId]
}

func (cr *ConsumerRegister) RegisterTopicAndHandler(topic string, groupId string, data interface{}, action HandlerFunc) (error) {
	groupConsumerMap, ok := cr.consumerMap[topic]
	if ok {
		_, ok1 := groupConsumerMap[groupId]
		if ok1 {
			return fmt.Errorf("consumer alreay registered !!")
		}
	} else {
		cr.consumerMap[topic] = make(map[string]*cluster.Consumer)
	}
	consumer, err := cluster.NewConsumer(cr.brokers, groupId, []string{topic}, cr.conf)
	if err != nil {
		panic(err)
	}

	go func() {
		for err := range consumer.Errors() {
			fmt.Printf("Error: %s\n", err.Error())
		}
	}()

	// consume notifications
	go func() {
		for ntf := range consumer.Notifications() {
			fmt.Printf("Notification : %+v\n", ntf)
		}
	}()

	cr.consumerMap[topic][groupId] = consumer
	go func() {
		for {
			select {
			case msg, ok := <-consumer.Messages():
				if ok {
					data := (reflect.New(reflect.TypeOf(data))).Interface()
					json.Unmarshal(msg.Value, data)
					action(data)
					consumer.MarkOffset(msg, "") // mark message as processed
				}
			}
		}
	}()

	return nil
}

func (cr *ConsumerRegister) Close() {
	for _, mp := range cr.consumerMap {
		for _, cm := range mp {
			cm.Close()
		}
	}
}