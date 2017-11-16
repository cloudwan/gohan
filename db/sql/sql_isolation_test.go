// Copyright (C) 2015 NTT Innovation Institute, Inc.
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

package sql_test

import (
	"context"
	"fmt"
	"strings"

	"github.com/cloudwan/gohan/db"
	. "github.com/cloudwan/gohan/db/sql"
	"github.com/cloudwan/gohan/db/transaction"
	"github.com/cloudwan/gohan/schema"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Mysql", func() {

	var (
		testDB     db.DB
		testSchema *schema.Schema
	)

	BeforeEach(func() {
		manager := schema.GetManager()
		var err error
		testDB, err = db.ConnectLocalMySQL()
		Expect(err).To(Succeed())
		Expect(manager.LoadSchemasFromFiles(
			"../../etc/schema/gohan.json",
			"../../tests/test_abstract_schema.yaml",
			"../../tests/test_schema.yaml")).To(Succeed())
		err = db.InitSchemaConn(testDB, db.DefaultTestSchemaParams())
		var ok bool
		testSchema, ok = manager.Schema("test")
		Expect(ok).To(BeTrue())
	})

	AfterEach(func() {
		testDB.Close()
		schema.ClearManager()
		testDB.Purge()
	})

	Describe("Isolation levels", func() {

		It("Isolation level is set on transaction", func() {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			tx1, err := testDB.BeginTx(ctx, &transaction.TxOptions{IsolationLevel: transaction.RepeatableRead})
			Expect(err).To(Succeed())

			tx2, err := testDB.BeginTx(ctx, &transaction.TxOptions{IsolationLevel: transaction.ReadCommited})
			Expect(err).To(Succeed())

			Expect(tx1.Exec("INSERT INTO `tests` (`id`, `tenant_id`) values ('id', 'tenant')")).To(Succeed())

			selectQuery := fmt.Sprintf(
				"SELECT %s FROM %s",
				strings.Join(MakeColumns(testSchema, testSchema.GetDbTableName(), nil, false), ", "),
				testSchema.GetDbTableName(),
			)
			results, err := tx2.Query(testSchema, selectQuery, []interface{}{})
			Expect(err).ToNot(HaveOccurred())
			Expect(len(results)).To(Equal(0))

			Expect(tx1.Commit()).To(Succeed())

			results, err = tx2.Query(testSchema, selectQuery, []interface{}{})
			Expect(err).ToNot(HaveOccurred())
			Expect(len(results)).To(Equal(1))

			Expect(tx2.Commit()).To(Succeed())
		})
	})
})
