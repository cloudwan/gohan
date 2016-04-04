// Copyright (C) 2016  Juniper Networks, Inc.
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

package main

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"unicode"

	"reflect"

	"github.com/codegangsta/cli"
	"github.com/flosch/pongo2"
)

func toUnderScoreCase(in *pongo2.Value, param *pongo2.Value) (*pongo2.Value, *pongo2.Error) {
	s := in.String()
	if len(s) == 0 {
		return nil, nil
	}
	parts := []string{}
	chars := []rune(s)
	var buffer bytes.Buffer
	for i := 0; i < len(chars)-1; i++ {
		if unicode.IsUpper(chars[i]) && unicode.IsLower(chars[i+1]) && buffer.String() != "" {
			parts = append(parts, buffer.String())
			buffer.Reset()
		}
		buffer.WriteRune(unicode.ToLower(chars[i]))
		if unicode.IsLower(chars[i]) && unicode.IsUpper(chars[i+1]) {
			parts = append(parts, buffer.String())
			buffer.Reset()
		}
	}
	buffer.WriteRune(unicode.ToLower(chars[len(chars)-1]))
	parts = append(parts, buffer.String())
	return pongo2.AsValue(strings.Join(parts, "_")), nil
}

func reflectType(in *pongo2.Value, param *pongo2.Value) (*pongo2.Value, *pongo2.Error) {
	i := in.Interface()
	v := reflect.ValueOf(i)
	t := v.Type()
	return pongo2.AsValue(t), nil
}

func exprToString(expr interface{}) string {
	switch a := expr.(type) {
	case *ast.Ident:
		return a.Name
	case *ast.SelectorExpr:
		return exprToString(a.X) + "." + exprToString(a.Sel)
	case *ast.ArrayType:
		return "[]" + exprToString(a.Elt)
	case *ast.InterfaceType:
		return "interface{}"
	case *ast.MapType:
		return "map[" + exprToString(a.Key) + "]" + exprToString(a.Value)
	case *ast.StarExpr:
		return "*" + exprToString(a.X)
	}
	return ""
}

func astType(in *pongo2.Value, param *pongo2.Value) (*pongo2.Value, *pongo2.Error) {
	i := in.Interface()
	return pongo2.AsValue(exprToString(i)), nil
}

func init() {
	pongo2.RegisterFilter("to_under_score_case", toUnderScoreCase)
	pongo2.RegisterFilter("type", reflectType)
	pongo2.RegisterFilter("astType", astType)
}

//using hacks in https://github.com/mattn/anko/blob/master/tool/makebuiltin.go

func pkgName(f string) string {
	file, err := parser.ParseFile(token.NewFileSet(), f, nil, parser.PackageClauseOnly)
	if err != nil || file == nil {
		return ""
	}
	return file.Name.Name
}

func isGoFile(dir os.FileInfo) bool {
	return !dir.IsDir() &&
		!strings.HasPrefix(dir.Name(), ".") && // ignore .files
		filepath.Ext(dir.Name()) == ".go"
}

func isPkgFile(dir os.FileInfo) bool {
	return isGoFile(dir) && !strings.HasSuffix(dir.Name(), "_test.go") // ignore test files
}

func parseDir(p string) (map[string]*ast.Package, error) {
	_, pn := filepath.Split(p)

	isGoDir := func(d os.FileInfo) bool {
		if isPkgFile(d) {
			name := pkgName(p + "/" + d.Name())
			return name == pn
		}
		return false
	}

	pkgs, err := parser.ParseDir(token.NewFileSet(), p, isGoDir, parser.ParseComments)
	if err != nil {
		return nil, err
	}
	return pkgs, nil
}

//Gen generate adapter code for specifed package
func Gen(pkg, template, export, exportPath string) {
	paths := []string{filepath.Join(os.Getenv("GOROOT"), "src")}
	if os.Getenv("GOPATH") == "" {
		fmt.Println("GOPATH isn't specified")
	}
	for _, p := range strings.Split(os.Getenv("GOPATH"), string(filepath.ListSeparator)) {
		paths = append(paths, filepath.Join(p, "src"))
	}
	for _, p := range paths {
		pkgPath := filepath.Join(p, pkg)
		pkgs, err := parseDir(pkgPath)
		if err != nil {
			continue
		}

		for _, pkgObj := range pkgs {

			for filePath, f := range pkgObj.Files {
				imports := []string{}
				funcs := []*ast.FuncDecl{}
				for _, d := range f.Decls {
					switch decl := d.(type) {
					case *ast.GenDecl:
						for _, s := range decl.Specs {
							switch spec := s.(type) {
							case *ast.ImportSpec:
								path := spec.Path.Value
								path = strings.Replace(path, "\"", "", -1)
								imports = append(imports, path)
							}
						}
					case *ast.FuncDecl:
						if decl.Recv != nil {
							continue
						}
						c := decl.Name.Name[0]
						if c < 'A' || c > 'Z' {
							continue
						}
						funcs = append(funcs, decl)
					}
				}
				if len(funcs) == 0 {
					continue
				}
				tpl, err := pongo2.FromFile(template)
				if err != nil {
					panic(err)
				}
				_, pkgName := filepath.Split(pkg)
				_, fileName := filepath.Split(filePath)
				outputFile := filepath.Join(exportPath, pkgName+"_"+fileName)
				output, err := tpl.Execute(
					pongo2.Context{"full_package": pkg, "package": pkgName,
						"funcs":          funcs,
						"imports":        imports,
						"export_package": export})

				if err != nil {
					panic(err)
				}
				re := regexp.MustCompile("\n+")
				output = re.ReplaceAllString(output, "\n")
				os.MkdirAll(exportPath, os.ModePerm)
				ioutil.WriteFile(outputFile, []byte(output), os.ModePerm)
				out, err := exec.Command("goimports", "./"+outputFile).CombinedOutput()
				if err != nil {
					fmt.Println(string(out))
					panic(err)
				}
				ioutil.WriteFile(outputFile, out, os.ModePerm)
			}
		}
		return
	}
}

//Run execute main command
func Run(name, usage, version string) {
	app := cli.NewApp()
	app.Name = name
	app.Usage = usage
	app.Version = version
	app.Commands = []cli.Command{
		getLibGenCommand(),
	}
	app.Run(os.Args)
}

func getLibGenCommand() cli.Command {
	return cli.Command{
		Name:      "generate_lib",
		ShortName: "genlib",
		Usage:     "Generate donburi lib code from plain go code",
		Description: `
helper command for Generate donburi lib code glue.(you need to apply go fmt for generated code)`,
		Flags: []cli.Flag{
			cli.StringFlag{Name: "package, p", Value: "", Usage: "Package"},
			cli.StringFlag{Name: "template, t", Value: "../templates/lib.tmpl", Usage: "Template File"},
			cli.StringFlag{Name: "export, e", Value: "autogen", Usage: "Package to export"},
			cli.StringFlag{Name: "export_path, ep", Value: "../autogen", Usage: "export path"},
		},
		Action: func(c *cli.Context) {
			pkg := c.String("package")
			template := c.String("template")
			export := c.String("export")
			exportPath := c.String("export_path")
			Gen(pkg, template, export, exportPath)
		},
	}
}

func main() {
	Run("donburi", "Donburi", "0.1.0")
}
