package zklock_test

import (
	"encoding/json"
	"github.com/Loopring/relay-lib/log"
	"github.com/Loopring/relay-lib/zklock"
	"go.uber.org/zap"
	"testing"
	"time"
)


func TestConfig(t *testing.T) {
	initLog()
	config := zklock.ZkLockConfig{}
	config.ZkServers = "127.0.0.1:2181"
	config.ConnectTimeOut = 2
	zklock.Initialize(config)
	zklock.RegisterConfigHandler("demoService", "name", func(data []byte) error {
		str := string(data[:])
		log.Infof("data changed, new data is : %s", str)
		return nil
	})
	time.Sleep(1000 * time.Second)
}


func initLog() {
	logConfig := `{
	  "level": "debug",
	  "development": false,
	  "encoding": "json",
	  "outputPaths": ["stdout"],
	  "errorOutputPaths": ["stderr"],
	  "encoderConfig": {
	    "messageKey": "message",
	    "levelKey": "level",
	    "levelEncoder": "lowercase"
	  }
	}`
	rawJSON := []byte(logConfig)
	var (
		cfg zap.Config
		err error
	)
	if err = json.Unmarshal(rawJSON, &cfg); err != nil {
		panic(err)
	}
	log.Initialize(cfg)
}