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

package name

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("common mark tests", func() {
	Describe("length difference tests", func() {
		It("Should get a positive length difference", func() {
			commonMark := &CommonMark{
				used:  true,
				begin: 1,
				end:   2,
			}

			result := commonMark.lengthDifference()

			expected := 5
			Expect(result).To(Equal(expected))
		})

		It("Should get a negative length difference", func() {
			commonMark := &CommonMark{
				used:  true,
				begin: 0,
				end:   10,
			}

			result := commonMark.lengthDifference()

			expected := -4
			Expect(result).To(Equal(expected))
		})

		It("Should get no difference for unused mark", func() {
			commonMark := &CommonMark{
				used:  false,
				begin: 0,
				end:   10,
			}

			result := commonMark.lengthDifference()

			expected := 0
			Expect(result).To(Equal(expected))
		})
	})

	Describe("update tests", func() {
		It("Should update begin and end of a make", func() {
			commonMark := &CommonMark{
				used:  true,
				begin: 2,
				end:   3,
			}
			otherMark := &CommonMark{
				used:  true,
				begin: 0,
				end:   8,
			}

			commonMark.Update(otherMark)

			Expect(commonMark.begin).To(Equal(0))
			Expect(commonMark.end).To(Equal(1))
		})
	})

	Describe("change tests", func() {
		It("Should not change string that starts with prefix followed by common", func() {
			prefix := "test"
			string := prefix + "common"
			old := string
			mark := &CommonMark{
				used:  false,
				begin: len(prefix),
				end:   len(prefix),
			}

			result := mark.Change(&string)

			Expect(result).To(BeFalse())
			Expect(string).To(Equal(old))
			Expect(mark.used).To(BeFalse())
		})

		It("Should change string", func() {
			prefix := "test"
			string := prefix + "suffix"
			mark := &CommonMark{
				used:  false,
				begin: len(prefix),
				end:   len(prefix),
			}

			result := mark.Change(&string)

			Expect(result).To(BeTrue())
			Expect(string).To(Equal(prefix + "common"))
			Expect(mark.used).To(BeTrue())
		})

		It("Should change string with suffix", func() {
			prefix := "test"
			suffix := "suffix"
			string := prefix + suffix
			mark := &CommonMark{
				used:  true,
				begin: len(prefix),
				end:   len(prefix),
			}

			result := mark.Change(&string)

			Expect(result).To(BeTrue())
			Expect(string).To(Equal(prefix + "common" + suffix))
			Expect(mark.used).To(BeTrue())
		})

		It("Should change twice", func() {
			prefix := "prefix"
			suffix := "long_suffix"
			first := prefix + suffix
			second := first + suffix
			mark := CreateMark(prefix)

			resultFirst := mark.Change(&first)
			resultSecond := mark.Change(&second)

			expected := prefix + "common"
			Expect(resultFirst).To(BeTrue())
			Expect(resultSecond).To(BeTrue())
			Expect(first).To(Equal(expected))
			Expect(second).To(Equal(expected + suffix))
		})
	})
})
