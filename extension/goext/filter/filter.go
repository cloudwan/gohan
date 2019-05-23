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

func True() FilterElem {
	return FilterElem{
		"__bool__": true,
	}
}

func False() FilterElem {
	return FilterElem{
		"__bool__": false,
	}
}

func MaybeEmptyAndFilter(filters ...FilterElem) FilterElem {
	if len(filters) == 0 {
		return True()
	}
	if len(filters) == 1 {
		return filters[0]
	}
	return And(filters...)
}

func MaybeEmptyOrFilter(filters ...FilterElem) FilterElem {
	if len(filters) == 0 {
		return False()
	}
	if len(filters) == 1 {
		return filters[0]
	}
	return Or(filters...)
}
