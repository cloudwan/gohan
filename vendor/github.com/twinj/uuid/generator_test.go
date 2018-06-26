package uuid

import (
	"crypto/rand"
	"errors"
	"gopkg.in/stretchr/testify.v1/assert"
	"sync"
	"testing"
	"time"
	"log"
	"io/ioutil"
)

var (
	nodeBytes = []byte{0xdd, 0xee, 0xff, 0xaa, 0xbb, 0x44, 0xcc}
)

var testGenerator, _ = NewGenerator(&GeneratorConfig{
	Logger: log.New(ioutil.Discard, "", 0),
})

func init() {
	config := &GeneratorConfig{
		Logger: log.New(ioutil.Discard, "", 0),
	}
	RegisterGenerator(config)
}

func TestGenerator_V1(t *testing.T) {
	u := testGenerator.NewV1()

	assert.Equal(t, VersionOne, u.Version(), "Expected correct version")
	assert.Equal(t, VariantRFC4122, u.Variant(), "Expected correct variant")
	assert.True(t, parseUUIDRegex.MatchString(u.String()), "Expected string representation to be valid")
}

func TestGenerator_V2(t *testing.T) {

	for _, v := range []SystemId{
		SystemIdUser, SystemIdGroup, SystemIdEffectiveUser, SystemIdEffectiveGroup,
		SystemIdCallerProcess, SystemIdCallerProcessParent,
	} {
		id := testGenerator.NewV2(v)

		assert.Equal(t, VersionTwo, id.Version(), "Expected correct version")
		assert.Equal(t, VariantRFC4122, id.Variant(), "Expected correct variant")
		assert.True(t, parseUUIDRegex.MatchString(id.String()), "Expected string representation to be valid")
		assert.Equal(t, byte(v), id.Bytes()[9], "Expected string representation to be valid")
	}
}

func TestRegisterGenerator(t *testing.T) {
	config := &GeneratorConfig{
		nil,
		func() Timestamp {
			return Timestamp(145876)
		}, 0,
		func() Node {
			return []byte{0x11, 0xaa, 0xbb, 0xaa, 0xbb, 0xcc}
		},
		func([]byte) (int, error) {
			return 58, nil
		}, nil, log.New(ioutil.Discard, "", 0)}

	once = new(sync.Once)
	RegisterGenerator(config)

	assert.Equal(t, config.Next(), generator.Next(), "These values should be the same")
	assert.Equal(t, config.Identifier(), generator.Identifier(), "These values should be the same")

	n1, err1 := generator.Random(nil)
	n, err := generator.Random(nil)
	assert.Equal(t, n1, n, "Values should be the same")
	assert.Equal(t, err, err1, "Values should be the same")
	assert.NoError(t, err)

	assert.True(t, didRegisterGeneratorPanic(config), "Should panic when invalid")
}

func didRegisterGeneratorPanic(config *GeneratorConfig) bool {
	return func() (didPanic bool) {
		defer func() {
			if recover() != nil {
				didPanic = true
			}
		}()

		RegisterGenerator(config)
		return
	}()
}

