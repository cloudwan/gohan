# Golang (plugin) extensions

Gohan supports extensions written in golang and imported as shared libraries.
Golang plugins are supported starting from version 1.8 of golang. Note that
there exist a bug [1] which will be fixed in version 1.9.

## Why golang extensions as a plugin?

In contrast to golang extensions "as a callback" feature golang plugins are loadable at runtime
and do not require compilation together with gohan binary. It isn't always suitable to build
extensions at the same time as gohan. On the other hand, golang extensions via plugins are similar
to javascript extensions since they both do not require to be compiled with gohan.

It is easier to manage golang extension for each independent project. Additional benefit is that
this enables developers to implement event handler more intuitively.
 
When comparing to javascript, golang extensions give much higher performance since the code is
compiled to native code (not interpreted as javascript extensions) and run fast.
 
In contrast to javascript, golang is a typed language which turns out to be very helpful for
developers. It is also possible to use a debugger to find problems in code easily.

# Creating a simple golang extension

## Extension structure

An extension consists of an init.go file placed in loader subdirectory
and a number of implementation files.

# Schema definitions

To enable a golang extension, append extensions section to schema and specify
**code_type** as **goext**. Specify plugin library in url entry. Note that placing
golang code in code entry is not supported.

```yaml
  extensions:
  - id: example_golang_extension
    code_type: goext
    path: ""
    url: file://golang_ext.so
```

## Required exports

Each plugin must export **Init** function. Normally, this function is used to get the schema which
the extension is used for, register runtime types (raw and resource) and register some handlers.
An example of **Init** function:

```go
func Init(env goext.IEnvironment) error {
    // get schema for which this extension will be registered
    schema := env.Schemas().Find("resource_schema")

    // register runtime raw resource type - plain data, which is annotated and serializable
    schema.RegisterRawType(resource.RawResource{})

    // register runtime resource type - an object with methods, derived from BaseResource
    schema.RegisterType(resource.Resource{})

    // register pre-create handler for the schema with the default priority
    schema.RegisterEventHandler(goext.PreCreate, resource.HandlePreCreateEvent, goext.PriorityDefault)

    // register a custom event for the schema with the default priority
    schema.RegisterEventHandler(resource.CustomEventName, resource.HandleEventCustomEvent, goext.PriorityDefault)

    return nil
}
```

## Compilation and run

To compile an extension run:

```bash
go build -buildmode=plugin init.go
```

### Notes about building go plugins

In golang, plugins must be built with the same options as a binary that will load them. This is
a strict requirement meaning that:
 - all packages from GOPATH and vendor must be the same and in the same path when building the binary and the plugin
 - all build options must be the same for both build - it includes the **-race** flag

### Notes about building go plugins

In golang, plugins must be built with the same options as a binary that will load them. This is
a strict requirement meaning that:
 - all packages from GOPATH and vendor must be the same and in the same path when building the binary and the plugin
 - all build options must be the same for both build - it includes the **-race** flag

## Extension environment

An environment is passed to Init function. It consists of modules which are available
for a golang extension: core, logger, schemas, sync. 

## Event handling

In golang extension one can register a global handler for a named event
and a schema handler for a particular schema and event.
Examples:

```go
func HandlePreUpdateInTransaction(context goext.Context, resource goext.Resource, env goext.IEnvironment) error {
	myResource := resource.(*RawMyResource)

	// some operations on the resource

	return nil
}
[...]
type RawMyResource struct { // according to the schema
	ID          string           `db:"id"`          // mandatory field (nullable)
	name        goext.NullString `db:"name"`        // optional field (nullable)
	Description goext.NullString `db:"description"` // optional field (nullable)
}
[...]
func Init(env goext.IEnvironment) error {
    schema := env.Schemas().Find("my_schema")
    schema.RegisterRawType(resource.RawMyResource{})
    schema.RegisterEventHandler(goext.PreUpdate, HandlePreUpdateInTransaction)
    return nil
}
```

## Nullable types

Go plugin based extensions support in Gohan gives a number of nullable types to support
optional fields (nullable). A field is nullable if it is defined so in a schema.

There are four nullable types in Gohan:
 - **NullString**: wrapped string type
 - **NullBool**: wrapped bool type (equivalently a three-state bool)
 - **NullInt**: wrapped int type
 - **NullFloat**: wrapped float64 type

Each type has two properties:
 - **Value**: represents the value iff Valid is set to true
 - **Valid**: indicates that the Value is available

There are constructors for each type:
 - **MakeNullString(value string)**: create a new valid nullable string with given value
 - **MakeNullBool(value bool)**: create a new valid nullable bool with given value
 - **MakeNullInt(value int)**: create a new valid nullable int with given value
 - **MakeNullFloat(value float64)**: create a new valid nullable float64 with given value

A null type is obtained by:
 - **NullString{}**
 - **NullBool{}**
 - **NullInt{}**
 - **NullFloat{}**

# Testing golang extensions

Tests for golang extensions are plugin libraries. In a test case, used defined which schemas
and plugins must be loaded for the test. Required schemas are defined by exporting symbol **Schemas**:
```go
func Schemas() []string {
	return []string{
		"../todo/entry.yaml",
		"../todo/link.yaml",
	}
}
```
Required plugins are defined by exporting **Binaries** symbol:
```go
func Binaries() []string {
	return []string{"../example.so"}
}
```
Main function of each test is **Test** in which, gomega test cases are described:
```go
func Test(env goext.IEnvironment) {
	env.Logger().Notice("Running example test suite")

	Describe("Example tests", func() {
		var (
			schema goext.ISchema
			entry  *todo.Entry
		)

		BeforeEach(func() {
			schema = env.Schemas().Find("entry")
			Expect(schema).To(Not(BeNil()))

			entry = &todo.Entry{
				ID:          env.Core().NewUUID(),
				Name:        "example name",
				Description: "example description",
				TenantID:    "admin",
			}
		})

		AfterEach(func() {
			env.Reset()
		})

		It("Smoke test CRUD", func() {
			Expect(schema.CreateRaw(entry, nil)).To(Succeed())
			Expect(schema.UpdateRaw(entry, nil)).To(Succeed())
			Expect(schema.DeleteRaw(goext.Filter{"id": entry.ID}, nil)).To(Succeed())
		})

		It("Should change name in pre_update event", func() {
			Expect(schema.CreateRaw(entry, nil)).To(Succeed())
			entry.Name = "random name"
			Expect(schema.UpdateRaw(entry, nil)).To(Succeed())
			Expect(entry.Name).To(Equal("name changed in pre_update event"))
		})
	})
}

```

# References:

[1] https://github.com/golang/go/commit/758d078fd5ab423d00f5a46028139c1d13983120
