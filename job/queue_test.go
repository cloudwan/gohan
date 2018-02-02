// Copyright (C) 2016  Juniper Networks, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or
// implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package job

import (
	"runtime"
	"testing"

	"github.com/cloudwan/gohan/util"
)

func TestQueue(t *testing.T) {
	queue := NewQueue(5)
	for i := 0; i < 5; i++ {
		queue.Add(NewJob(
			func() {
				panic("Test panic")
			},
		))
	}
	counter := util.NewCounter(0)
	for i := 0; i < 100; i++ {
		queue.Add(NewJob(
			func() {
				counter.Add(1)
				runtime.Gosched()
			},
		))
	}

	queue.Stop()
	var expected int64 = 100
	if expected != counter.Value() {
		t.Fail()
	}
}
