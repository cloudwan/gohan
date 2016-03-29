package lib

import (
	"github.com/cloudwan/gohan/util"
)

//FetchContent fetch contents from arbitrary path
func FetchContent(path string) (interface{}, error) {
	return util.LoadFile(path)
}

//SaveContent saves data for path
func SaveContent(path string, data interface{}) error {
	return util.SaveFile(path, data)
}
