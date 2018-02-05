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

type ConverterParams struct {
	Config           string
	Output           string
	GoextPackage     string
	GoodiesPackage   string
	ResourcePackage  string
	InterfacePackage string
	SchemasPackage   string
	RawSuffix        string
	InterfaceSuffix  string
}

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
func Run(params ConverterParams) error {
	params.InterfaceSuffix = util.AddName(params.RawSuffix, params.InterfaceSuffix)
	all, err := readConfig(params.Config, "")
	if err != nil {
		return err
	}

	generated, err := schema.Convert(
		nil,
		all,
		params.RawSuffix,
		params.InterfaceSuffix,
		params.GoextPackage,
	)
	if err != nil {
		return err
	}

	if err = writeResult(
		generated.RawInterfaces,
		params.InterfacePackage,
		params.Output,
		"generated_interface.go",
	); err != nil {
		return err
	}

	if err = writeResult(
		generated.Interfaces,
		params.InterfacePackage,
		params.Output,
		"interface.go",
	); err != nil {
		return err
	}

	if err = writeResult(
		generated.Structs,
		params.ResourcePackage,
		params.Output,
		"raw.go",
	); err != nil {
		return err
	}

	if err = writeResult(
		generated.Implementations,
		params.ResourcePackage,
		params.Output,
		"implementation.go",
	); err != nil {
		return err
	}

	if err = writeResult(
		generated.Constructors,
		params.ResourcePackage,
		params.Output,
		"constructors.go",
	); err != nil {
		return err
	}

	if err = writeResult(
		generated.Crud,
		params.GoodiesPackage,
		params.Output,
		"crud.go",
	); err != nil {
		return err
	}

	if err = writeResult(
		generated.RawCrud,
		params.GoodiesPackage,
		params.Output,
		"raw_crud.go",
	); err != nil {
		return err
	}

	return writeResult(
		[]string{util.Const(generated.Names)},
		params.SchemasPackage,
		params.Output,
		"names.go",
	)
}
