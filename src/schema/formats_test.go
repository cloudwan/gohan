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

package schema

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/xeipuuv/gojsonschema"
)

var _ = Describe("format checkers", func() {
	var formatChecker gojsonschema.FormatChecker

	Describe("MAC format checker", func() {
		BeforeEach(func() {
			formatChecker = macFormatChecker{}
		})

		It("Should pass - ':' deliminer", func() {
			result := formatChecker.IsFormat("aa:bb:cc:dd:ee:ff")
			Expect(result).To(Equal(true))
		})

		It("Should not pass - '-' deliminer", func() {
			result := formatChecker.IsFormat("aa-bb-cc-dd-ee-ff")
			Expect(result).To(Equal(false))
		})

		It("Should not pass - illegal characters", func() {
			result := formatChecker.IsFormat("gg:bb:cc:dd:ee:ff")
			Expect(result).To(Equal(false))
		})
	})

	Describe("CIDR format checker", func() {
		BeforeEach(func() {
			formatChecker = cidrFormatChecker{}
		})

		It("Should pass - IPv4", func() {
			result := formatChecker.IsFormat("127.0.0.1/16")
			Expect(result).To(Equal(true))
		})

		It("Should pass - IPv6", func() {
			result := formatChecker.IsFormat("::1/16")
			Expect(result).To(Equal(true))
		})

		It("Should not pass - IPv4 only", func() {
			result := formatChecker.IsFormat("127.0.0.1")
			Expect(result).To(Equal(false))
		})

		It("Should not pass - IPv6 only", func() {
			result := formatChecker.IsFormat("::1")
			Expect(result).To(Equal(false))
		})

		It("Should not pass - wrong IPv4", func() {
			result := formatChecker.IsFormat("218.108.149.379/16")
			Expect(result).To(Equal(false))
		})

		It("Should not pass - wrong IPv6", func() {
			result := formatChecker.IsFormat("134:g/16")
			Expect(result).To(Equal(false))
		})

		It("Should not pass - wrong mask with IPv4", func() {
			result := formatChecker.IsFormat("127.0.0.1/33")
			Expect(result).To(Equal(false))
		})

		It("Should not pass - wrong mask with IPv6", func() {
			result := formatChecker.IsFormat("::1/129")
			Expect(result).To(Equal(false))
		})
	})

	Describe("CIDR or IPv4 checker", func() {
		BeforeEach(func() {
			formatChecker = cidrOrIPv4FormatChecker{}
		})

		It("Should pass - IPv4", func() {
			result := formatChecker.IsFormat("127.0.0.1")
			Expect(result).To(Equal(true))
		})

		It("Should not pass - IPv6", func() {
			result := formatChecker.IsFormat("::1")
			Expect(result).To(Equal(false))
		})

		It("Should pass - IPv4 cidr", func() {
			result := formatChecker.IsFormat("127.0.0.1/16")
			Expect(result).To(Equal(true))
		})

		It("Should not pass - IPv6 cidr", func() {
			result := formatChecker.IsFormat("::1/16")
			Expect(result).To(Equal(false))
		})

		It("Should not pass - wrong IPv4", func() {
			result := formatChecker.IsFormat("218.108.149.379/16")
			Expect(result).To(Equal(false))
		})

		It("Should not pass - wrong IPv6", func() {
			result := formatChecker.IsFormat("134:g/16")
			Expect(result).To(Equal(false))
		})

		It("Should not pass - wrong mask with IPv4", func() {
			result := formatChecker.IsFormat("127.0.0.1/33")
			Expect(result).To(Equal(false))
		})

		It("Should not pass - wrong mask with IPv6", func() {
			result := formatChecker.IsFormat("::1/129")
			Expect(result).To(Equal(false))
		})
	})

	Describe("Regex format checker", func() {
		BeforeEach(func() {
			formatChecker = regexFormatChecker{}
		})

		It("Should pass", func() {
			result := formatChecker.IsFormat("^[a-z]{1,3}-\\1")
			Expect(result).To(Equal(false))
		})

		It("Should not pass - incorrect regex", func() {
			result := formatChecker.IsFormat(".{3,1}")
			Expect(result).To(Equal(false))
		})

		It("Should pass - illogical regex", func() {
			result := formatChecker.IsFormat("$^")
			Expect(result).To(Equal(true))
		})
	})

	Describe("UUID format checker", func() {
		BeforeEach(func() {
			formatChecker = uuidFormatChecker{}
		})

		Context("UUID with hyphens", func() {
			It("Should pass", func() {
				result := formatChecker.IsFormat("12345678-1234-1234-1234-123456789012")
				Expect(result).To(Equal(true))
			})

			It("Should pass - all characters", func() {
				result := formatChecker.IsFormat("12345678-90ab-3cde-fABC-DEF456789012")
				Expect(result).To(Equal(true))
			})

			It("Should not pass - wrong syntax", func() {
				result := formatChecker.IsFormat("12345678-1234-1234-12345-12345678901")
				Expect(result).To(Equal(false))
			})

			It("Should not pass - wrong version", func() {
				result := formatChecker.IsFormat("12345678-1234-7234-12345-12345678901")
				Expect(result).To(Equal(false))
			})
		})

		Context("UUID without hyphens", func() {
			It("Should pass", func() {
				result := formatChecker.IsFormat("12345678123412341234123456789012")
				Expect(result).To(Equal(true))
			})

			It("Should pass - all characters", func() {
				result := formatChecker.IsFormat("1234567890ab3cdefABCDEF456789012")
				Expect(result).To(Equal(true))
			})

			It("Should not pass - wrong length", func() {
				result := formatChecker.IsFormat("123456781234123412345123456789011")
				Expect(result).To(Equal(false))
			})

			It("Should not pass - wrong version", func() {
				result := formatChecker.IsFormat("12345678123472341234512345678901")
				Expect(result).To(Equal(false))
			})
		})
	})

	Describe("UUID with hyphens format checker", func() {
		BeforeEach(func() {
			formatChecker = hyphenatedUUIDFormatChecker{}
		})

		It("Should pass", func() {
			result := formatChecker.IsFormat("12345678-1234-1234-1234-123456789012")
			Expect(result).To(Equal(true))
		})

		It("Should not pass - lack of hyphens", func() {
			result := formatChecker.IsFormat("12345678123412341234123456789012")
			Expect(result).To(Equal(false))
		})

		It("Should pass - all characters", func() {
			result := formatChecker.IsFormat("12345678-90ab-3cde-fABC-DEF456789012")
			Expect(result).To(Equal(true))
		})

		It("Should not pass - wrong syntax", func() {
			result := formatChecker.IsFormat("12345678-1234-1234-12345-12345678901")
			Expect(result).To(Equal(false))
		})

		It("Should not pass - wrong version", func() {
			result := formatChecker.IsFormat("12345678-1234-7234-12345-12345678901")
			Expect(result).To(Equal(false))
		})
	})

	Describe("UUID without hyphens format checker", func() {
		BeforeEach(func() {
			formatChecker = nonHyphenatedUUIDFormatChecker{}
		})

		It("Should pass", func() {
			result := formatChecker.IsFormat("12345678123412341234123456789012")
			Expect(result).To(Equal(true))
		})

		It("Should not pass - too much hyphens", func() {
			result := formatChecker.IsFormat("12345678-1234-1234-1234-123456789012")
			Expect(result).To(Equal(false))
		})

		It("Should pass - all characters", func() {
			result := formatChecker.IsFormat("1234567890ab3cdefABCDEF456789012")
			Expect(result).To(Equal(true))
		})

		It("Should not pass - wrong length", func() {
			result := formatChecker.IsFormat("123456781234123412345123456789011")
			Expect(result).To(Equal(false))
		})

		It("Should not pass - wrong version", func() {
			result := formatChecker.IsFormat("12345678123472341234512345678901")
			Expect(result).To(Equal(false))
		})
	})

	Describe("Port format checker", func() {
		BeforeEach(func() {
			formatChecker = portFormatChecker{}
		})

		It("Should pass", func() {
			result := formatChecker.IsFormat("1024")
			Expect(result).To(Equal(true))
		})

		It("Should not pass - zero", func() {
			result := formatChecker.IsFormat("0")
			Expect(result).To(Equal(false))
		})

		It("Should not pass - negative", func() {
			result := formatChecker.IsFormat("-1024")
			Expect(result).To(Equal(false))
		})

		It("Should not pass - too big", func() {
			result := formatChecker.IsFormat("65536")
			Expect(result).To(Equal(false))
		})
	})
})
