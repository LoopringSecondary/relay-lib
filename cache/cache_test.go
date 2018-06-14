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

package cache_test

import (
	"encoding/json"
	"github.com/Loopring/relay-lib/cache"
	"github.com/Loopring/relay-lib/cache/redis"
	"github.com/Loopring/relay-lib/log"
	"github.com/naoina/toml"
	"go.uber.org/zap"
	"os"
	"testing"
	"time"
)

func init() {
	//logConfig := `{
	//  "level": "debug",
	//  "development": false,
	//  "encoding": "json",
	//  "outputPaths": ["stdout"],
	//  "errorOutputPaths": ["stderr"],
	//  "encoderConfig": {
	//    "messageKey": "message",
	//    "levelKey": "level",
	//    "levelEncoder": "lowercase",
	//    "encodeTime": "iso8601"
	//  }
	//}`
	//rawJSON := []byte(logConfig)
	//
	//var (
	//	cfg zap.Config
	//	err error
	//)
	//if err = json.Unmarshal(rawJSON, &cfg); err != nil {
	//	panic(err)
	//}
	//cfg.DisableStacktrace = false

	type Config struct {
		Log zap.Config
	}
	io, err := os.Open("log/log_test.toml")
	if err != nil {
		panic(err)
	}
	defer io.Close()

	c := &Config{}
	if err := toml.NewDecoder(io).Decode(c); err != nil {
		panic(err)
	}
	log.Initialize(c.Log)

	cache.NewCache(redis.RedisOptions{Host: "127.0.0.1", Port: "6379", IdleTimeout: 20, MaxActive: 20, MaxIdle: 20})
}

type User struct {
	Id   int64  `json:"id"`
	Name string `json:"name"`
}

func TestGet(t *testing.T) {
	cacheKey := "conn_test"
	for i := 1; i <= 100; i++ {
		user := &User{
			Id:   int64(i + 100),
			Name: "name1",
		}
		if data, err := json.Marshal(user); nil != err {
			log.Debugf("err:%s", err.Error())
		} else {
			go func(data []byte) {
				defer func() {
					if r := recover(); r != nil {
						println(r)
						log.Info("llllllllllll")
						//log.Errorf("Recovered in f", r)
						//log.Errorf("Recovered in f", r)
					}
				}()
				f(cacheKey, data)
				f1(cacheKey, data)
				f2()
				//time.Sleep(5 * time.Second)
			}(data)
		}
	}

	time.Sleep(10 * time.Second)
}

func f(cacheKey string, data []byte) {
	if err := cache.SAdd(cacheKey, int64(10000), data); nil != err {
		log.Errorf("err:%s", err.Error())
	}
}

func f1(cacheKey string, data []byte) {
	if err := cache.SAdd(cacheKey, int64(10000), data); nil != err {
		log.Errorf("err:%s", err.Error())
	}
}

func f2() {
	panic("eeeeeeee")
}
