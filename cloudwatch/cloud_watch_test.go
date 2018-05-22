package cloudwatch_test


import (
	"github.com/Loopring/relay-lib/cloudwatch"
	"testing"
	"time"
	"fmt"
)


func TestSender(t *testing.T) {
	cloudwatch.Initialize()
	for i := 0; i < 10; i++ {
		err := cloudwatch.PutResponseTimeMetric("hello", 100)
		if err != nil {
			fmt.Printf("Failed send metric data %s", err.Error())
		}
		time.Sleep(time.Second * 3)
	}
}