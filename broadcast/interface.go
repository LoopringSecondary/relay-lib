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

package broadcast

var pubSubManager PubSubManager

type PubSubManager struct {
}

type Publisher struct {
}
type Subscriber struct {
}

type IPublisher interface {
	pub() (err error)
}

type ISubscriber interface {
	sub() (err error)
}

func NewPubSubManager(cfg PubSubConfig) PubSubManager {
	return PubSubManager{}
}

func (pb *PubSubManager) Start() {

}
