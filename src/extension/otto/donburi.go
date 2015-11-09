// Copyright (C) 2015 NTT Innovation Institute, Inc.
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

package otto

import (
	"github.com/cloudwan/gohan/util"
	"gopkg.in/yaml.v2"
)

//runDonburi converts Donburi script to javascript
func (env *Environment) runDonburi(yamlCode string) error {
	vm := env.VM
	var donburi map[interface{}]interface{}
	err := yaml.Unmarshal([]byte(yamlCode), &donburi)
	if err != nil {
		return err
	}
	donburiInVM, err := vm.ToValue(util.DecodeYAMLLibObject(donburi))
	if err != nil {
		return err
	}
	_, err = vm.Call("donburi.run", nil, donburiInVM)
	return err
}
