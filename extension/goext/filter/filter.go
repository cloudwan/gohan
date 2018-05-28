package filter

type FilterElem = map[string]interface{}

func Predicate(property, comp string, value interface{}) FilterElem {
	return FilterElem{
		"property": property,
		"type":     comp,
		"value":    value,
	}
}

func Eq(property string, value interface{}) FilterElem {
	return Predicate(property, "eq", value)
}

func Neq(property string, value interface{}) FilterElem {
	return Predicate(property, "neq", value)
}

func And(filters ...FilterElem) FilterElem {
	return FilterElem{
		"__and__": filters,
	}
}

func Or(filters ...FilterElem) FilterElem {
	return FilterElem{
		"__or__": filters,
	}
}