func TestNewGenerator(t *testing.T) {
	gen, _ := NewGenerator(nil)

	assert.NotNil(t, gen.Next, "There shoud be a default Next function")
	assert.NotNil(t, gen.Random, "There shoud be a default Random function")
	assert.NotNil(t, gen.HandleRandomError, "There shoud be a default HandleRandomError function")
	assert.NotNil(t, gen.Identifier, "There shoud be a default Id function")

	assert.Equal(t, findFirstHardwareAddress(), gen.Identifier(), "There shoud be the gieen Id function")

	gen, _ = NewGenerator(&GeneratorConfig{
		Identifier: func() Node {
			return Node{0xaa, 0xff}
		},
		Next: func() Timestamp {
			return Timestamp(2)
		},
		HandleRandomError: func([]byte, int, error) error {
			return nil
		},
		Random: func([]byte) (int, error) {
			return 1, nil
		},
		Resolution: 0,
		Logger: log.New(ioutil.Discard, "", 0),
	})

	assert.NotNil(t, gen.Next, "There shoud be a default Next function")
	assert.NotNil(t, gen.Random, "There shoud be a default Random function")
	assert.NotNil(t, gen.HandleRandomError, "There shoud be a default HandleRandomError function")
	assert.NotNil(t, gen.Identifier, "There shoud be a default Id function")

	n, err := gen.Random(nil)

	assert.Equal(t, Timestamp(2), gen.Next(), "There shoud be the given Next function")
	assert.Equal(t, 1, n, "There shoud be the given Random function")
	assert.NoError(t, err, "There shoud be the given Random function")
	assert.Nil(t, gen.HandleRandomError(nil, 0, nil), "There shoud be the given HandleRandomError function")
	assert.Equal(t, Node{0xaa, 0xff}, gen.Identifier(), "There shoud be the gieen Id function")

	gen, _ = NewGenerator(&GeneratorConfig{
		Identifier: func() Node {
			return []byte{0xaa, 0xff}
		},
		Next: nil,
		HandleRandomError: func([]byte, int, error) error {
			return nil
		},
		Random: func([]byte) (int, error) {
			return 1, nil
		},
		Resolution: 4096,
		Logger: log.New(ioutil.Discard, "", 0),
	})

	n, err = gen.Random(nil)

	assert.NotNil(t, gen.Next, "There shoud be a default Next function")
	assert.NotNil(t, gen.Random, "There shoud be a default Random function")
	assert.NotNil(t, gen.HandleRandomError, "There shoud be a default HandleRandomError function")
	assert.NotNil(t, gen.Identifier, "There shoud be a default Id function")

	assert.Equal(t, 1, n, "There shoud be the given Random function")
	assert.NoError(t, err, "There shoud be the given Random function")
	assert.Nil(t, gen.HandleRandomError(nil, 0, nil), "There shoud be the given HandleRandomError function")
	assert.Equal(t, Node{0xaa, 0xff}, gen.Identifier(), "There shoud be the gieen Id function")

}

func TestGeneratorInit(t *testing.T) {
	// A new time that is older than stored time should cause the sequence to increment
	now, node := registerTestGenerator(Now(), nodeBytes)
	storageStamp, err := registerSaver(now.Add(time.Second), node)

	assert.NoError(t, err)
	assert.NotNil(t, generator.Store, "Generator should not return an empty store")
	assert.True(t, generator.Timestamp < storageStamp, "Increment sequence when old timestamp newer than new")
	assert.Equal(t, Sequence(3), generator.Sequence, "Successful read should have incremented sequence")

	// Nodes not the same should generate a random sequence
	now, node = registerTestGenerator(Now(), nodeBytes)
	storageStamp, err = registerSaver(now.Sub(time.Second), []byte{0xaa, 0xee, 0xaa, 0xbb, 0x44, 0xcc})

	assert.NoError(t, err)
	assert.NotNil(t, generator.Store, "Generator should not return an empty store")
	assert.True(t, generator.Timestamp > storageStamp, "New timestamp should be newer than old")
	assert.NotEqual(t, Sequence(2), generator.Sequence, "Sequence should not be same as storage")
	assert.NotEqual(t, Sequence(3), generator.Sequence, "Sequence should not be incremented but be random")
	assert.Equal(t, generator.Node, node, generator.Sequence, "Node should be equal")

	now, node = registerTestGenerator(Now(), nodeBytes)

	// Random read error should alert user
	generator.Random = func(b []byte) (int, error) {
		return 0, errors.New("EOF")
	}

	storageStamp, err = registerSaver(now.Sub(time.Second), []byte{0xaa, 0xee, 0xaa, 0xbb, 0x44, 0xcc})

	assert.Error(t, err, "Read error should exist")

	now, _ = registerTestGenerator(Now(), nil)

	// Random read error should alert user
	generator.Random = func(b []byte) (int, error) {
		return 0, errors.New("EOF")
	}

	storageStamp, err = registerSaver(now.Sub(time.Second), []byte{0xaa, 0xee, 0xaa, 0xbb, 0x44, 0xcc})
	assert.Error(t, err, "Read error should exist")

	gen, err := NewGenerator(&GeneratorConfig{
		Random: func(b []byte) (int, error) {
			return 0, errors.New("EOF")
		},
		Logger: log.New(ioutil.Discard, "", 0),
	})

	assert.Error(t, err, "EOF")
	assert.Nil(t, gen)

	registerDefaultGenerator()
}

