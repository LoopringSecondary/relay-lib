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

package marketcap

import (
	"math/big"
	"strings"
	"time"
	"github.com/Loopring/relay-lib/zklock"
	"io/ioutil"

	"encoding/json"
	"errors"
	"fmt"
	"github.com/Loopring/relay-lib/cache"
	"github.com/Loopring/relay-lib/cloudwatch"
	"github.com/Loopring/relay-lib/log"
	util "github.com/Loopring/relay-lib/marketutil"
	"github.com/Loopring/relay-lib/types"
	"github.com/ethereum/go-ethereum/common"
	"net/http"
	"strconv"
)


const (
	CACHEKEY_COIN_MARKETCAP  = "coin_marketcap_"
	ZKNAME_COIN_MARKETCAP    = "coin_marketcap_"
	HEARTBEAT_COIN_MARKETCAP = "coin_marketcap"
)

type MarketCap struct {
	Price *big.Rat
	Volume24H *big.Rat
	MarketCap *big.Rat
	PercentChange1H *big.Rat
	PercentChange24H *big.Rat
	PercentChange7D *big.Rat
}

func (cap *MarketCap) UnmarshalJSON(input []byte) error {
	type Cap struct {
		Price     string `json:"price"`
		Volume24H string `json:"24h_volume"`
		MarketCap string `json:"market_cap"`
		PercentChange1H string `json:"percent_change_1h"`
		PercentChange24H string `json:"percent_change_24h"`
		PercentChange7D string `json:"percent_change_7d"`
	}
	c := &Cap{}
	if err := json.Unmarshal(input, c); nil != err {
		return err
	} else {
		if price, err1 := strconv.ParseFloat(c.Price, 10); nil != err1 {
			return err1
		} else {
			cap.Price = new(big.Rat).SetFloat64(price)
		}
		if volume, err1 := strconv.ParseFloat(c.Volume24H, 10); nil != err1 {
			return err1
		} else {
			cap.Volume24H = new(big.Rat).SetFloat64(volume)
		}
		if marketCap, err1 := strconv.ParseFloat(c.MarketCap, 10); nil != err1 {
			return err1
		} else {
			cap.MarketCap = new(big.Rat).SetFloat64(marketCap)
		}
		if change1H, err1 := strconv.ParseFloat(c.PercentChange1H, 10); nil != err1 {
			return err1
		} else {
			cap.PercentChange1H = new(big.Rat).SetFloat64(change1H)
		}
		if change24H, err1 := strconv.ParseFloat(c.PercentChange24H, 10); nil != err1 {
			return err1
		} else {
			cap.PercentChange24H = new(big.Rat).SetFloat64(change24H)
		}
		if change7D, err1 := strconv.ParseFloat(c.PercentChange7D, 10); nil != err1 {
			return err1
		} else {
			cap.PercentChange7D = new(big.Rat).SetFloat64(change7D)
		}
	}
	return nil
}

type CoinMarketCap struct {
	Id int `json:"id"`
	Address      common.Address `json:"address"`
	Name string `json:"name"`
	Symbol string `json:"name"`
	WebsiteSlug string `json:"website_slug"`
	Rand int `json:"rank"`
	CirculatingSupply float64 `json:"circulating_supply"`
	TotalSupply float64 `json:"total_supply"`
	MaxSupply float64 `json:"max_supply"`
	Quotes map[string]*MarketCap `json:"quotes"`
	LastUpdated int64 `json:"last_updated"`
	Decimals     *big.Int
}
type CoinMarketCapResult struct {
	Data map[string]CoinMarketCap `json:"data"`
	Metadata struct{
		Timestamp int64 `json:"timestamp"`
		NumCryptocurrencies int `json:"num_cryptocurrencies"`
		Error string `json:"error"`
		 } `json:"metadata"`
}

type CapProvider_CoinMarketCap struct {
	baseUrl         string
	tokenMarketCaps map[common.Address]*CoinMarketCap
	idToAddress     map[string]common.Address
	currency        string
	duration        int
	dustValue       *big.Rat
	stopFuncs       []func()
}

