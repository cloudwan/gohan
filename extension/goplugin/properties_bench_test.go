package goplugin_test

import (
	"io/ioutil"
	"testing"

	"github.com/cloudwan/gohan/extension/goext"
	"github.com/cloudwan/gohan/extension/goplugin"
	"github.com/cloudwan/gohan/log"
	"github.com/cloudwan/gohan/schema"
)

var (
	resultMap map[string]goext.Property
)

const GoextSchemaPath = "test_data/test_schema.yaml"

func load(b *testing.B, schemaPath string, schemaId goext.SchemaID) goext.ISchema {
	log.SetUpBasicLogging(ioutil.Discard, log.CliFormat)

	schemaManager := schema.GetManager()
	if err := schemaManager.LoadSchemaFromFile(schemaPath); err != nil {
		b.Fatal("loading failed", schemaPath, err)
	}

	env := goplugin.NewEnvironment("test", nil, nil)
	if err := env.Start(); err != nil {
		b.Fatal("starting failed", err)
	}

	defer b.ResetTimer()
	return env.Schemas().Find(schemaId)
}

func benchmark(b *testing.B, schemaPath string, schemaId goext.SchemaID) {
	s := load(b, schemaPath, schemaId)

	var r map[string]goext.Property
	for n := 0; n < b.N; n++ {
		r = s.Properties()
	}
	resultMap = r
}

func BenchmarkProperties_Test(b *testing.B) {
	benchmark(b, GoextSchemaPath, "test")
}

func BenchmarkProperties_TestSuite(b *testing.B) {
	benchmark(b, GoextSchemaPath, "test_suite")
}

func BenchmarkProperties_TestSchemaNoExt(b *testing.B) {
	benchmark(b, GoextSchemaPath, "test_schema_no_ext")
}
