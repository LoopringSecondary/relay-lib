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

package matrix_test

import (
	"encoding/json"
	"github.com/Loopring/relay-lib/broadcast/matrix"
	"github.com/Loopring/relay-lib/log"
	"go.uber.org/zap"
	"testing"
)

var matrixClient *matrix.MatrixClient

func init() {
	logConfig := `{
	  "level": "debug",
	  "development": false,
	  "encoding": "json",
	  "outputPaths": ["stdout"],
	  "errorOutputPaths": ["stderr"],
	  "encoderConfig": {
	    "messageKey": "message",
	    "levelKey": "level",
	    "levelEncoder": "lowercase",
	    "encodeTime": "iso8601"
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

	options := matrix.MatrixClientOptions{
		HSUrl:       "http://13.112.62.24:8008",
		User:        "root",
		Password:    "1",
		AccessToken: "MDAxN2xvY2F0aW9uIGxvY2FsaG9zdAowMDEzaWRlbnRpZmllciBrZXkKMDAxMGNpZCBnZW4gPSAxCjAwMjJjaWQgdXNlcl9pZCA9IEByb290OmxvY2FsaG9zdAowMDE2Y2lkIHR5cGUgPSBhY2Nlc3MKMDAyMWNpZCBub25jZSA9IHl3diwwV0VNWmZALFgsX14KMDAyZnNpZ25hdHVyZSDUtdBILxMd-_DHMLxlN-GwZYZkSwdYJJL9uWM8Wx1UeAo",
	}
	matrixClient, _ = matrix.NewMatrixClient(options)
}

func TestMatrixClient_Login(t *testing.T) {
	matrixClient := matrix.MatrixClient{}
	matrixClient.HSUrl = "http://13.112.62.24:8008"
	matrixClient.User = "root"
	matrixClient.Password = "1222"
	err := matrixClient.Login()

	if nil != err {
		t.Fatalf(err.Error())
	} else {
		t.Logf("accessToken:%s", matrixClient.LoginRes.AccessToken)
	}
}

func TestMatrixClient_JoinRoom(t *testing.T) {
	if err1 := matrixClient.JoinRoom("!RoJQgzCfBKHQznReRT:localhost"); nil != err1 {
		t.Error(err1.Error())
	}
}

func TestMatrixClient_RoomMessages(t *testing.T) {
	if _, err1 := matrixClient.RoomMessages("!RoJQgzCfBKHQznReRT:localhost", "t9-9_9_0_1_1_1_1_4_1", "", "", "", ""); nil != err1 {
		t.Error(err1.Error())
	}
}

func TestMatrixClient_SendMessages(t *testing.T) {
	if eventId,err1 := matrixClient.SendMessages("!RoJQgzCfBKHQznReRT:localhost", matrix.LoopringOrderType, "40", matrix.LoopringOrderType, "orderdata40"); nil != err1 {
		t.Error(err1.Error())
	} else {
		t.Log(eventId)
	}
}
