package schema

import "github.com/pkg/errors"

type FilterFactory struct {
	visible, hidden []string
}

func (f *FilterFactory) CreateFilterFromProperties(visible, hidden []string) (*Filter, error) {
	f.visible = visible
	f.hidden = hidden

	strategy, err := f.createFilterStrategy()
	if err != nil {
		return nil, err
	}

	return &Filter{predicate: strategy}, nil
}

func CreateExcludeAllFilter() *Filter {
	return &Filter{predicate: &excludeAllPredicate{}}
}

func (f *FilterFactory) createFilterStrategy() (Predicate, error) {
	switch f.getFilterType() {
	case All:
		return &includeAllPredicate{}, nil
	case Visible:
		return &visiblePredicate{properties: sliceToSet(f.visible)}, nil
	case Hidden:
		return &hiddenPredicate{properties: sliceToSet(f.hidden)}, nil
	case Invalid:
		return nil, errors.New("Cannot have filter with both visible and hidden properties")
	default:
		return nil, errors.New("Unknown filter type")
	}
}

type filterType int

const (
	All filterType = iota
	Visible
	Hidden
	Invalid
)

func (f *FilterFactory) getFilterType() filterType {
	if f.visible == nil {
		if f.hidden == nil {
			return All
		}
		return Hidden
	}
	if f.hidden == nil {
		return Visible
	}
	return Invalid
}

func sliceToSet(keys []string) map[string]bool {
	result := make(map[string]bool)
	for _, key := range keys {
		result[key] = true
	}
	return result
}

type Filter struct {
	predicate Predicate
}

func (f *Filter) RemoveHiddenKeysFromMap(data map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	for key, value := range data {
		if f.predicate.Validate(key) {
			result[key] = value
		}
	}
	return result
}

func (f *Filter) RemoveHiddenKeysFromSlice(data []string) []string {
	result := make([]string, 0)
	for _, key := range data {
		if f.predicate.Validate(key) {
			result = append(result, key)
		}
	}
	return result
}

func (f *Filter) IsForbidden(key string) bool {
	return !f.predicate.Validate(key)
}

type Predicate interface {
	Validate(string) bool
}

type includeAllPredicate struct {
}

func (i *includeAllPredicate) Validate(string) bool {
	return true
}

type excludeAllPredicate struct {
}

func (e *excludeAllPredicate) Validate(string) bool {
	return false
}

type visiblePredicate struct {
	properties map[string]bool
}

func (v *visiblePredicate) Validate(s string) bool {
	return v.properties[s]
}

type hiddenPredicate struct {
	properties map[string]bool
}

func (h *hiddenPredicate) Validate(s string) bool {
	return !h.properties[s]
}