func TestGeneratorRead(t *testing.T) {
	// A new time that is older than stored time should cause the sequence to increment
	now := Now()
	i := 0

	timestamps := []Timestamp{
		now.Sub(time.Second),
		now.Sub(time.Second * 2),
	}

	var err error
	generator, err = NewGenerator(&GeneratorConfig{
		nil,
		func() Timestamp {
			return timestamps[i]
		}, 0,
		func() Node {
			return nodeBytes
		},
		rand.Read,
		nil, log.New(ioutil.Discard, "", 0)})
	assert.NoError(t, err)
	storageStamp, err := registerSaver(now.Add(time.Second), nodeBytes)
	assert.NoError(t, err)
	i++

	generator.read()

	assert.True(t, generator.Timestamp != 0, "Should not return an empty store")
	assert.True(t, generator.Timestamp != 0, "Should not return an empty store")
	assert.NotEmpty(t, generator.Node, "Should not return an empty store")

	assert.True(t, generator.Timestamp < storageStamp, "Increment sequence when old timestamp newer than new")
	assert.Equal(t, Sequence(4), generator.Sequence, "Successful read should have incremented sequence")

	// A new time that is older than stored time should cause the sequence to increment
	now, node := registerTestGenerator(Now().Sub(time.Second), nodeBytes)
	storageStamp, err = registerSaver(now.Add(time.Second), node)
	assert.NoError(t, err)
	generator.read()

	assert.NotEqual(t, 0, generator.Sequence, "Should return an empty store")
	assert.NotEmpty(t, generator.Node, "Should not return an empty store")

	// A new time that is older than stored time should cause the sequence to increment
	registerTestGenerator(Now().Sub(time.Second), nil)
	registerSaver(now.Add(time.Second), []byte{0xdd, 0xee, 0xff, 0xaa, 0xbb})

	generator.read()

	assert.NotEmpty(t, generator.Store, "Should not return an empty store")
	assert.NotEqual(t, []byte{0xdd, 0xee, 0xff, 0xaa, 0xbb}, generator.Node, "Should not return an empty store")

	registerDefaultGenerator()
}

func TestGeneratorRandom(t *testing.T) {
	registerTestGenerator(Now(), []byte{0xdd, 0xee, 0xff, 0xaa, 0xbb})

	b := make([]byte, 6)
	n, err := generator.Random(b)

	assert.NoError(t, err, "There should No be an error", err)
	assert.NotEmpty(t, b, "There should be random data in the slice")
	assert.Equal(t, 6, n, "Amount read should be same as length")

	generator.Random = func(b []byte) (int, error) {
		for i := 0; i < len(b); i++ {
			b[i] = byte(i)
		}
		return len(b), nil
	}

	b = make([]byte, 6)
	n, err = generator.Random(b)
	assert.NoError(t, err, "There should No be an error", err)
	assert.NotEmpty(t, b, "There should be random data in the slice")
	assert.Equal(t, 6, n, "Amount read should be same as length")

	generator.Random = func(b []byte) (int, error) {
		return 0, errors.New("EOF")
	}

	b = make([]byte, 6)
	c := []byte{}
	c = append(c, b...)

	n, err = generator.Random(b)
	assert.Error(t, err, "There should be an error", err)
	assert.Equal(t, 0, n, "Amount read should be same as length")
	assert.Equal(t, c, b, "Slice should be empty")

	id := NewV4()
	assert.NotEqual(t, id, Nil)

	generator.HandleRandomError = func([]byte, int, error) error {
		return errors.New("BOOM")
	}

	assert.Panics(t, didNewV4Panic, "NewV4 should panic when invalid")

	generator.HandleRandomError = func([]byte, int, error) error {
		generator.Random = func([]byte) (int, error) {
			return 1, nil
		}
		return nil
	}

	id = NewV4()
	assert.NotEqual(t, id, Nil)

	registerDefaultGenerator()
}

func didNewV4Panic() {
	NewV4()
}

func TestGeneratorSave(t *testing.T) {
	var err error
	registerTestGenerator(Now(), []byte{0xdd, 0xee, 0xff, 0xaa, 0xbb})
	generator.Do(func() {
		err = generator.init()
	})
	generator.read()
	generator.save()
	registerDefaultGenerator()
}

func TestGetHardwareAddress(t *testing.T) {
	a := findFirstHardwareAddress()
	assert.NotEmpty(t, a, "There should be a node id")
}

func registerTestGenerator(pNow Timestamp, pId Node) (Timestamp, Node) {
	generator, _ = NewGenerator(&GeneratorConfig{
		nil,
		func() Timestamp {
			return pNow
		}, 0,
		func() Node {
			return pId
		}, rand.Read,
		nil, log.New(ioutil.Discard, "", 0)},
	)
	once = new(sync.Once)
	return pNow, pId
}

func registerDefaultGenerator() {
	generator, _ = NewGenerator(&GeneratorConfig{
		Logger: log.New(ioutil.Discard, "", 0),
	})
}
