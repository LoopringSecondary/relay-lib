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

package motan_test

import (
	"fmt"
	"bytes"
	"testing"
	"github.com/Loopring/relay-lib/motan"
	"time"
	"github.com/Loopring/relay-lib/motan/demo"
)

type MotanDemoService struct{}

func (m *MotanDemoService) Hello(book *demo.AddressBook) *demo.AddressBook {
	fmt.Printf("book.People.length :%d\n", len(book.People))
	fmt.Printf("book.People[0].Email :%s\n", book.People[0].Email)
	fmt.Printf("book.People[0].name :%s\n", book.People[0].Name)
	book.People[0].Name = book.People[0].Name + " >>>"
	book.People[0].Email = book.People[0].Email + " >>>"
	book.People[0].Phones[0].Number = book.People[0].Phones[0].Number + " >>>"
	return book
}

type Motan2TestService struct{}

func (m *Motan2TestService) Hello(params map[string]string) string {
	if params == nil {
		return "params is nil!"
	}
	var buffer bytes.Buffer
	for k, v := range params {
		if buffer.Len() > 0 {
			buffer.WriteString(",")
		}
		buffer.WriteString(k)
		buffer.WriteString("=")
		buffer.WriteString(v)
	}
	fmt.Printf("Motan2TestService hello:%s\n", buffer.String())
	return buffer.String()
}

func TestRunServer(t *testing.T) {
	serverInstance := &MotanDemoService{}
	options := motan.MotanServerOptions{}
	options.ConfFile = "./serverZkDemo.yaml"
	options.ServerInstance = serverInstance
	motan.RunServer(options)

	time.Sleep(5 * time.Minute)
}