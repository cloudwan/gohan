package resources

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
)

type list = []interface{}
type object = map[string]interface{}

var _ = Describe("Policy filter", func() {
	DescribeTable("Apply filter to resource",
		func(resource object, filter object, result bool) {
			Expect(applyFilterToResource(resource, filter)).To(Equal(result))
		},
		Entry("Top level single property should match",
			object{
				"a": "b",
			},
			property("a", "b"),
			true,
		),
		Entry("Top level single property should not match",
			object{
				"a": "b",
			},
			property("a", 1),
			false,
		),
		Entry("Top level single property list should match",
			object{
				"a": "b",
			},
			property("a", list{"a", "b", "c"}),
			true,
		),
		Entry("Top level single property list should not match",
			object{
				"a": "b",
			},
			property("a", list{"a", 1, false}),
			false,
		),
		Entry("Conjunction of properties should match",
			object{
				"a": 1,
				"b": false,
			},
			and(
				eq("a", 1),
				neq("b", true),
			),
			true,
		),
		Entry("Conjunction of properties should not match",
			object{
				"a": 1,
				"b": false,
			},
			and(
				neq("a", "a"),
				eq("b", true),
			),
			false,
		),
		Entry("Disjunction of properties should match",
			object{
				"a": 1,
				"b": false,
			},
			or(
				eq("a", list{"a", "b"}),
				neq("b", list{true}),
			),
			true,
		),
		Entry("Disjunction of properties should not match",
			object{
				"a": 1,
				"b": false,
			},
			or(
				neq("a", 1),
				neq("b", list{true, false}),
			),
			false,
		),
		Entry("Complex combination of properties should match",
			object{
				"a": 1,
				"b": false,
				"c": "x",
			},
			all(
				property("a", list{1, 2}),
				property("b", false),
				or(
					and(
						eq("a", 1),
						boolean(false),
					),
					and(
						eq("c", list{"z", "y", "x"}),
						boolean(true),
					),
				),
			),
			true,
		),
		Entry("Complex combination of properties should not match",
			object{
				"a": 1,
				"b": false,
				"c": "x",
			},
			all(
				property("a", 1),
				and(
					and(
						eq("a", 1),
						boolean(false),
					),
					and(
						eq("c", list{"z", "y", "x"}),
						boolean(true),
					),
				),
			),
			false,
		),
		Entry("List of strings should match",
			object{
				"a": "x",
				"b": "y",
				"c": "z",
			},
			all(
				property("a", "x"),
				property("b", []string{"y"}),
			),
			true,
		),
		Entry("List of strings should not match",
			object{
				"a": "x",
				"b": "y",
				"c": "z",
			},
			all(
				property("a", "x"),
				property("b", []string{"1", "2"}),
			),
			false,
		),
		Entry("Nonexistent property should not match",
			object{
				"a": 1,
			},
			property("b", 1),
			false,
		),
	)
})

func all(filters ...object) object {
	result := object{}
	for _, filter := range filters {
		for key, value := range filter {
			result[key] = value
		}
	}
	return result
}

func property(name string, value interface{}) object {
	return object{
		name: value,
	}
}

func boolean(value bool) object {
	return object{
		"__bool__": value,
	}
}

func and(filters ...object) object {
	return object{
		"__and__": filters,
	}
}

func or(filters ...object) object {
	return object{
		"__or__": filters,
	}
}

func eq(property string, value interface{}) object {
	return object{
		"property": property,
		"type":     "eq",
		"value":    value,
	}
}

func neq(property string, value interface{}) object {
	return object{
		"property": property,
		"type":     "neq",
		"value":    value,
	}
}
