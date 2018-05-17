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
