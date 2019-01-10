package resources

const (
	orCondition   = "__or__"
	andCondition  = "__and__"
	boolCondition = "__bool__"
)

func applyFilterToResource(resource map[string]interface{}, filter map[string]interface{}) bool {
	for key, value := range filter {
		if !applyTopLevelFilter(resource, key, value) {
			return false
		}
	}
	return true
}

func applyTopLevelFilter(resource map[string]interface{}, key string, filter interface{}) bool {
	return applyNestedFilter(resource, transformTopLevelFilterToNestedFilter(key, filter))
}

func transformTopLevelFilterToNestedFilter(key string, filter interface{}) map[string]interface{} {
	switch key {
	case boolCondition, andCondition, orCondition:
		return map[string]interface{}{
			key: filter,
		}
	default:
		return map[string]interface{}{
			"property": key,
			"type":     "eq",
			"value":    filter,
		}
	}
}

func applyNestedFilter(resource map[string]interface{}, filter map[string]interface{}) bool {
	if f, ok := filter[boolCondition]; ok {
		return f.(bool)
	} else if f, ok := filter[andCondition]; ok {
		return applyAllFilters(resource, f.([]map[string]interface{}))
	} else if f, ok := filter[orCondition]; ok {
		return applyAnyFilters(resource, f.([]map[string]interface{}))
	} else {
		return applyPropertyFilter(resource, filter)
	}
}

func applyAllFilters(resource map[string]interface{}, filters []map[string]interface{}) bool {
	for _, filter := range filters {
		if !applyNestedFilter(resource, filter) {
			return false
		}
	}
	return true
}

func applyAnyFilters(resource map[string]interface{}, filters []map[string]interface{}) bool {
	for _, filter := range filters {
		if applyNestedFilter(resource, filter) {
			return true
		}
	}
	return false
}

func applyPropertyFilter(resource map[string]interface{}, filter map[string]interface{}) bool {
	actualValue, ok := resource[filter["property"].(string)]
	if !ok {
		return false
	}
	switch filter["type"] {
	case "eq":
		switch expectedValue := filter["value"].(type) {
		case []interface{}:
			for _, v := range expectedValue {
				if actualValue == v {
					return true
				}
			}
			return false
		case []string:
			for _, v := range expectedValue {
				if actualValue == v {
					return true
				}
			}
			return false
		default:
			return actualValue == expectedValue
		}
	case "neq":
		switch expectedValue := filter["value"].(type) {
		case []interface{}:
			for _, v := range expectedValue {
				if actualValue == v {
					return false
				}
			}
			return true
		case []string:
			for _, v := range expectedValue {
				if actualValue == v {
					return false
				}
			}
			return true
		default:
			return actualValue != expectedValue
		}
	default:
		return false
	}
}
