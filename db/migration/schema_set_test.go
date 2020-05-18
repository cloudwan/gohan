// Copyright (C) 2020 NTT Innovation Institute, Inc.
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

package migration

import (
	"database/sql"
	"errors"
	"sort"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

type fakeSchemaSetBackend struct {
	initialIDs    []string
	persistedIDs  []string
	deletedIDs    []string
	failLoad      bool
	failOperation bool
}

func (b *fakeSchemaSetBackend) loadSchemaIDs() (stringSet, error) {
	if b.failLoad {
		return nil, errors.New("failure test")
	}

	result := make(stringSet, len(b.initialIDs))
	for _, id := range b.initialIDs {
		result[id] = true
	}

	return result, nil
}

func (b *fakeSchemaSetBackend) persistSchemaID(schemaID string, tx *sql.Tx) error {
	if b.failOperation {
		return errors.New("failure test")
	}

	b.persistedIDs = append(b.persistedIDs, schemaID)
	return nil
}

func (b *fakeSchemaSetBackend) deleteSchemaID(schemaID string) error {
	if b.failOperation {
		return errors.New("failure test")
	}

	b.deletedIDs = append(b.deletedIDs, schemaID)
	return nil
}

var _ schemaSetBackend = &fakeSchemaSetBackend{}

var _ = Describe("Schema Set", func() {
	var (
		set *schemaSet
	)

	BeforeEach(func() {
		set = newSchemaSet()
	})

	expectEmptySet := func() {
		Expect(set.getSchemaIDs()).To(BeEmpty())
	}

	It("Should have no schemas after construction", func() {
		expectEmptySet()
	})

	It("Should fail initialization with nil backend", func() {
		Expect(set.init(nil)).NotTo(Succeed())
		expectEmptySet()
	})

	It("Should fail initialization when backend fails", func() {
		Expect(set.init(&fakeSchemaSetBackend{failLoad: true})).NotTo(Succeed())
		expectEmptySet()
	})

	expectSchemaIDs := func(expectedIDs []string) {
		actualIDs := set.getSchemaIDs()

		sort.Strings(actualIDs)
		sort.Strings(expectedIDs)

		Expect(actualIDs).To(Equal(expectedIDs))
	}

	It("Should load schemaIDs from backend", func() {
		expectedIDs := []string{"a", "b", "c"}

		Expect(set.init(&fakeSchemaSetBackend{initialIDs: expectedIDs})).To(Succeed())

		expectSchemaIDs(expectedIDs)
	})

	It("Should persist marked schemaIDs", func() {
		backend := &fakeSchemaSetBackend{initialIDs: []string{"a", "b"}}
		Expect(set.init(backend)).To(Succeed())

		Expect(set.markSchemaID("x", nil)).To(Succeed())
		Expect(set.markSchemaID("y", nil)).To(Succeed())

		expectSchemaIDs([]string{"a", "b", "x", "y"})
		Expect(backend.persistedIDs).To(Equal([]string{"x", "y"}))
	})

	It("Should not mark schemaID if backend nil", func() {
		Expect(set.markSchemaID("x", nil)).NotTo(Succeed())
		expectEmptySet()
	})

	It("Should not mark schemaID if backend fails", func() {
		backend := &fakeSchemaSetBackend{}
		Expect(set.init(backend)).To(Succeed())
		Expect(set.markSchemaID("x", nil)).To(Succeed())

		backend.failOperation = true
		Expect(set.markSchemaID("y", nil)).NotTo(Succeed())

		expectSchemaIDs([]string{"x"})
	})

	Context("Given initial schema IDs", func() {
		const (
			aID = "schemaA"
			bID = "schemaB"
		)

		var (
			backend *fakeSchemaSetBackend
		)

		BeforeEach(func() {
			backend = &fakeSchemaSetBackend{initialIDs: []string{aID, bID}}
			Expect(set.init(backend)).To(Succeed())
		})

		It("Should persist removed schemaIDs", func() {
			Expect(set.removeSchemaID(aID)).To(Succeed())

			expectSchemaIDs([]string{bID})
			Expect(backend.deletedIDs).To(Equal([]string{aID}))
		})

		It("Should ignore removal of not stored schemaID", func() {
			Expect(set.removeSchemaID("no such id")).To(Succeed())

			expectSchemaIDs([]string{aID, bID})
		})

		It("Should not remove schemaID if backend fails", func() {
			backend.failOperation = true

			Expect(set.removeSchemaID(bID)).NotTo(Succeed())

			expectSchemaIDs([]string{aID, bID})
		})
	})

	It("Should not remove schemaID if backend nil", func() {
		Expect(set.removeSchemaID("id")).NotTo(Succeed())
	})
})
