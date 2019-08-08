package sql

import (
	"strconv"

	sq "github.com/Masterminds/squirrel"
	"github.com/cloudwan/gohan/db/search"
	"github.com/cloudwan/gohan/schema"
)

const (
	OrCondition   = "__or__"
	AndCondition  = "__and__"
	BoolCondition = "__bool__"
)

type queryBuilder interface {
	Where(pred interface{}, args ...interface{})
}

type selectQueryBuilder struct {
	sq.SelectBuilder
}

func (b *selectQueryBuilder) Where(pred interface{}, args ...interface{}) {
	b.SelectBuilder = b.SelectBuilder.Where(pred, args...)
}

type deleteQueryBuilder struct {
	sq.DeleteBuilder
}

func (b *deleteQueryBuilder) Where(pred interface{}, args ...interface{}) {
	b.DeleteBuilder = b.DeleteBuilder.Where(pred, args...)
}

func AddFilterToSelectQuery(s *schema.Schema, q sq.SelectBuilder, filter map[string]interface{}, join bool) (sq.SelectBuilder, error) {
	qb := &selectQueryBuilder{q}
	err := addFilterToQuery(s, qb, filter, join)
	return qb.SelectBuilder, err
}

func AddFilterToDeleteQuery(s *schema.Schema, q sq.DeleteBuilder, filter map[string]interface{}, join bool) (sq.DeleteBuilder, error) {
	qb := &deleteQueryBuilder{q}
	err := addFilterToQuery(s, qb, filter, join)
	return qb.DeleteBuilder, err
}

func addFilterToQuery(s *schema.Schema, q queryBuilder, filter map[string]interface{}, join bool) error {
	if filter == nil {
		return nil
	}
	for key, value := range filter {
		if key == OrCondition {
			orFilter, err := addOrToQuery(s, q, value, join)
			if err != nil {
				return err
			}
			q.Where(orFilter)
			continue
		} else if key == AndCondition {
			andFilter, err := addAndToQuery(s, q, value, join)
			if err != nil {
				return err
			}
			q.Where(andFilter)
			continue
		} else if b, ok := filter[BoolCondition]; ok {
			if b.(bool) {
				q.Where("(1=1)")
			} else {
				q.Where("(1=0)")
			}
			continue
		}

		property, err := s.GetPropertyByID(key)

		if err != nil {
			return err
		}

		var column string
		if join {
			column = makeColumn(s.GetDbTableName(), *property)
		} else {
			column = quote(key)
		}

		substr, ok := value.(search.Search)
		if ok {
			q.Where(Like{column: substr.Value})
			continue
		}

		q.Where(sq.Eq{column: parseBoolean(property.Type, value)})
	}
	return nil
}

func parseBoolean(typ string, value interface{}) interface{} {
	queryValues, ok := value.([]string)
	if ok && typ == "boolean" {
		v := make([]bool, len(queryValues))
		for i, j := range queryValues {
			v[i], _ = strconv.ParseBool(j)
		}
		return v
	}
	return value
}

func addOrToQuery(s *schema.Schema, q queryBuilder, filter interface{}, join bool) (sq.Or, error) {
	return addToFilter(s, q, filter, join, sq.Or{})
}

func addAndToQuery(s *schema.Schema, q queryBuilder, filter interface{}, join bool) (sq.And, error) {
	return addToFilter(s, q, filter, join, sq.And{})
}

func addToFilter(s *schema.Schema, q queryBuilder, filter interface{}, join bool, sqlizer []sq.Sqlizer) ([]sq.Sqlizer, error) {
	filters := filter.([]map[string]interface{})
	for _, filter := range filters {
		if match, ok := filter[OrCondition]; ok {
			res, err := addOrToQuery(s, q, match, join)
			if err != nil {
				return nil, err
			}
			sqlizer = append(sqlizer, res)
		} else if match, ok := filter[AndCondition]; ok {
			res, err := addAndToQuery(s, q, match, join)
			if err != nil {
				return nil, err
			}
			sqlizer = append(sqlizer, res)
		} else if b, ok := filter[BoolCondition]; ok {
			if b.(bool) {
				sqlizer = append(sqlizer, sq.Expr("(1=1)"))
			} else {
				sqlizer = append(sqlizer, sq.Expr("(1=0)"))
			}
		} else {
			key := filter["property"].(string)
			property, err := s.GetPropertyByID(key)
			if err != nil {
				return nil, err
			}

			var column string
			if join {
				column = makeColumn(s.GetDbTableName(), *property)
			} else {
				column = quote(key)
			}

			// TODO: add other operators
			value := filter["value"]
			substr, ok := value.(search.Search)
			if ok {
				sqlizer = append(sqlizer, Like{column: substr.Value})
				continue
			}

			value = parseBoolean(property.Type, value)
			switch filter["type"] {
			case "eq":
				sqlizer = append(sqlizer, sq.Eq{column: value})
			case "neq":
				sqlizer = append(sqlizer, sq.NotEq{column: value})
			default:
				panic("type has to be one of [eq, neq]")
			}
		}
	}
	return sqlizer, nil
}