func (p *CapProvider_CoinMarketCap) LegalCurrencyValue(tokenAddress common.Address, amount *big.Rat) (*big.Rat, error) {
	return p.LegalCurrencyValueByCurrency(tokenAddress, amount, p.currency)
}

func (p *CapProvider_CoinMarketCap) LegalCurrencyValueOfEth(amount *big.Rat) (*big.Rat, error) {
	tokenAddress := util.AllTokens["WETH"].Protocol
	return p.LegalCurrencyValueByCurrency(tokenAddress, amount, p.currency)
}

func (p *CapProvider_CoinMarketCap) LegalCurrencyValueByCurrency(tokenAddress common.Address, amount *big.Rat, currencyStr string) (*big.Rat, error) {
	if c, exists := p.tokenMarketCaps[tokenAddress]; !exists {
		return nil, errors.New("not found tokenCap:" + tokenAddress.Hex())
	} else {
		v := new(big.Rat).SetInt(c.Decimals)
		v.Quo(amount, v)
		price, _ := p.GetMarketCapByCurrency(tokenAddress, currencyStr)
		//log.Debugf("LegalCurrencyValueByCurrency token:%s,decimals:%s, amount:%s, currency:%s, price:%s", tokenAddress.Hex(), c.Decimals.String(), amount.FloatString(2), currencyStr, price.FloatString(2) )
		v.Mul(price, v)
		return v, nil
	}
}

func (p *CapProvider_CoinMarketCap) GetMarketCap(tokenAddress common.Address) (*big.Rat, error) {
	return p.GetMarketCapByCurrency(tokenAddress, p.currency)
}

func (p *CapProvider_CoinMarketCap) GetEthCap() (*big.Rat, error) {
	return p.GetMarketCapByCurrency(util.AllTokens["WETH"].Protocol, p.currency)
}

func (p *CapProvider_CoinMarketCap) GetMarketCapByCurrency(tokenAddress common.Address, currencyStr string) (*big.Rat, error) {
	currency := StringToLegalCurrency(currencyStr)
	if c, exists := p.tokenMarketCaps[tokenAddress]; exists {
		var v *big.Rat
		switch currency {
		case CNY:
			v = c.PriceCny
		case USD:
			v = c.PriceUsd
		case BTC:
			v = c.PriceBtc
		}
		if "VITE" == c.Symbol || "ARP" == c.Symbol {
			wethCap, _ := p.GetMarketCapByCurrency(util.AllTokens["WETH"].Protocol, currencyStr)
			v = wethCap.Mul(wethCap, util.AllTokens[c.Symbol].IcoPrice)
		}
		if v == nil {
			return nil, errors.New("tokenCap is nil")
		} else {
			return new(big.Rat).Set(v), nil
		}
	} else {
		err := errors.New("not found tokenCap:" + tokenAddress.Hex())
		res := new(big.Rat).SetInt64(int64(1))
		if nil != err {
			log.Errorf("get MarketCap of token:%s, occurs error:%s. the value will be default value:%s", tokenAddress.Hex(), err.Error(), res.String())
		}
		return res, err
	}
}

func (p *CapProvider_CoinMarketCap) Stop() {
	for _, f := range p.stopFuncs {
		f()
	}
}

func (p *CapProvider_CoinMarketCap) Start() {
	stopChan := make(chan bool)
	p.stopFuncs = append(p.stopFuncs, func() {
		stopChan <- true
	})
	go func() {
		for {
			select {
			case <-time.After(time.Duration(p.duration) * time.Minute):
				log.Debugf("sync marketcap from redis...")
				if err := p.syncMarketCapFromRedis(); nil != err {
					log.Errorf("can't sync marketcap, time:%d", time.Now().Unix())
				}
			case stopped := <-stopChan:
				if stopped {
					return
				}
			}
		}
	}()

	go p.syncMarketCapFromAPIWithZk()
}

