package uuid

import (
	"gopkg.in/stretchr/testify.v1/assert"
	"testing"
)

func TestUuid_Bytes(t *testing.T) {
	id := UUID{}
	copy(id[:], NameSpaceDNS.Bytes())
	assert.Equal(t, id.Bytes(), NameSpaceDNS.Bytes(), "Bytes should be the same")
}

func TestUuid_Size(t *testing.T) {
	id := UUID{}
	assert.Equal(t, 16, id.Size(), "The size of the array should be sixteen")
}

func TestUuid_String(t *testing.T) {
	id := UUID{}
	copy(id[:], uuidBytes)
	assert.Equal(t, idString, id.String(), "The Format given should match the output")
}

func TestUuid_Variant(t *testing.T) {
	id := UUID{}
	copy(id[:], uuidBytes)
	bytes := id[:]

	for _, v := range uuidVariants {
		for i := 0; i <= 255; i++ {
			bytes[variantIndex] = byte(i)
			id := createMarshaler(bytes[:], 4, v)
			b := id[variantIndex] >> 4
			tVariantConstraint(v, b, t)
			assert.Equal(t, v, id.Variant(), "%x does not resolve to %x", id.Variant(), v)
		}
	}

	assert.True(t, didMarshalerSetVariantPanic(bytes[:]), "Array creation should panic  if invalid variant")
}

func didMarshalerSetVariantPanic(bytes []byte) bool {
	return func() (didPanic bool) {
		defer func() {
			if recover() != nil {
				didPanic = true
			}
		}()

		createMarshaler(bytes[:], 4, 0xbb)
		return
	}()
}

func TestUuid_Version(t *testing.T) {
	id := UUID{}
	bytes := UUID{}
	bytes.unmarshal(uuidBytes[:])

	assert.Equal(t, VersionUnknown, id.Version(), "The version should be 0")

	for v := 0; v < 16; v++ {
		for i := 0; i <= 255; i++ {
			bytes[versionIndex] = byte(i)
			id.unmarshal(bytes[:])
			setVersion(&id[versionIndex], v)
			if v > 0 && v < 6 {
				assert.Equal(t, Version(v), id.Version(), "%x does not resolve to %x", id.Version(), v)
			} else {
				assert.Equal(t, Version(v), getVersion(id), "%x does not resolve to %x", getVersion(id), v)
			}
		}
	}
}

func TestImmutable_Bytes(t *testing.T) {
	b := make([]byte, length)
	copy(b[:], NameSpaceDNS.Bytes())

	id := Immutable(b)

	assert.Equal(t, NameSpaceDNS.Bytes(), id.Bytes())
}

func TestImmutable_Size(t *testing.T) {
	assert.Equal(t, 16, Nil.Size(), "The size of the array should be sixteen")
}

func TestImmutable_String(t *testing.T) {
	id := Immutable(uuidBytes)
	assert.Equal(t, idString, id.String(), "The Format given should match the output")
}

func TestImmutable_Variant(t *testing.T) {
	bytes := UUID{}
	bytes.unmarshal(uuidBytes[:])

	for _, v := range uuidVariants {
		for i := 0; i <= 255; i++ {
			bytes[variantIndex] = byte(i)
			id := createMarshaler(bytes[:], 4, v)
			b := id[variantIndex] >> 4
			tVariantConstraint(v, b, t)
			id2 := Immutable(id[:])
			assert.Equal(t, v, id2.Variant(), "%x does not resolve to %x", id2.Variant(), v)
		}
	}
}

func TestImmutable_Version(t *testing.T) {

	id := UUID{}
	bytes := UUID{}
	bytes.unmarshal(uuidBytes[:])

	for v := 0; v < 16; v++ {
		for i := 0; i <= 255; i++ {
			bytes[versionIndex] = byte(i)
			id.unmarshal(bytes[:])
			setVersion(&id[versionIndex], v)
			id2 := Immutable(id[:])

			if v > 0 && v < 6 {
				assert.Equal(t, Version(v), id2.Version(), "%x does not resolve to %x", id2.Version(), v)
			} else {
				assert.Equal(t, Version(v), getVersion(UUID(id)), "%x does not resolve to %x", getVersion(UUID(id)), v)
			}
		}
	}
}

