package uuid_test

import (
	. "github.com/myesui/uuid"
	"testing"
	"log"
	"io/ioutil"
)

func BenchmarkNewV1Resolution_0(b *testing.B) {
	run(b, 0)
}

func BenchmarkNewV1Resolution_1024(b *testing.B) {
	run(b, 1024) // default 1024
}

func BenchmarkNewV1Resolution_2048(b *testing.B) {
	run(b, 2048)
}

func BenchmarkNewV1Resolution_3072(b *testing.B) {
	run(b, 3072)
}

func BenchmarkNewV1Resolution_4096(b *testing.B) {
	run(b, 4096)
}

func BenchmarkNewV1Resolution_5120(b *testing.B) {
	run(b, 5120)
}

func BenchmarkNewV1Resolution_6144(b *testing.B) {
	run(b, 6144)
}

func BenchmarkNewV1Resolution_7168(b *testing.B) {
	run(b, 7168)
}

func BenchmarkNewV1Resolution_8192(b *testing.B) {
	run(b, 8192) // Best for my machine
}

func BenchmarkNewV1Resolution_9216(b *testing.B) {
	run(b, 9216)
}

func BenchmarkNewV1Resolution_18432(b *testing.B) {
	run(b, 18432)
}

func BenchmarkNewV1Resolution_36864(b *testing.B) {
	run(b, 36864)
}

var gen *Generator

func run(b *testing.B, resolution uint) {
	gen, _ = NewGenerator(&GeneratorConfig{Resolution: resolution, Logger: log.New(ioutil.Discard, "", 0)})
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		gen.NewV1() // Sets up initial store on first run
	}
	b.StopTimer()
	b.ReportAllocs()
}