func (p *CapProvider_CoinMarketCap) zklockName() string {
	return ZKNAME_COIN_MARKETCAP + p.currency
}

func (p *CapProvider_CoinMarketCap) cacheKey(addr common.Address) string {
	return CACHEKEY_COIN_MARKETCAP + strings.ToLower(addr.Hex())
}

func (p *CapProvider_CoinMarketCap) syncMarketCapFromAPIWithZk() {
	//todo:
	zklock.TryLock(p.zklockName())
	log.Debugf("syncMarketCapFromAPIWithZk....")
	stopChan := make(chan bool)
	p.stopFuncs = append(p.stopFuncs, func() {
		stopChan <- true
	})

	go func() {
		for {
			select {
			case <-time.After(time.Duration(p.duration) * time.Minute):
				log.Debugf("sync marketcap from api...")
				p.syncMarketCapFromAPI()
				if err := cloudwatch.PutHeartBeatMetric(HEARTBEAT_COIN_MARKETCAP); nil != err {
					log.Errorf("err:%s", err.Error())
				}
			case stopped := <-stopChan:
				if stopped {
					zklock.ReleaseLock(p.zklockName())
					return
				}
			}
		}
	}()
}

func (p *CapProvider_CoinMarketCap) syncMarketCapFromAPI() error {
	log.Debugf("syncMarketCapFromAPI...")
	//https://api.coinmarketcap.com/v2/ticker/?convert=%s&start=%d&limit=%d
	numCryptocurrencies := 105
	start := 0
	limit := 100
	for numCryptocurrencies > start * limit {
		url := fmt.Sprintf(p.baseUrl, p.currency, start, limit)
		resp, err := http.Get(url)
		if err != nil {
			log.Errorf("err:%s", err.Error())
			return []byte{}, err
		}
		defer func() {
			if nil != resp && nil != resp.Body {
				resp.Body.Close()
			}
		}()

		body, err := ioutil.ReadAll(resp.Body)
		if nil != err {
			log.Errorf("err:%s", err.Error())
			return err
		} else {
			result := &CoinMarketCapResult{}
			if err1 := json.Unmarshal(body, result); nil != err1 {
				log.Errorf("err:%s", err1.Error())
				return err1
			} else {

				if "" == result.Metadata.Error {
					for _,cap1 := range result.Data {
						if data,err2 := json.Marshal(cap1); nil != err2 {
							log.Errorf("err:%s", err2.Error())
							return err2
						} else {
							err = cache.Set(p.cacheKey(cap1.Address), data, int64(0))
							if nil != err {
								log.Errorf("err:%s", err.Error())
								return err
							}
						}
					}
					numCryptocurrencies = result.Metadata.NumCryptocurrencies - (start * limit)
					start = start + 1
				} else {
					log.Errorf("err:%s", result.Metadata.Error)
				}
			}
		}
		return err
	}
	return nil
}

