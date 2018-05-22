package cloudwatch_test


import (
	"github.com/Loopring/relay-lib/cloudwatch"
	"testing"
	"time"
	"fmt"
)


func TestHeartBeatMetric(t *testing.T) {
	cloudwatch.Initialize()
	for i := 0; i < 10; i++ {
		err := cloudwatch.PutHeartBeatMetric("cronJob")
		if err != nil {
			fmt.Printf("Failed send metric data %s", err.Error())
		}
		time.Sleep(time.Second * 3)
	}
	cloudwatch.Close()
}