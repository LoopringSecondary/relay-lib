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

package marketcap_test

import (
	"encoding/json"
	"github.com/Loopring/relay-lib/cache"
	"github.com/Loopring/relay-lib/cache/redis"
	"github.com/Loopring/relay-lib/log"
	"github.com/Loopring/relay-lib/marketcap"
	"github.com/Loopring/relay-lib/marketutil"
	"github.com/Loopring/relay-lib/zklock"
	"github.com/ethereum/go-ethereum/common"
	"go.uber.org/zap"
	"math/big"
	"testing"
	"time"
)

func init() {
	logConfig := `{
	  "level": "debug",
	  "development": false,
	  "encoding": "json",
	  "outputPaths": ["zap.log","stderr"],
	  "errorOutputPaths": ["err.log"],
	  "encoderConfig": {
	    "messageKey": "message",
	    "levelKey": "level",
	    "levelEncoder": "lowercase",
	    "encodeTime": "iso8601",
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

	options := marketutil.MarketOptions{}
	options.TokenFile = "/Users/yuhongyu/Desktop/service/go/src/github.com/Loopring/relay/config/tokens.json"
	marketutil.Initialize(&options)

	zkconfig := zklock.ZkLockConfig{}
	zkconfig.ZkServers = "127.0.0.1:2181"
	zkconfig.ConnectTimeOut = 10000
	zklock.Initialize(zkconfig)
}

func TestCapProvider_CoinMarketCap_Start(t *testing.T) {
	options := marketcap.MarketCapOptions{}
	options.BaseUrl = "https://api.coinmarketcap.com/v1/ticker/?limit=0&convert=%s"
	options.Duration = 5
	options.Currency = "USD"
	options.IsSync = false
	provider := marketcap.NewMarketCapProvider(&options)
	provider.Start()
	a := new(big.Rat)
	a.SetString("2806326640744990")
	f, err := provider.LegalCurrencyValueByCurrency(common.HexToAddress("0xBeB6fdF4ef6CEb975157be43cBE0047B248a8922"), a, "USD")
	if nil != err {
		t.Errorf(err.Error())
	} else {
		println(f.FloatString(2))
	}

	time.Sleep(11 * time.Second)
}
