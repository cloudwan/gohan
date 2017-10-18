# Converter

Converter is a tool that can be used to generate a code for golang extensions.

# Usage

```bash
NAME:
   converter - Generates code used by golang extensions

USAGE:
   command converter [command options] [arguments...]

DESCRIPTION:
   gohan converter [path to file with schemas] [flags...]

Converter generates code from yaml schemas.
Generated code:
	* Definition of structs representing objects from each schema
	* Interfaces for getters and setters for these objects
	* Implementation of these interfaces by pointers to generated structs
	* Interfaces that can be extended
	* Constructors for objects with default values
	* Database functions for generated structs (fetch, list)
ARGUMENTS:
	There is one argument - path to file with yaml schemas


OPTIONS:
   --goext-package "goext"		Package name for golang extension interfaces
   --crud-package "goodies"		Package name for crud functions
   --raw-package "resources"		Package name for raw structs
   --interface-package "interfaces"	Package name for interfaces
   --output, -o 			Prefix add to output files
   --raw-suffix 			Suffix added to raw struct names
   --interface-suffix "gen"		Suffix added to generated interface names
```

## Arguments

### File with schemas

File with schemas should be a yaml file containing list named "schemas".
This list should contain path to yaml schemas.

```yaml
schemas:
- test_schema.yaml
```

### Flags

- --goext-package<br>
  default: "goext"<br>
  Package name for golang extension interfaces

- --crud-package<br>
  default: "goodies"<br>
  Package name for files:
  - crud.go
  - raw_crud.go

- --raw-package<br>
  default: "resources"<br>
  Package name for files:
  - raw.go
  - implementation.go
  - constructors.go

- --interface-package<br>
  default: "interfaces"<br>
  Package name for files:
  - interface.go
  - generated_interface.go

- --raw-suffix<br>
  Suffix added to the name of each generated type

- --interface-suffix<br>
  default: "gen"<br>
  Suffix added to the name of each generated interface type

- --output, -o<br>
  If provided (value) output will be generated to files [value]_[name].go,
  otherwise all data will be generated to stdout.

# Generated files

Example schema:

```yaml
schemas:
- id: base
  schema:
    type: object
    properties:
      id:
        type: string
        default: test
      num:
        type: number
      field:
        type: array
        items:
          type: integer
- id: derived
  extends:
  - base
  parent: parent
  schema:
    type: object
    properties:
      object:
        default:
          a: abc
          b: 123
        type: object
        properties:
          a:
            type: string
          b:
            type: integer
          c:
            type: boolean
    required:
    - num
    - object
```

If output flag is provided the following files will be generated:

- [output]_raw.go<br>

File with definition of structs representing schemas and json properties.

```go
package resources

type Base struct {
    Field []int `db:"field" json:"field"`
    ID string `db:"id" json:"id"`
    Num goext.NullFloat `db:"num" json:"num,omitempty"`
}

type Derived struct {
    Field []int `db:"field" json:"field"`
    ID string `db:"id" json:"id"`
    Num float64 `db:"num" json:"num"`
    Object *DerivedObject `db:"object" json:"object"`
    ParentID string `db:"parent_id" json:"parent_id"`
}

type DerivedObject struct {
    A string `json:"a,omitempty"`
    B int `json:"b,omitempty"`
    C bool `json:"c,omitempty"`
}
```

- [output]_constructors.go<br>

File with constructors for resource with default values.

```go
package resources

func MakeBase() *Base {
    return &Base{
        ID: "test",
    }
}

func MakeDerived() *Derived {
    return &Derived{
        ID: "test",
        Object: MakeDerivedObject(),
    }
}

func MakeDerivedObject() *DerivedObject {
    return &DerivedObject{
        A: "abc",
        B: 123,
    }
}
```

- [output]_genereted_interface.go<br>

File with interfaces of getters and setters for structs that are in [output]_raw.go file

