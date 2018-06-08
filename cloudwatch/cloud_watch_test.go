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

package cloudwatch_test

import (
	"fmt"
	"github.com/Loopring/relay-lib/cloudwatch"
	"testing"
	"time"
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