func TestUuid_MarshalBinary(t *testing.T) {
	id := UUID{}
	id.unmarshal(uuidBytes)
	bytes, err := id.MarshalBinary()
	assert.Nil(t, err, "There should be no error")
	assert.Equal(t, uuidBytes, bytes, "Byte should be the same")
}

func TestUuid_UnmarshalBinary(t *testing.T) {

	//assert.True(t, didUnmarshalPanic(), "Should panic")

	u := UUID{}
	err := u.UnmarshalBinary([]byte{1, 2, 3, 4, 5})
	assert.Error(t, err, "Expect length error")

	err = u.UnmarshalBinary(uuidBytes[:])

	u = UUID{}

	err = u.UnmarshalBinary(uuidBytes[:])

	assert.Nil(t, err, "There should be no error but got %s", err)

	for k, v := range namespaces {
		id, _ := Parse(v)
		u = UUID{}
		u.UnmarshalBinary(id.Bytes())

		assert.Equal(t, id.Bytes(), u.Bytes(), "The array id should equal the uuid id")
		assert.Equal(t, k.Bytes(), u.Bytes(), "The array id should equal the uuid id")
	}
}

func didUnmarshalPanic(id UUID, data []byte) bool {
	return func() (didPanic bool) {
		defer func() {
			if recover() != nil {
				didPanic = true
			}
		}()
		id.UnmarshalBinary(data)
		return
	}()
}

func TestUuid_Scan(t *testing.T) {
	var v UUID
	assert.True(t, IsNil(v))

	err := v.Scan(nil)
	assert.NoError(t, err, "When nil there should be no error")
	assert.True(t, IsNil(v))

	err = v.Scan("")
	assert.NoError(t, err, "When nil there should be no error")
	assert.True(t, IsNil(v))

	var v2 UUID
	err = v2.Scan(NameSpaceDNS.Bytes())
	assert.NoError(t, err, "When nil there should be no error")
	assert.Equal(t, NameSpaceDNS.Bytes(), v2.Bytes(), "Values should be the same")

	err = v.Scan(NameSpaceDNS.String())
	assert.NoError(t, err, "When nil there should be no error")
	assert.Equal(t, NameSpaceDNS.String(), v.String(), "Values should be the same")

	var v3 UUID
	err = v3.Scan([]byte(NameSpaceDNS.String()))
	assert.NoError(t, err, "When []byte represents string should be no error")
	assert.Equal(t, NameSpaceDNS.String(), v3.String(), "Values should be the same")

	err = v.Scan(22)
	assert.Error(t, err, "When wrong type should error")
}

func TestUuid_Value(t *testing.T) {
	var v UUID
	assert.True(t, IsNil(v))

	id, err := v.Value()
	assert.Nil(t, id, "There should be no driver value")
	assert.NoError(t, err, "There should be no error")

	ns := NameSpaceDNS.UUID()

	id, err = ns.Value()
	assert.NotNil(t, id, "There should be a valid driver value")
	assert.NoError(t, err, "There should be no error")
}

func getVersion(pId UUID) Version {
	return Version(pId[versionIndex] >> 4)
}

func createMarshaler(data []byte, version int, variant uint8) UUID {
	o := UUID{}
	o.unmarshal(data)
	setVersion(&o[versionIndex], version)
	setVariant(&o[variantIndex], variant)
	return o
}

func setVersion(byte *byte, version int) {
	*byte &= 0x0f
	*byte |= uint8(version << 4)
}

func setVariant(byte *byte, variant uint8) {
	switch variant {
	case VariantRFC4122:
		*byte &= variantSet
	case VariantFuture, VariantMicrosoft:
		*byte &= 0x1F
	case VariantNCS:
		*byte &= 0x7F
	default:
		panic("uuid: invalid variant mask")
	}
	*byte |= variant
}
