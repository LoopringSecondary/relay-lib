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

package extractor_test

import (
	"encoding/json"
	"github.com/Loopring/relay-lib/eventemitter"
	"github.com/Loopring/relay-lib/types"
	"math/big"
	"testing"
)

func Test_Json(t *testing.T) {
	var a, b types.ForkedEvent
	a.DetectedBlock = big.NewInt(3)
	a.ForkBlock = big.NewInt(4)
	bs, err := json.Marshal(&a)
	if err != nil {
		t.Fatalf(err.Error())
	}

	var e interface{}
	e = &b
	if err := json.Unmarshal(bs, e); err != nil {
		t.Fatalf(err.Error())
	}

	switch te := e.(type) {
	case *types.ForkedEvent:
		t.Log(te.ForkBlock.String())
		t.Log(te.DetectedBlock.String())

	default:
		t.Log("type error")
	}
}

func Test_Disassemble(t *testing.T) {
	src := &types.KafkaOnChainEvent{}
	data := `{"block_hash":"0xc9924ec6e70a768455537401ee94365696d42644e531295229ceb5e6075486e4","block_number":42731,"block_time":1525860749,"tx_hash":"0x81554b2f972892ed6fe108aa2c890b50cb30d5454e4cd631a6c18198039716b1","tx_index":0,"tx_log_index":0,"value":0,"status":3,"gas_limit":500000,"gas_used":500000,"gas_price":14192604633,"nonce":4208,"identify":"submitRing","order_list":[{"protocol":"0x0000000000000000000000000000000000000000","delegateAddress":"0x0000000000000000000000000000000000000000","authAddr":"0x47fe1648b80fa04584241781488ce4c0aaca23e4","authPrivateKey":"","walletAddress":"0x1b978a1d302335a6f2ebe4b8823b5e17c3c84135","tokenS":"0xf079e0612e869197c5f4c7d0a95df570b163232b","tokenB":"0x1b978a1d302335a6f2ebe4b8823b5e17c3c84135","amountS":"0x16345785d8a0000","amountB":"0x1043561a8829300000","validSince":"0x5af2c976","validUntil":"0x5b769f76","lrcFee":"0x4563918244f40000","buyNoMoreThanAmountB":false,"marginSplitPercentage":0,"v":28,"r":"0x36b4203b64978f90684ab2ab69136303758965a02e4b57f21c0d14039dc6e989","s":"0x48ccff89527d764170aee0a5b68299b519a3eb29ebacea8e96d54dc40d643076","price":null,"owner":"0x1b978a1d302335a6f2ebe4b8823b5e17c3c84135","hash":"0x0000000000000000000000000000000000000000000000000000000000000000","market":"","createTime":0,"powNonce":0,"side":"","orderType":""},{"protocol":"0x0000000000000000000000000000000000000000","delegateAddress":"0x0000000000000000000000000000000000000000","authAddr":"0xb1018949b241d76a1ab2094f473e9befeabb5ead","authPrivateKey":"","walletAddress":"0x47fe1648b80fa04584241781488ce4c0aaca23e4","tokenS":"0x1b978a1d302335a6f2ebe4b8823b5e17c3c84135","tokenB":"0xf079e0612e869197c5f4c7d0a95df570b163232b","amountS":"0x1043561a8829300000","amountB":"0x5af2c976","validSince":"0x5b769f76","validUntil":"0x4563918244f40000","lrcFee":"0x16345785d8a0000","buyNoMoreThanAmountB":false,"marginSplitPercentage":0,"v":28,"r":"0xe9b8bfe36ad4a247e2dff75bfe8bc5e40adfe10047d8d5b5286c854ee838e749","s":"0x0603c26e356e0caf9dfe8e2af5bda4f71d2c9b70e177a6a04b5d19eb564d6477","price":null,"owner":"0xf079e0612e869197c5f4c7d0a95df570b163232b","hash":"0x0000000000000000000000000000000000000000000000000000000000000000","market":"","createTime":0,"powNonce":0,"side":"","orderType":""}],"fee_receipt":"0x4bad3053d574cd54513babe21db3f09bea1d387d","fee_selection":0,"err":"method submitRing transaction failed"}`
	src.Data = data
	src.Topic = eventemitter.Miner_SubmitRing_Method

	event := &types.SubmitRingMethodEvent{}
	if err := json.Unmarshal([]byte(src.Data), event); err != nil {
		t.Fatalf(err.Error())
	}

	for _, order := range event.OrderList {
		order.Hash = order.GenerateHash()
		t.Logf("order:%s, owner:%s, privatekey:%s", order.Hash.Hex(), order.Owner.Hex(), order.AuthPrivateKey.Address().Hex())
	}
}
