package gls

import (
	"sync"
	"testing"

	"github.com/tylerb/is"
)

func TestGLS(t *testing.T) {
	is := is.New(t)

	Set("key", "value")
	v := Get("key")
	is.NotNil(v)
	is.Equal(v, "value")

	Cleanup()
}

func TestGLSWith(t *testing.T) {
	is := is.New(t)

	With(Values{"key": "value"}, func() {
		v := Get("key")
		is.NotNil(v)
		is.Equal(v, "value")
	})

	v := Get("key")
	is.Nil(v)
}

func TestGLSSetValues(t *testing.T) {
	is := is.New(t)

	Set("key", "value")
	v := Get("key")
	is.NotNil(v)
	is.Equal(v, "value")

	SetValues(Values{"answer": 42})
	v = Get("key")
	is.Nil(v)

	v = Get("answer")
	is.NotNil(v)
	is.Equal(v, 42)

	Cleanup()
}

func TestGLSGo(t *testing.T) {
	is := is.New(t)

	var wg sync.WaitGroup
	wg.Add(3)

	Set("key", "value")

	Go(func() {
		v := Get("key")
		is.NotNil(v)
		is.Equal(v, "value")
		Go(func() {
			v := Get("key")
			is.NotNil(v)
			is.Equal(v, "value")
			Set("answer", 42)
			Go(func() {
				v := Get("key")
				is.NotNil(v)
				is.Equal(v, "value")
				v = Get("answer")
				is.NotNil(v)
				is.Equal(v, 42)
				wg.Done()
			})
			wg.Done()
		})
		wg.Done()
	})

	v := Get("key")
	is.NotNil(v)
	is.Equal(v, "value")

	wg.Wait()

	Cleanup()
}

func BenchmarkGLSSet(b *testing.B) {
	for i := 0; i < b.N; i++ {
		Set("key", "value")
	}
	Cleanup()
}

func BenchmarkGLSGet(b *testing.B) {
	b.StopTimer()
	Set("key", "value")
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		Get("key")
	}
	Cleanup()
}
