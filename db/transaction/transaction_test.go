// Copyright (C) 2017 NTT Innovation Institute, Inc.
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

package transaction_test

import (
	tx "github.com/cloudwan/gohan/db/transaction"
	"github.com/cloudwan/gohan/schema"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Transaction", func() {
	var netSchema *schema.Schema

	Describe("GetTransactionIsolationLevel", func() {
		BeforeEach(func() {
			var exists bool
			manager := schema.GetManager()
			basePath := "../../tests/test_abstract_schema.yaml"
			Expect(manager.LoadSchemaFromFile(basePath)).To(Succeed())

			schemaPath := "../../tests/test_schema.yaml"
			Expect(manager.LoadSchemaFromFile(schemaPath)).To(Succeed())
			netSchema, exists = manager.Schema("network")
			Expect(exists).To(BeTrue())
		})

		It("Defaults to serializable", func() {
			Expect(tx.GetIsolationLevel(netSchema, "create")).To(Equal(tx.Serializable))
		})

		It("Inherits base schema isolation level", func() {
			Expect(tx.GetIsolationLevel(netSchema, "delete")).To(Equal(tx.ReadCommited))
		})

		It("Gets schema overrides", func() {
			Expect(tx.GetIsolationLevel(netSchema, "update")).To(Equal(tx.Serializable))
		})
	})
})
