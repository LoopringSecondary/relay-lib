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

package broadcast_test

import (
	"encoding/json"
	"github.com/Loopring/relay-lib/broadcast"
	"github.com/Loopring/relay-lib/broadcast/matrix"
	"github.com/Loopring/relay-lib/cache"
	"github.com/Loopring/relay-lib/cache/redis"
	"github.com/Loopring/relay-lib/log"
	"github.com/Loopring/relay-lib/types"
	"go.uber.org/zap"
	"strings"
	"testing"
	"time"
	"math/rand"
	"math/big"
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

	cache.NewCache(redis.RedisOptions{Host: "127.0.0.1", Port: "6379", IdleTimeout: 20, MaxActive: 20, MaxIdle: 20})
}

func TestPubOrder(t *testing.T) {
	s := `{"protocol":"0x123456789012340F73A93993E5101362656Af116",
	"delegateAddress":"0x123456789012340F73A93993E5101362656Af116",
	"walletAddress":"0x123456789012340F73A93993E5101362656Af116",
"owner":"0x48ff2269e58a373120ffdbbdee3fbcea854ac30a",
"tokenB":"0xEF68e7C694F40c8202821eDF525dE3782458639f","tokenS":"0x2956356cD2a2bf3202F771F50D3D14A367b48070",
"authAddr":"0x90feb7c492db20afce48e830cc0c6bea1b6721dd",
"authPrivateKey":"acfe437a8e0f65124c44647737c0471b8adc9a0763f139df76766f46d6af8e15",
"amountB":"0x56bc75e2d63100000","amountS":"0x16345785d8a0000",
"lrcFee":"0xad78ebc5ac6200000",
"validSince":"0x5aa104a5",
"validUntil":"0x5ac891a5",
"marginSplitPercentage":35,"buyNoMoreThanAmountB":true,"walletId":"0x1","v":27,
"r":"0xbbc27e0aa7a3df3942ab7886b78d205d7bf8161abbece04e8d841f0de508522e","s":"0x2b19076f2fe24b58eedd00f0151d058bd7b1bf5fa38759c15902f03552492042"}`
	o := &types.Order{}
	if err := json.Unmarshal([]byte(s), o); nil != err {
		t.Fatalf(err.Error())
	}

	options := []matrix.MatrixPublisherOption{
		matrix.MatrixPublisherOption{
			MatrixClientOptions: matrix.MatrixClientOptions{
				HSUrl:       "http://13.112.62.24:8008",
				User:        "root",
				Password:    "1",
				AccessToken: "MDAxN2xvY2F0aW9uIGxvY2FsaG9zdAowMDEzaWRlbnRpZmllciBrZXkKMDAxMGNpZCBnZW4gPSAxCjAwMjJjaWQgdXNlcl9pZCA9IEByb290OmxvY2FsaG9zdAowMDE2Y2lkIHR5cGUgPSBhY2Nlc3MKMDAyMWNpZCBub25jZSA9IHl3diwwV0VNWmZALFgsX14KMDAyZnNpZ25hdHVyZSDUtdBILxMd-_DHMLxlN-GwZYZkSwdYJJL9uWM8Wx1UeAo",
			},
			Rooms: []string{"!RoJQgzCfBKHQznReRT:localhost"},
		},
	}

	if publishers, err := matrix.NewPublishers(options); nil != err {
		t.Errorf("err:%s", err.Error())
	} else {
		broadcast.Initialize(publishers, nil)
		seq := big.NewInt(rand.Int63())
		o.AmountS.Add(o.AmountS, seq)
		if errs := broadcast.PubOrder(strings.ToLower(o.GenerateHash().Hex()), s); nil != errs {
			t.Log(errs)
		}
	}
}

func TestSubOrderNext(t *testing.T) {
	options := []matrix.MatrixSubscriberOption{
		matrix.MatrixSubscriberOption{
			MatrixClientOptions: matrix.MatrixClientOptions{
				HSUrl:       "http://13.112.62.24:8008",
				User:        "root",
				Password:    "1",
				AccessToken: "MDAxN2xvY2F0aW9uIGxvY2FsaG9zdAowMDEzaWRlbnRpZmllciBrZXkKMDAxMGNpZCBnZW4gPSAxCjAwMjJjaWQgdXNlcl9pZCA9IEByb290OmxvY2FsaG9zdAowMDE2Y2lkIHR5cGUgPSBhY2Nlc3MKMDAyMWNpZCBub25jZSA9IHl3diwwV0VNWmZALFgsX14KMDAyZnNpZ25hdHVyZSDUtdBILxMd-_DHMLxlN-GwZYZkSwdYJJL9uWM8Wx1UeAo",
			},
			Rooms:     []string{"!RoJQgzCfBKHQznReRT:localhost"},
			CacheFrom: true,
			CacheTtl:  3600,
		},
	}
	if subscribers, err := matrix.NewSubscribers(options); nil != err {
		t.Errorf("err:%s", err.Error())
	} else {
		broadcast.Initialize(nil, subscribers)
		if orderChan, err := broadcast.SubOrderNext(); nil != err {
			t.Fatal(err.Error())
		} else {
			go func() {
				for {
					select {
					case dataI := <-orderChan:
						if data, ok := dataI.([]byte); ok {
							order := &types.Order{}
							if err := json.Unmarshal(data, order); nil != err {
								t.Errorf(err.Error())
							} else {
								t.Logf("hash:%s", order.GenerateHash().Hex())
							}
						}
					}
				}
			}()
		}
	}
	time.Sleep(10 * time.Second)
}
