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

package loopringaccessor_test

import (
	"encoding/json"
	"github.com/Loopring/relay-lib/cache"
	"github.com/Loopring/relay-lib/cache/redis"
	"github.com/Loopring/relay-lib/eth/accessor"
	"github.com/Loopring/relay-lib/eth/loopringaccessor"
	"github.com/Loopring/relay-lib/log"
	"github.com/ethereum/go-ethereum/common"
	"go.uber.org/zap"
	"testing"
)

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
	cache.NewCache(redis.RedisOptions{Host: "127.0.0.1", Port: "6379"})

	accessor.Initialize(accessor.AccessorOptions{RawUrls: []string{"http://13.230.23.98:8545"}})

	options := loopringaccessor.LoopringProtocolOptions{}
	options.Address = make(map[string]string)
	options.Address["1.5"] = "0x8d8812b72d1e4ffCeC158D25f56748b7d67c1e78"

	if err := loopringaccessor.Initialize(options); nil != err {
		log.Fatal(err.Error())
	}
}

func TestInitLoopringAccessor(t *testing.T) {
	addrs := loopringaccessor.DelegateAddresses()
	for addr, _ := range addrs {
		t.Log(addr.Hex())
	}
}

func TestBatchErc20Balance(t *testing.T) {
	reqs := loopringaccessor.BatchBalanceReqs{}
	req1 := &loopringaccessor.BatchBalanceReq{}
	req1.BlockParameter = "latest"
	req1.Token = common.HexToAddress("0x")
	req1.Owner = common.HexToAddress("0x3acdf3e3d8ec52a768083f718e763727b0210650")

	req2 := &loopringaccessor.BatchBalanceReq{}
	req2.BlockParameter = "latest"
	req2.Token = common.HexToAddress("0xef68e7c694f40c8202821edf525de3782458639f")
	req2.Owner = common.HexToAddress("0x3acdf3e3d8ec52a768083f718e763727b0210650")

	reqs = append(reqs, req1, req2)

	loopringaccessor.BatchErc20Balance("latest", reqs)

	for _, req := range reqs {
		t.Logf("owner:%s, token:%s, balance:%s", req.Owner.Hex(), req.Token.Hex(), req.Balance.BigInt().String())
	}

}
