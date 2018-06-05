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

package zklock_test

import (
	"encoding/json"
	"fmt"
	"github.com/Loopring/relay-lib/log"
	"github.com/Loopring/relay-lib/zklock"
	"go.uber.org/zap"
	"testing"
	"time"
)

func TestProducer(t *testing.T) {
	initLog()
	config := zklock.ZkLockConfig{}
	config.ZkServers = "127.0.0.1:2181"
	config.ConnectTimeOut = 2
	zklock.Initialize(config)
	runBalancer()
}

func runBalancer() {
	balancer := zklock.ZkBalancer{}
	if err := balancer.Init("test", buildTasks()); err != nil {
		log.Errorf("init balancer failed : %s", err.Error())
	}
	var localTasks map[string]zklock.Task
	var rmTasks []zklock.Task
	balancer.OnAssign(func(newAssignedTasks []zklock.Task) error {
		localTasks, rmTasks = splitTasks(localTasks, newAssignedTasks)
		balancer.Released(rmTasks)
		log.Infof("balancer release tasks %+v", rmTasks)
		log.Infof("balancer maintain tasks %+v", localTasks)
		return nil
	})
	balancer.Start()
	time.Sleep(time.Second * 300)
	balancer.Stop()
}

func splitTasks(local map[string]zklock.Task, newAssigned []zklock.Task) (map[string]zklock.Task, []zklock.Task) {
	newAssignedMap := make(map[string]zklock.Task)
	if len(newAssigned) == 0 {
		rmTasks := make([]zklock.Task, len(local))
		for _, v := range local {
			rmTasks = append(rmTasks, v)
		}
		return newAssignedMap, rmTasks
	} else {
		for _, v := range newAssigned {
			newAssignedMap[v.Path] = v
		}
		removedTasks := make([]zklock.Task, len(local))
		for _, t := range local {
			if v, ok := newAssignedMap[t.Path]; !ok {
				removedTasks = append(removedTasks, v)
			}
		}
		return newAssignedMap, removedTasks
	}
}

func buildTasks() []zklock.Task {
	res := make([]zklock.Task, 0, 10)
	for i := 0; i < 10; i++ {
		task := zklock.Task{Payload: fmt.Sprintf("payload-%d", i), Path: fmt.Sprintf("task%d", i), Weight: i, Status: zklock.Init, Owner: "", Timestamp: 0}
		res = append(res, task)
	}
	return res
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
