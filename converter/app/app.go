// Copyright (C) 2017 NTT Innovation Institute, Inc.
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

package app

import (
	"github.com/cloudwan/gohan/converter/reader"
	"github.com/cloudwan/gohan/converter/schema"
	"github.com/cloudwan/gohan/converter/util"
	"github.com/cloudwan/gohan/converter/writer"
)

func readConfig(config, input string) ([]map[interface{}]interface{}, error) {
	if len(config) == 0 {
		return nil, nil
	}
	return reader.ReadAll(config, input)
}

func writeResult(data []string, packageName, outputPrefix, outputSuffix string) error {
	rawData := util.CollectData(packageName, data)
	if rawData == "" {
		return nil
	}
	file := writer.CreateWriter(util.TryToAddName(outputPrefix, outputSuffix))
	return file.Write(rawData)
}

// Run application
func Run(
	config,
	output,
	goextPackage,
	goodiesPackage,
	resourcePackage,
	interfacePackage,
	rawSuffix,
	interfaceSuffix string,
) error {
	interfaceSuffix = util.AddName(rawSuffix, interfaceSuffix)
	all, err := readConfig(config, "")
	if err != nil {
		return err
	}

	generated, err := schema.Convert(
		nil,
		all,
		rawSuffix,
		interfaceSuffix,
		goextPackage,
	)
	if err != nil {
		return err
	}

	if err = writeResult(
		generated.RawInterfaces,
		interfacePackage,
		output,
		"generated_interface.go",
	); err != nil {
		return err
	}

	if err = writeResult(
		generated.Interfaces,
		interfacePackage,
		output,
		"interface.go",
	); err != nil {
		return err
	}

	if err = writeResult(
		generated.Structs,
		resourcePackage,
		output,
		"raw.go",
	); err != nil {
		return err
	}

	if err = writeResult(
		generated.Implementations,
		resourcePackage,
		output,
		"implementation.go",
	); err != nil {
		return err
	}

	if err = writeResult(
		generated.Constructors,
		resourcePackage,
		output,
		"constructors.go",
	); err != nil {
		return err
	}

	if err = writeResult(
		generated.Crud,
		goodiesPackage,
		output,
		"crud.go",
	); err != nil {
		return err
	}

	return writeResult(
		generated.RawCrud,
		goodiesPackage,
		output,
		"raw_crud.go",
	)
}
