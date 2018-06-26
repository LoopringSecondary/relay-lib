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

package marketutil_test

import (
	"fmt"
	"github.com/Loopring/relay-lib/cache"
	"github.com/Loopring/relay-lib/cache/redis"
	"github.com/Loopring/relay-lib/marketutil"
	"github.com/Loopring/relay-lib/types"
	"github.com/ethereum/go-ethereum/common"
	"math/big"
	"testing"
	//"time"
	"github.com/Loopring/relay-lib/zklock"
	"strings"
	"time"
)

func TestGetCustomTokenList(t *testing.T) {

	zkconfig := zklock.ZkLockConfig{}
	zkconfig.ZkServers = "13.112.62.24:2181"
	zkconfig.ConnectTimeOut = 10000
	zklock.Initialize(zkconfig)

	marketutil.SupportTokens = make(map[string]types.Token)
	marketutil.AllTokens = make(map[string]types.Token)
	funToken := types.Token{Protocol: common.HexToAddress("0x419D0d8BdD9aF5e606Ae2232ed285Aff190E711b"), Decimals: big.NewInt(1e8), Symbol: "FUN"}
	wethToken := types.Token{Protocol: common.HexToAddress("0x2956356cD2a2bf3202F771F50D3D14A367b48070"), Decimals: big.NewInt(1e18), Symbol: "WETH"}
	marketutil.SupportTokens["FUN"] = funToken
	marketutil.AllTokens["FUN"] = funToken
	marketutil.AllTokens["WETH"] = wethToken

	cache.NewCache(redis.RedisOptions{Host: "13.112.62.24", Port: "6379", Password: "", IdleTimeout: 20, MaxIdle: 50, MaxActive: 50})

	address := common.HexToAddress("0X0B4C35F76AF1ACEBF52A8513255AE3FDA2E9D42C")
	//token := common.HexToAddress("0xbf78B6E180ba2d1404c92Fc546cbc9233f616C43")
	//symbol := "testX"
	decimals := new(big.Int)
	decimals.SetString("1"+strings.Repeat("0", 6), 0)
	//err := marketutil.AddToken(address, marketutil.CustomToken{Symbol:symbol, Address:token, Decimals:decimals})

	err := marketutil.AddToken(address, marketutil.CustomToken{Symbol: "LRN", Address: common.HexToAddress("0x1111B6E180ba2d1404c92Fc546cbc9233f616C43"), Decimals: decimals})
	fmt.Println(err)
	time.Sleep(1 * time.Second)
	fmt.Println("---------->")
	fmt.Println(marketutil.HadRegisted(common.HexToAddress("0x1111B6E180ba2d1404c92Fc546cbc9233f616C43")))
	fmt.Println(marketutil.HadRegisted(common.HexToAddress("0x1111B6E180ba2d1404c92Fc546cbc9233f616C44")))
	fmt.Println(marketutil.HadRegistedByAddress(address, common.HexToAddress("0x1111B6E180ba2d1404c92Fc546cbc9233f616C43")))
	fmt.Println(marketutil.HadRegistedByAddress(common.HexToAddress("0X0B4C35F76AF1ACEBF52A8513255AE3FDA2E9D42B"), common.HexToAddress("0x1111B6E180ba2d1404c92Fc546cbc9233f616C43")))

	tokens, err := marketutil.GetAllCustomTokenList()
	for k, v := range tokens {
		fmt.Printf("k : %s, kk : %s, v : %s, d : %d\n", k, v.Symbol, v.Address.Hex(), v.Decimals.Int64())
	}
	fmt.Println(err)

}
