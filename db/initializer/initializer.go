package initializer

import (
	l "github.com/cloudwan/gohan/log"
	"github.com/cloudwan/gohan/schema"
	"github.com/cloudwan/gohan/util"
)

var log = l.NewLogger()

type Initializer struct {
	data map[string]interface{}
}

func NewInitializer(filePath string) (*Initializer, error) {
	i := &Initializer{}
	if err := i.load(filePath); err != nil {
		return nil, err
	}

	return i, nil
}

func (db *Initializer) getTable(s *schema.Schema) []interface{} {
	rawTable, ok := db.data[s.GetDbTableName()]
	if ok {
		return rawTable.([]interface{})
	}
	newTable := []interface{}{}
	db.data[s.GetDbTableName()] = newTable
	return newTable
}

func (db *Initializer) load(filePath string) error {
	data, err := util.LoadMap(filePath)
	if err != nil {
		db.data = map[string]interface{}{}
		return err
	}
	db.data = data
	return nil
}

func (db *Initializer) List(s *schema.Schema) (list []*schema.Resource, total uint64, err error) {
	table := db.getTable(s)
	for _, rawData := range table {
		data := rawData.(map[string]interface{})
		resource, errLoad := schema.NewResource(s, data)
		if errLoad != nil {
			log.Warning("Can't load %s %s", resource, errLoad)
			err = errLoad
			return
		}
		list = append(list, resource)
	}
	total = uint64(len(list))
	return
}