```go
package interfaces

type IBaseGen interface {
    GetField() []int
    SetField([]int)
    GetID() string
    SetID(string)
    GetNum() goext.NullFloat
    SetNum(goext.NullFloat)
}

type IDerivedGen interface {
    GetField() []int
    SetField([]int)
    GetID() string
    SetID(string)
    GetNum() float64
    SetNum(float64)
    GetObject() IDerivedObjectGen
    SetObject(IDerivedObjectGen)
    GetParentID() string
    SetParentID(string)
}

type IDerivedObjectGen interface {
    GetA() string
    SetA(string)
    GetB() int
    SetB(int)
    GetC() bool
    SetC(bool)
}
```

- [output]_implementation.go<br>

File with implementation of generater interfaces by generated structs.

```go
package resources

func (base *Base) GetField() []int {
	return base.Field
}

func (base *Base) SetField(field []int) {
	base.Field = field
}

func (base *Base) GetID() string {
	return base.ID
}

func (base *Base) SetID(id string) {
	base.ID = id
}

func (base *Base) GetNum() goext.NullFloat {
	return base.Num
}

func (base *Base) SetNum(num goext.NullFloat) {
	base.Num = num
}

func (derived *Derived) GetField() []int {
	return derived.Field
}

func (derived *Derived) SetField(field []int) {
	derived.Field = field
}

func (derived *Derived) GetID() string {
	return derived.ID
}

func (derived *Derived) SetID(id string) {
	derived.ID = id
}

func (derived *Derived) GetNum() float64 {
	return derived.Num
}

func (derived *Derived) SetNum(num float64) {
	derived.Num = num
}

func (derived *Derived) GetObject() IDerivedObjectGen {
	return derived.Object
}

func (derived *Derived) SetObject(object IDerivedObjectGen) {
	derived.Object, _ = object.(*DerivedObject)
}

func (derived *Derived) GetParentID() string {
	return derived.ParentID
}

func (derived *Derived) SetParentID(parentID string) {
	derived.ParentID = parentID
}

func (derivedObject *DerivedObject) GetA() string {
	return derivedObject.A
}

func (derivedObject *DerivedObject) SetA(a string) {
	derivedObject.A = a
}

func (derivedObject *DerivedObject) GetB() int {
	return derivedObject.B
}

func (derivedObject *DerivedObject) SetB(b int) {
	derivedObject.B = b
}

func (derivedObject *DerivedObject) GetC() bool {
	return derivedObject.C
}

func (derivedObject *DerivedObject) SetC(c bool) {
	derivedObject.C = c
}
```

- [output]_interface.go

File with interfaces that can be extended

```go
package interfaces

type IBase interface {
    IBaseGen
}

type IDerived interface {
    IDerivedGen
}

type IDerivedObject interface {
    IDerivedObjectGen
}
```

- [output]_raw_crud.go

File with fetch and list functions for raw resources

```go
package goodies

func FetchRawBase(schema goext.ISchema, id string, context goext.Context) (*Base, error) {
	result, err := schema.FetchRaw(id, context)
	if err != nil {
		return nil, err
	}
	return result.(*Base), nil
}

func LockFetchRawBase(schema goext.ISchema, id string, context goext.Context, policy goext.LockPolicy) (*Base, error) {
	result, err := schema.LockFetchRaw(id, context, policy)
	if err != nil {
		return nil, err
	}
	return result.(*Base), nil
}

func ListRawBase(schema goext.ISchema, filter goext.Filter, paginator *goext.Paginator, context goext.Context) ([]*Base, error) {
	list, err := schema.ListRaw(filter, paginator, context)
	if err != nil {
		return nil, err
	}
	result := make([]*Base, len(list))
	for i, object := range list {
		result[i] = object.(*Base)
	}
	return result, nil
}

func LockListRawBase(schema goext.ISchema, filter goext.Filter, paginator *goext.Paginator, context goext.Context, policy goext.LockPolicy) ([]*Base, error) {
	list, err := schema.LockListRaw(filter, paginator, context, policy)
	if err != nil {
		return nil, err
	}
	result := make([]*Base, len(list))
	for i, object := range list {
		result[i] = object.(*Base)
	}
	return result, nil
}

func FetchRawDerived(schema goext.ISchema, id string, context goext.Context) (*Derived, error) {
	result, err := schema.FetchRaw(id, context)
	if err != nil {
		return nil, err
	}
	return result.(*Derived), nil
}

func LockFetchRawDerived(schema goext.ISchema, id string, context goext.Context, policy goext.LockPolicy) (*Derived, error) {
	result, err := schema.LockFetchRaw(id, context, policy)
	if err != nil {
		return nil, err
	}
	return result.(*Derived), nil
}

func ListRawDerived(schema goext.ISchema, filter goext.Filter, paginator *goext.Paginator, context goext.Context) ([]*Derived, error) {
	list, err := schema.ListRaw(filter, paginator, context)
	if err != nil {
		return nil, err
	}
	result := make([]*Derived, len(list))
	for i, object := range list {
		result[i] = object.(*Derived)
	}
	return result, nil
}

func LockListRawDerived(schema goext.ISchema, filter goext.Filter, paginator *goext.Paginator, context goext.Context, policy goext.LockPolicy) ([]*Derived, error) {
	list, err := schema.LockListRaw(filter, paginator, context, policy)
	if err != nil {
		return nil, err
	}
	result := make([]*Derived, len(list))
	for i, object := range list {
		result[i] = object.(*Derived)
	}
	return result, nil
}
```

