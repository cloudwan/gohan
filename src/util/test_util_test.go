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

package util

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Custom matchers", func() {
	Describe("JSON matcher", func() {
		var (
			boot   map[string]interface{}
			sandal map[string]interface{}
		)

		BeforeEach(func() {
			boot = map[string]interface{}{
				"upper":   "long",
				"sole":    "thick",
				"comfort": 7,
			}
			sandal = map[string]interface{}{
				"upper":   "none",
				"sole":    "flexible",
				"comfort": 11,
			}
		})

		Context("When actual and expected match", func() {
			It("should match them", func() {
				Expect(boot).To(MatchAsJSON(boot))
			})
		})

		Context("When actual and expected don't match", func() {
			It("should match them", func() {
				Expect(boot).ToNot(MatchAsJSON(sandal))
			})
		})
	})
})
