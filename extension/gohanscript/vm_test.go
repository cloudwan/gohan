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

package gohanscript_test

import (
	"testing"

	"github.com/cloudwan/gohan/extension/gohanscript"
	//Import gohan script lib
	_ "github.com/cloudwan/gohan/extension/gohanscript/autogen"

	. "github.com/onsi/ginkgo"
)

var _ = Describe("Run Gohan script test", func() {
	Context("When given gohan script test", func() {
		It("All test should be passed", func() {
			_, _, failed := gohanscript.RunTests("./tests")
			if failed != 0 {
				Fail("gohanscript test failed")
			}
		})
	})
})

func BenchmarkFib(b *testing.B) {
	vm := gohanscript.NewVM()
	vm.LoadFile("./examples/fib.yaml")
	for n := 0; n < b.N; n++ {
		vm.Run(map[string]interface{}{})
	}
}
