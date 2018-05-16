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

package gasprice_evaluator

import (
	"github.com/Loopring/relay-lib/eth/accessor"
	ethtyp "github.com/Loopring/relay-lib/eth/types"
	"github.com/Loopring/relay-lib/log"
	"github.com/Loopring/relay-lib/types"
	"math/big"
	"sort"
	"github.com/Loopring/relay-lib/cache"
	"github.com/Loopring/relay-lib/zklock"
)

const (
	CacheKey_Evaluated_GasPrice = "evaluated_gasprice_"
	ZkName_Evaluated_GasPrice = "evaluated_gasprice"
)

var priceEvaluator *GasPriceEvaluator

type GasPriceEvaluator struct {
	Blocks []*ethtyp.BlockWithTxAndReceipt

	gasPrice *big.Int
	stopChan chan bool
}

func (e *GasPriceEvaluator) GasPrice(minGasPrice, maxGasPrice *big.Int) *big.Int {
	gasPrice := new(big.Int)
	if nil != e.gasPrice {
		if nil != maxGasPrice && maxGasPrice.Cmp(e.gasPrice) < 0 {
			gasPrice.Set(maxGasPrice)
		} else if nil != minGasPrice && minGasPrice.Cmp(e.gasPrice) > 0 {
			gasPrice.Set(minGasPrice)
		} else {
			gasPrice.Set(e.gasPrice)
		}
	} else {
		gasPrice.Set(maxGasPrice)
	}
	return gasPrice
}

func (e *GasPriceEvaluator) start() {
	zkLock,_ := zklock.NewLock(zklock.ZkLockConfig{})
	zkLock.TryLock(ZkName_Evaluated_GasPrice)

	var blockNumber types.Big
	if err := accessor.BlockNumber(&blockNumber); nil == err {
		go func() {
			number := new(big.Int).Set(blockNumber.BigInt())
			number.Sub(number, big.NewInt(30))
			iterator := accessor.NewBlockIterator(number, nil, true, uint64(0))
			for {
				select {
				case <-e.stopChan:
					return
				default:
					blockInterface, err := iterator.Next()
					if nil == err {
						blockWithTxAndReceipt := blockInterface.(*ethtyp.BlockWithTxAndReceipt)
						log.Debugf("gasPriceEvaluator, blockNumber:%s, gasPrice:%s", blockWithTxAndReceipt.Number.BigInt().String(), e.gasPrice.String())
						e.Blocks = append(e.Blocks, blockWithTxAndReceipt)
						if len(e.Blocks) > 30 {
							e.Blocks = e.Blocks[1:]
						}
						var prices gasPrices = []*big.Int{}
						for _, block := range e.Blocks {
							for _, tx := range block.Transactions {
								prices = append(prices, tx.GasPrice.BigInt())
							}
						}
						e.gasPrice = prices.bestGasPrice()
						cache.Set(CacheKey_Evaluated_GasPrice, int64(0), []byte(e.gasPrice.String()))
					}
				}
			}
		}()

	}
}

func (e *GasPriceEvaluator) stop() {
	e.stopChan <- true
}

type gasPrices []*big.Int

func (prices gasPrices) Len() int {
	return len(prices)
}

func (prices gasPrices) Swap(i, j int) {
	prices[i], prices[j] = prices[j], prices[i]
}

func (prices gasPrices) Less(i, j int) bool {
	return prices[i].Cmp(prices[j]) > 0
}

func (prices gasPrices) bestGasPrice() *big.Int {
	sort.Sort(prices)
	startIdx := 0
	endIdx := (len(prices) / 3) * 2

	averagePrice := big.NewInt(0)
	for _, price := range prices[startIdx:endIdx] {
		averagePrice.Add(averagePrice, price)
	}
	averagePrice.Div(averagePrice, big.NewInt(int64(endIdx-startIdx+1)))

	if averagePrice.Cmp(big.NewInt(int64(0))) <= 0 {
		averagePrice = big.NewInt(int64(1000000000))
	}
	return averagePrice
}

func InitGasPriceEvaluator() {
	if nil != priceEvaluator {
		priceEvaluator.stop()
	}
	priceEvaluator = &GasPriceEvaluator{}
	priceEvaluator.start()
}

func EstimateGasPrice(minGasPrice, maxGasPrice *big.Int) *big.Int {
	if data,err := cache.Get(CacheKey_Evaluated_GasPrice); nil == err {
		return new(big.Int).SetString(string(data), 10)
	} else {
		InitGasPriceEvaluator()
		return big.NewInt(int64(1000000000))
	}
	//return priceEvaluator.GasPrice(minGasPrice, maxGasPrice)
}
