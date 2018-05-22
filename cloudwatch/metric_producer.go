package cloudwatch

import (
	"fmt"
	"github.com/Loopring/relay-lib/log"
	"github.com/Loopring/relay-lib/utils"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"net"
	"time"
)

const region = "ap-northeast-1"
const namespace = "LoopringDefine"
const obsoleteThreshold = 1000

var cwc *cloudwatch.CloudWatch
var inChan chan<- interface{}
var outChan <-chan interface{}

func Initialize() error {
	//NOTE: use default config ~/.asw/credentials
	sess, err := session.NewSession(&aws.Config{
		Region:      aws.String(region),
		Credentials: credentials.NewSharedCredentials("", ""),
	})
	if err != nil {
		return err
	} else {
		cwc = cloudwatch.New(sess)
		inChan, outChan = utils.MakeInfinite()
		go func() {
			var obsoleteTimes uint32
			for {
				select {
				case data, ok := <-outChan:
					if !ok {
						log.Error("receive from watchcloud output channel failed")
					} else {
						inputData, ok := data.(*cloudwatch.PutMetricDataInput)
						if !ok {
							log.Error("convert data to PutMetricDataInput failed")
						} else {
							if checkObsolete(inputData) {
								obsoleteTimes += 1
								if obsoleteTimes >= obsoleteThreshold {
									log.Errorf("obsolete cloud watch metric data count is %d, just drop\n", obsoleteTimes)
									obsoleteTimes = 0
								}
							} else {
								cwc.PutMetricData(inputData)
								if obsoleteTimes > 0 {
									log.Errorf("Drop %d obsolete cloud watch metric data\n", obsoleteTimes)
									obsoleteTimes = 0
								}
							}
						}
					}
				}
			}
		}()
		return nil
	}
}

func Close() {
	close(inChan)
}

func IsValid() bool {
	return cwc != nil
}

func PutResponseTimeMetric(methodName string, costTime float64) error {
	if !IsValid() {
		return fmt.Errorf("Cloudwatch client has not initialized\n")
	}
	dt := &cloudwatch.MetricDatum{}
	metricName := fmt.Sprintf("response_%s", methodName)
	dt.MetricName = &metricName
	dt.Dimensions = []*cloudwatch.Dimension{}
	dt.Dimensions = append(dt.Dimensions, hostDimension())
	dt.Value = &costTime
	unit := cloudwatch.StandardUnitMilliseconds
	dt.Unit = &unit
	tms := time.Now()
	dt.Timestamp = &tms

	datums := []*cloudwatch.MetricDatum{}
	datums = append(datums, dt)

	input := &cloudwatch.PutMetricDataInput{}
	input.MetricData = datums
	input.Namespace = namespaceNormal()

	return storeMetricLocal(input)
}

func PutHeartBeatMetric(metricName string) error {
	if !IsValid() {
		return fmt.Errorf("Cloudwatch client has not initialized\n")
	}
	dt := &cloudwatch.MetricDatum{}
	dt.MetricName = &metricName
	dt.Dimensions = []*cloudwatch.Dimension{}
	dt.Dimensions = append(dt.Dimensions, globalDimension())
	hearbeatValue := 1.0
	dt.Value = &hearbeatValue
	unit := cloudwatch.StandardUnitCount
	dt.Unit = &unit
	tms := time.Now()
	dt.Timestamp = &tms

	datums := []*cloudwatch.MetricDatum{}
	datums = append(datums, dt)

	input := &cloudwatch.PutMetricDataInput{}
	input.MetricData = datums
	input.Namespace = namespaceNormal()

	return storeMetricLocal(input)
}

func storeMetricLocal(input *cloudwatch.PutMetricDataInput) error {
	inChan <- input
	return nil
}

func checkObsolete(input *cloudwatch.PutMetricDataInput) bool {
	return time.Now().UnixNano()-input.MetricData[0].Timestamp.UnixNano() > int64(1000*1000)
}

func namespaceNormal() *string {
	sp := namespace
	return &sp
}

func hostDimension() *cloudwatch.Dimension {
	dim := &cloudwatch.Dimension{}
	ipDimName := "host"
	dim.Name = &ipDimName
	dim.Value = getIp()
	return dim
}

func globalDimension() *cloudwatch.Dimension {
	dim := &cloudwatch.Dimension{}
	dimName := "global"
	dim.Name = &dimName
	dimValue := "nt"
	dim.Value = &dimValue
	return dim
}

func getIp() *string {
	var res = "unknown"
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return &res
	}
	for _, a := range addrs {
		if ipnet, ok := a.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				res = ipnet.IP.To4().String()
			}
		}
	}
	return &res
}
