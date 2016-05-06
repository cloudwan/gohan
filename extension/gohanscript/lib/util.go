package lib

import (
	"os"
	"strings"

	uuidlib "github.com/twinj/uuid"
)

//UUID makes uuidv4
func UUID() string {
	return uuidlib.NewV4().String()
}

//FormatUUID format uuidv4
func FormatUUID(uuid string) (string, error) {
	u, err := uuidlib.Parse(uuid)
	return u.String(), err
}

//Env returns map of env values
func Env() map[string]interface{} {
	envs := os.Environ()
	env := map[string]interface{}{}
	for _, keyvalueString := range envs {
		keyvalue := strings.Split(keyvalueString, "=")
		var key, value string
		key = keyvalue[0]
		if len(keyvalue) > 1 {
			value = keyvalue[1]
		}
		env[key] = value
	}
	return env
}

//NormalizeMap normalizes data which can't be used for standard yaml or json
func NormalizeMap(data map[string]interface{}) map[string]interface{} {
	result := map[string]interface{}{}
	for key, value := range data {
		key = strings.Replace(key, ":", "_", -1)
		key = strings.Replace(key, "-", "_", -1)
		result[key] = value
	}
	return result
}
