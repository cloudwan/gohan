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
	. "github.com/onsi/ginkgo/extensions/table"
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

		DescribeTable("should pass",
			func(input string) {
				Expect(formatChecker.IsFormat(input)).To(Equal(true))
			},
			Entry("IPv4 with 1 in the last octet", "127.0.0.1/16"),
			Entry("IPv4 with 100 in the last octet", "127.0.0.100/16"),
			Entry("IPv4 with 20 in the last octet", "127.0.0.20/16"),
			Entry("IPv6", "::1/16"),
		)

		DescribeTable("should fail",
			func(input string) {
				Expect(formatChecker.IsFormat(input)).To(Equal(false))
			},
			Entry("IPv4 only", "127.0.0.1"),
			Entry("IPv4 only with one redundant zero in the last octet", "127.0.0.01"),
			Entry("IPv4 only with two redundant zeros in the last octet", "127.0.0.001"),
			Entry("IPv6 only", "::1"),
			Entry("wrong IPv4", "218.108.149.379/16"),
			Entry("wrong IPv6", "134:g/16"),
			Entry("IPv4 with wrong mask", "127.0.0.1/33"),
			Entry("IPv6 with wrong mask", "::1/129"),
			Entry("IPv4 with two zeros in octet", "127.00.0.1/24"),
			Entry("IPv4 with three zeros in octet", "127.000.0.1/24"),
			Entry("IPv4 with two zeros in the first octet", "00.0.0.1/24"),
			Entry("IPv4 with three zeros in the first octet", "000.0.0.1/24"),
			Entry("IPv4 with one redundant zero in the last octet ", "1.4.0.01/24"),
			Entry("IPv4 with two redundant zeros in the last octet ", "1.4.0.001/24"),
			Entry("IPv4 network with leading zero in mask ", "128.0.0.0/04"),
			Entry("IPv4 network with leading zeros in mask ", "128.0.0.0/004"),
			Entry("IPv4 network with incorrect mask ", "128.0.0.0/4/6"),
			Entry("IPv4 network with empty mask ", "128.0.0.0/"),
			Entry("IPv4 network without mask ", "128.0.0.0"),
		)
	})

	Describe("IPv4 format checker", func() {

		BeforeEach(func() {
			formatChecker = ipv4FormatChecker{}
		})

		DescribeTable("should pass",
			func(input string) {
				Expect(formatChecker.IsFormat(input)).To(Equal(true))
			},
			Entry("IPv4 with 1 in last octet", "127.0.0.1"),
			Entry("IPv4 with 10 in last octet", "127.0.0.10"),
			Entry("IPv4 with 100 in last octet", "127.0.0.100"),

		)

		DescribeTable("should fail",
			func(input string) {
				Expect(formatChecker.IsFormat(input)).To(Equal(false))
			},
			Entry("IPv6", "::1"),
			Entry("IPv4 cidr", "127.0.0.1/16"),
			Entry("IPv4 network", "127.0.0.0/16"),
			Entry("IPv6 cidr", "::1/16"),
			Entry("IPv6 network", "fe80::9e17/64"),
			Entry("wrong IPv4", "218.108.149.379"),
			Entry("wrong IPv4 with three octets only", "218.149.179"),
			Entry("IPv4 with two zeros in octet ", "127.00.0.1"),
			Entry("IPv4 with three zeros in octet ", "127.000.0.1"),
			Entry("IPv4 with two zeros in the first octet ", "00.3.0.1"),
			Entry("IPv4 with three zeros in the first octet ", "000.4.0.1"),
			Entry("IPv4 with one redundant zero in the last octet ", "1.4.0.01"),
			Entry("IPv4 with two redundant zeros in the last octet ", "1.4.0.001"),
		)
	})

	Describe("CIDR or IPv4 checker", func() {

		BeforeEach(func() {
			formatChecker = cidrOrIPv4FormatChecker{}
		})

		DescribeTable("should pass",
			func(input string) {
				Expect(formatChecker.IsFormat(input)).To(Equal(true))
			},
			Entry("IPv4 with 1 in the last octet", "127.0.0.1"),
			Entry("IPv4 with 100 in the last octet", "127.0.0.100"),
			Entry("IPv4 cidr with 1 in the last octet", "127.0.0.1/16"),
			Entry("IPv4 cidr with 100 in the last octet", "127.0.0.100/16"),
			Entry("IPv4 network", "127.0.0.0/24"),
		)

		DescribeTable("should fail",
			func(input string) {
				Expect(formatChecker.IsFormat(input)).To(Equal(false))
			},
			Entry("IPv6", "::1"),
			Entry("IPv6 cidr", "::1/16"),
			Entry("IPv6 network", "fe80::9e17/64"),
			Entry("wrong IPv4", "218.108.149.379/16"),
			Entry("wrong IPv6", "134:g/16"),
			Entry("IPv4 with wrong mask", "127.0.0.1/33"),
			Entry("IPv6 with wrong mask", "::1/129"),
			Entry("IPv4 cidr with two zeros in octet", "127.00.0.1/24"),
			Entry("IPv4 cidr with three zeros in octet", "127.000.0.1/24"),
			Entry("IPv4 with two zeros in octet", "127.00.0.1"),
			Entry("IPv4 with three zeros in octet", "127.000.0.1"),
			Entry("IPv4 cidr with two zeros in the first octet", "00.0.0.1/24"),
			Entry("IPv4 cidr with three zeros in the first octet", "000.0.0.1/24"),
			Entry("IPv4 with two zeros in the first octet", "00.0.0.1"),
			Entry("IPv4 with three zeros in first octet", "000.0.0.1"),
			Entry("IPv4 with one redundant zero in the last octet ", "1.4.0.01"),
			Entry("IPv4 with two redundant zeros in the last octet ", "1.4.0.001"),
			Entry("IPv4 with one redundant zero in the last octet ", "1.4.0.01"),
			Entry("IPv4 with two redundant zeros in the last octet ", "1.4.0.001"),
			Entry("IPv4 cidr and one redundant zero in the last octet ", "1.4.0.01/24"),
			Entry("IPv4 cidr and two redundant zeros in the last octet ", "1.4.0.001/24"),
			Entry("IPv4 network with one redundant zero in the last octet ", "1.4.0.00/24"),
			Entry("IPv4 network with two redundant zeros in the last octet ", "1.4.0.000/24"),
			Entry("IPv4 network with leading zero in mask ", "128.0.0.0/04"),
			Entry("IPv4 network with leading zeros in mask ", "128.0.0.0/004"),
			Entry("IPv4 network with incorrect mask ", "128.0.0.0/4/6"),
			Entry("IPv4 network with empty mask ", "128.0.0.0/"),
		)
	})

	Describe("IPv4 Network checker", func() {

		BeforeEach(func() {
			formatChecker = ipv4NetworkFormatChecker{}
		})

		DescribeTable("should pass",
			func(input string) {
				Expect(formatChecker.IsFormat(input)).To(Equal(true))
			},
			Entry("no '1' host bits", "192.168.0.0/24"),
		)

		DescribeTable("should fail",
			func(input string) {
				Expect(formatChecker.IsFormat(input)).To(Equal(false))
			},
			Entry("IPv4 network with '1' host bits present", "192.168.0.2/24"),
			Entry("IPv4", "10.11.12.13"),
			Entry("wrong IPv4 network", "218.308.0.0/16"),
			Entry("IPv4 network with wrong mask", "127.0.0.0/33"),
			Entry("IPv4 network with two zeros in octet", "192.168.00.0/24"),
			Entry("IPv4 network with three zeros in octet", "192.168.000.0/24"),
			Entry("IPv4 network with two zeros in the first octet ", "00.168.0.0/24"),
			Entry("IPv4 network with three zeros in the first octet ", "000.168.0.0/24"),
			Entry("IPv4 network with one redundant zero in the last octet ", "1.168.0.00/24"),
			Entry("IPv4 network with two redundant zeros in the last octet ", "1.168.0.000/24"),
			Entry("IPv4 network with leading zero in mask ", "128.0.0.0/04"),
			Entry("IPv4 network with leading zeros in mask ", "128.0.0.0/004"),
			Entry("IPv4 network with incorrect mask ", "128.0.0.0/4/6"),
			Entry("IPv4 network with empty mask ", "128.0.0.0/"),
			Entry("IPv4 network without mask ", "128.0.0.0"),
		)
	})

	Describe("IPv4 Address with CIDR checker", func() {

		BeforeEach(func() {
			formatChecker = ipv4AddressWithCidrFormatChecker{}
		})

		DescribeTable("should pass",
			func(input string) {
				Expect(formatChecker.IsFormat(input)).To(Equal(true))
			},
			Entry("cidr with '1' host bits present", "192.168.0.2/24"),
			Entry("cidr which is not a broadcast", "192.168.255.2/16"),
		)

		DescribeTable("should fail",
			func(input string) {
				Expect(formatChecker.IsFormat(input)).To(Equal(false))
			},
			Entry("IPv4 network", "192.168.0.0/24"),
			Entry("broadcast IPv4 address with cidr", "192.168.255.255/16"),
			Entry("broadcast IPv4 address with mask /25", "192.168.2.127/25"),
			Entry("wrong IPv4", "218.108.149.379/16"),
			Entry("IPv4 with wrong mask", "127.0.0.1/33"),
			Entry("IPv4 with two zeros in octet", "192.168.00.2/24"),
			Entry("IPv4 with three zeros in octet", "192.168.000.2/24"),
			Entry("IPv4 with two zeros in the first octet", "00.168.0.2/24"),
			Entry("IPv4 with three zeros in the first octet ", "000.168.0.2/24"),
			Entry("IPv4 with one redundant zero in the last octet", "00.168.0.02/24"),
			Entry("IPv4 with two redundant zeros in the last octet ", "000.168.0.002/24"),
			Entry("IPv4 network with leading zero in mask ", "128.0.0.0/04"),
			Entry("IPv4 network with leading zeros in mask ", "128.0.0.0/004"),
			Entry("IPv4 network with incorrect mask ", "128.0.0.0/4/6"),
			Entry("IPv4 network with empty mask ", "128.0.0.0/"),
			Entry("IPv4 network without mask ", "128.0.0.0"),
		)
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

	Describe("Version format checker", func() {
		BeforeEach(func() {
			formatChecker = versionFormatChecker{}
		})

		DescribeTable("should pass", func(version string) {
			Expect(formatChecker.IsFormat(version)).To(BeTrue())
		},
			Entry("", "1.1.1"),
			Entry("", "1.1.1-abc"),
			Entry("", "1"),
			Entry("", "1.1"),
			Entry("", "v1.1.1"),
		)

		DescribeTable("should fail", func(version string) {
			Expect(formatChecker.IsFormat(version)).To(BeFalse())
		},
			Entry("", "1.1,1"),
			Entry("", "1.1."),
			Entry("", "1.1.1abc"),
		)
	})

	Describe("Version constraint format checker", func() {
		BeforeEach(func() {
			formatChecker = versionConstraintFormatChecker{}
		})

		DescribeTable("valid constraints", func(version string) {
			Expect(formatChecker.IsFormat(version)).To(BeTrue())
		},
			Entry("equal", "=1.1.1"),
			Entry("greater equal", ">=1.1.1-abc"),
			Entry("less", "<1.1"),
			Entry("no comparison", "1.1.1"),
		)

		DescribeTable("invalid constraints", func(version string) {
			Expect(formatChecker.IsFormat(version)).To(BeFalse())
		},
			Entry("not a version", ">abc"),
		)
	})
})