func (p *CapProvider_CoinMarketCap) syncMarketCapFromRedis() error {
	//todo:use zk to keep
	tokenMarketCaps := make(map[common.Address]*CoinMarketCap)
	for addr,c1 := range p.tokenMarketCaps {
		data,err := cache.Get(p.cacheKey(addr))
		if nil != err {
			p.syncMarketCapFromAPI()
			data,err = cache.Get(p.cacheKey(addr))
			if nil != err {
				return err
			} else {
				c := &CoinMarketCap{}
				if err := json.Unmarshal(data, c); nil != err {
					log.Errorf("err:%s", err.Error())
				} else {
					tokenMarketCaps[addr] = c
				}
			}
		}
	}
	p.tokenMarketCaps = tokenMarketCaps
	body, err := cache.Get(p.cacheKey())
	if nil != err {
		err = p.syncMarketCapFromAPI()
	}

	if nil != err {
		return err
	} else {
		var caps []*CoinMarketCap
		if err := json.Unmarshal(body, &caps); nil != err {
			return err
		} else {
			syncedTokens := make(map[common.Address]bool)
			for _, tokenCap := range caps {
				if tokenAddress, exists := p.idToAddress[strings.ToUpper(tokenCap.Id)]; exists {
					p.tokenMarketCaps[tokenAddress].PriceUsd = tokenCap.PriceUsd
					p.tokenMarketCaps[tokenAddress].PriceBtc = tokenCap.PriceBtc
					p.tokenMarketCaps[tokenAddress].PriceCny = tokenCap.PriceCny
					p.tokenMarketCaps[tokenAddress].Volume24HCNY = tokenCap.Volume24HCNY
					p.tokenMarketCaps[tokenAddress].Volume24HUSD = tokenCap.Volume24HUSD
					p.tokenMarketCaps[tokenAddress].LastUpdated = tokenCap.LastUpdated
					log.Debugf("token:%s, priceUsd:%s", tokenAddress.Hex(), tokenCap.PriceUsd.FloatString(2))
					syncedTokens[p.tokenMarketCaps[tokenAddress].Address] = true
				}
			}
			for _, tokenCap := range p.tokenMarketCaps {
				if _, exists := syncedTokens[tokenCap.Address]; !exists && "VITE" != tokenCap.Symbol && "ARP" != tokenCap.Symbol {
					//todo:
					log.Errorf("token:%s, id:%s, can't sync marketcap at time:%d, it't last updated time:%d", tokenCap.Symbol, tokenCap.Id, time.Now().Unix(), tokenCap.LastUpdated)
				}
			}
		}
	}
	return nil
}

type MarketCapOptions struct {
	BaseUrl   string
	Currency  string
	Duration  int
	IsSync    bool
	DustValue *big.Rat
}

func NewMarketCapProvider(options *MarketCapOptions) *CapProvider_CoinMarketCap {
	provider := &CapProvider_CoinMarketCap{}
	provider.baseUrl = options.BaseUrl
	provider.currency = options.Currency
	provider.tokenMarketCaps = make(map[common.Address]*CoinMarketCap)
	provider.idToAddress = make(map[string]common.Address)
	provider.duration = options.Duration
	provider.dustValue = options.DustValue
	if provider.duration <= 0 {
		//default 5 min
		provider.duration = 5
	}
	provider.stopFuncs = []func(){}

	// default dust value is 1.0 usd/cny
	if provider.dustValue.Cmp(new(big.Rat).SetFloat64(0)) <= 0 {
		provider.dustValue = new(big.Rat).SetFloat64(1.0)
	}

	for _, v := range util.AllTokens {
		if "ARP" == v.Symbol || "VITE" == v.Symbol {
			c := &CoinMarketCap{}
			c.Address = v.Protocol
			c.Id = v.Source
			c.Name = v.Symbol
			c.Symbol = v.Symbol
			c.Decimals = new(big.Int).Set(v.Decimals)
			provider.tokenMarketCaps[c.Address] = c
		} else {
			c := &CoinMarketCap{}
			c.Address = v.Protocol
			c.Id = v.Source
			c.Name = v.Symbol
			c.Symbol = v.Symbol
			c.Decimals = new(big.Int).Set(v.Decimals)
			provider.tokenMarketCaps[c.Address] = c
			provider.idToAddress[strings.ToUpper(c.Id)] = c.Address
		}
	}

	if err := provider.syncMarketCapFromRedis(); nil != err {
		log.Fatalf("can't sync marketcap with error:%s", err.Error())
	}

	return provider
}

func (p *CapProvider_CoinMarketCap) IsOrderValueDust(state *types.OrderState) bool {
	remainedAmountS, _ := state.RemainedAmount()
	remainedValue, _ := p.LegalCurrencyValue(state.RawOrder.TokenS, remainedAmountS)

	return p.IsValueDusted(remainedValue)
}

func (p *CapProvider_CoinMarketCap) IsValueDusted(value *big.Rat) bool {
	return p.dustValue.Cmp(value) > 0
}


