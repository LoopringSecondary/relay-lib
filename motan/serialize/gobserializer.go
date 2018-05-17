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

package serialize

import (
"bytes"
"encoding/gob"
"fmt"
)

const (
	Gob = "gob"
)

type GobSerialization struct {
}

func (s *GobSerialization) GetSerialNum() int {
	return 8
}

func (s *GobSerialization) Serialize(v interface{}) ([]byte, error) {
	buf := bytes.NewBuffer(make([]byte, 0, 100))
	if v == nil {
		buf.WriteByte(0)
		return buf.Bytes(), nil
	}
	var coderBuf bytes.Buffer
	enc := gob.NewEncoder(&coderBuf)
	err := enc.Encode(v)
	if err != nil {
		return nil, err
	} else {
		return coderBuf.Bytes(), nil
	}
}

func (s *GobSerialization) DeSerialize(b []byte, v interface{}) (interface{}, error) {
	if len(b) == 0 {
		return nil, nil
	}
	var coderBuf bytes.Buffer
	coderBuf.Write(b)
	dec := gob.NewDecoder(&coderBuf)
	err := dec.Decode(v)
	if err != nil {
		return nil, err
	} else {
		return v, nil
	}
}

func (s *GobSerialization) SerializeMulti(v []interface{}) ([]byte, error) {
	if len(v) != 1 {
		return nil, fmt.Errorf("Not support SerializeMulti")
	}
	return s.Serialize(v[0])
}

func (s *GobSerialization) DeSerializeMulti(b []byte, v []interface{}) (ret []interface{}, err error) {
	if len(v) != 1 {
		return nil, fmt.Errorf("Not support DeSerializeMulti")
	}
	if res, err := s.DeSerialize(b, v[0]); err == nil {
		return []interface{}{res}, nil
	} else {
		return nil, fmt.Errorf("Failed DeSerializeMulti for ", err)
	}
}