- [output]_crud.go

File with fetch and list functions for intrefaces

```go
package goodies

func FetchBase(schema goext.ISchema, id string, context goext.Context) (IBase, error) {
	result, err := schema.Fetch(id, context)
	if err != nil {
		return nil, err
	}
	return result.(IBase), nil
}

func LockFetchBase(schema goext.ISchema, id string, context goext.Context, policy goext.LockPolicy) (IBase, error) {
	result, err := schema.LockFetch(id, context, policy)
	if err != nil {
		return nil, err
	}
	return result.(IBase), nil
}

func ListBase(schema goext.ISchema, filter goext.Filter, paginator *goext.Paginator, context goext.Context) ([]IBase, error) {
	list, err := schema.List(filter, paginator, context)
	if err != nil {
		return nil, err
	}
	result := make([]IBase, len(list))
	for i, object := range list {
		result[i] = object.(IBase)
	}
	return result, nil
}

func LockListBase(schema goext.ISchema, filter goext.Filter, paginator *goext.Paginator, context goext.Context, policy goext.LockPolicy) ([]IBase, error) {
	list, err := schema.LockList(filter, paginator, context, policy)
	if err != nil {
		return nil, err
	}
	result := make([]IBase, len(list))
	for i, object := range list {
		result[i] = object.(IBase)
	}
	return result, nil
}

func FetchDerived(schema goext.ISchema, id string, context goext.Context) (IDerived, error) {
	result, err := schema.Fetch(id, context)
	if err != nil {
		return nil, err
	}
	return result.(IDerived), nil
}

func LockFetchDerived(schema goext.ISchema, id string, context goext.Context, policy goext.LockPolicy) (IDerived, error) {
	result, err := schema.LockFetch(id, context, policy)
	if err != nil {
		return nil, err
	}
	return result.(IDerived), nil
}

func ListDerived(schema goext.ISchema, filter goext.Filter, paginator *goext.Paginator, context goext.Context) ([]IDerived, error) {
	list, err := schema.List(filter, paginator, context)
	if err != nil {
		return nil, err
	}
	result := make([]IDerived, len(list))
	for i, object := range list {
		result[i] = object.(IDerived)
	}
	return result, nil
}

func LockListDerived(schema goext.ISchema, filter goext.Filter, paginator *goext.Paginator, context goext.Context, policy goext.LockPolicy) ([]IDerived, error) {
	list, err := schema.LockList(filter, paginator, context, policy)
	if err != nil {
		return nil, err
	}
	result := make([]IDerived, len(list))
	for i, object := range list {
		result[i] = object.(IDerived)
	}
	return result, nil
}
```

# Limitations

- All schemas should have type object or abstract
- Only possible default value for arrays is empty array