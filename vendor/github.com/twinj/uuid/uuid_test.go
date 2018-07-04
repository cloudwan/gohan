package uuid

import (
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/binary"
	"gopkg.in/stretchr/testify.v1/assert"
	"hash"
	"net/url"
	"runtime"
	"testing"
)

var (
	goLang = "https://google.com/golang.org?q=golang"

	uuidBytes = []byte{
		0xaa, 0xcf, 0xee, 0x12,
		0xd4, 0x00,
		0x27, 0x23,
		0x00,
		0xd3,
		0x23, 0x12, 0x4a, 0x11, 0x89, 0xbb,
	}

	idString = "aacfee12-d400-2723-00d3-23124a1189bb"

	uuidVariants = []byte{
		VariantNCS, VariantRFC4122, VariantMicrosoft, VariantFuture,
	}

	namespaces = make(map[Implementation]string)

	invalidHexStrings = [...]string{
		"foo",
		"6ba7b814-9dad-11d1-80b4-",
		"6ba7b814--9dad-11d1-80b4--00c04fd430c8",
		"6ba7b814-9dad7-11d1-80b4-00c04fd430c8999",
		"{6ba7b814-9dad-1180b4-00c04fd430c8",
		"{6ba7b814--11d1-80b4-00c04fd430c8}",
		"urn:uuid:6ba7b814-9dad-1666666680b4-00c04fd430c8",
	}

	validHexStrings = [...]string{
		"6ba7b8149dad-11d1-80b4-00c04fd430c8}",
		"{6ba7b8149dad-11d1-80b400c04fd430c8}",
		"{6ba7b814-9dad11d180b400c04fd430c8}",
		"6ba7b8149dad-11d1-80b4-00c04fd430c8",
		"6ba7b814-9dad11d1-80b4-00c04fd430c8",
		"6ba7b814-9dad-11d180b4-00c04fd430c8",
		"6ba7b814-9dad-11d1-80b400c04fd430c8",
		"6ba7b8149dad11d180b400c04fd430c8",
		"6ba7b814-9dad-11d1-80b4-00c04fd430c8",
		"{6ba7b814-9dad-11d1-80b4-00c04fd430c8}",
		"{6ba7b814-9dad-11d1-80b4-00c04fd430c8",
		"6ba7b814-9dad-11d1-80b4-00c04fd430c8}",
		"(6ba7b814-9dad-11d1-80b4-00c04fd430c8)",
		"urn:uuid:6ba7b814-9dad-11d1-80b4-00c04fd430c8",
	}
)

func init() {
	namespaces[NameSpaceX500] = "6ba7b814-9dad-11d1-80b4-00c04fd430c8"
	namespaces[NameSpaceOID] = "6ba7b812-9dad-11d1-80b4-00c04fd430c8"
	namespaces[NameSpaceURL] = "6ba7b811-9dad-11d1-80b4-00c04fd430c8"
	namespaces[NameSpaceDNS] = "6ba7b810-9dad-11d1-80b4-00c04fd430c8"
	generator.init()
}

func TestEqual(t *testing.T) {
	for k, v := range namespaces {
		u, _ := Parse(v)
		assert.True(t, Equal(k, u), "Id's should be equal")
		assert.Equal(t, k.String(), u.String(), "Stringer versions should equal")
	}
}

func TestCompare(t *testing.T) {
	assert.True(t, Compare(NameSpaceDNS, NameSpaceDNS) == 0, "SDNS should be equal to DNS")
	assert.True(t, Compare(NameSpaceDNS, NameSpaceURL) == -1, "DNS should be less than URL")
	assert.True(t, Compare(NameSpaceURL, NameSpaceDNS) == 1, "URL should be greater than DNS")

	assert.True(t, Compare(nil, NameSpaceDNS) == -1, "Nil should be less than DNS")
	assert.True(t, Compare(NameSpaceDNS, nil) == 1, "DNS should be greater than Nil")
	assert.True(t, Compare(nil, nil) == 0, "nil should equal to nil")

	assert.True(t, Compare(Nil, NameSpaceDNS) == -1, "Nil should be less than DNS")
	assert.True(t, Compare(NameSpaceDNS, Nil) == 1, "DNS should be greater than Nil")
	assert.True(t, Compare(Nil, Nil) == 0, "Nil should equal to Nil")

	b1 := UUID([16]byte{
		0x01, 0x09, 0x09, 0x00,
		0xff, 0x02,
		0xff, 0x03,
		0x00,
		0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00})

	b2 := UUID([16]byte{
		0x01, 0x09, 0x09, 0x00,
		0xff, 0x02,
		0xff, 0x03,
		0x00,
		0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00})

	assert.Equal(t, 0, Compare(b1, b2), "Should equal")

	binary.BigEndian.PutUint32(b1[:4], 16779999)
	binary.BigEndian.PutUint32(b2[:4], 16780000)
	assert.Equal(t, -1, Compare(b1, b2), "Should be less")

	binary.BigEndian.PutUint32(b1[:4], 16780000)
	binary.BigEndian.PutUint32(b2[:4], 16779999)
	assert.Equal(t, 1, Compare(b1, b2), "Should be greater")

	binary.BigEndian.PutUint32(b2[:4], 16780000)
	assert.Equal(t, 0, Compare(b1, b2), "Should equal")

	binary.BigEndian.PutUint16(b1[4:6], 25000)
	binary.BigEndian.PutUint16(b2[4:6], 25001)
	assert.Equal(t, -1, Compare(b1, b2), "Should be less")

	binary.BigEndian.PutUint16(b1[4:6], 25001)
	binary.BigEndian.PutUint16(b2[4:6], 25000)
	assert.Equal(t, 1, Compare(b1, b2), "Should be greater")

	binary.BigEndian.PutUint16(b2[4:6], 25001)
	assert.Equal(t, 0, Compare(b1, b2), "Should equal")

	binary.BigEndian.PutUint16(b1[6:8], 25000)
	binary.BigEndian.PutUint16(b2[6:8], 25001)
	assert.Equal(t, -1, Compare(b1, b2), "Should be less")

	binary.BigEndian.PutUint16(b1[6:8], 25001)
	binary.BigEndian.PutUint16(b2[6:8], 25000)
	assert.Equal(t, 1, Compare(b1, b2), "Should be greater")

	binary.BigEndian.PutUint16(b2[6:8], 25001)
	assert.Equal(t, 0, Compare(b1, b2), "Should equal")

	b2[8] = 1
	assert.Equal(t, -1, Compare(b1, b2), "Should be less")

	b1[8] = 3
	assert.Equal(t, 1, Compare(b1, b2), "Should be greater")

}

func TestNewHex(t *testing.T) {
	s := "e902893a9d223c7ea7b8d6e313b71d9f"
	u := NewHex(s)
	assert.Equal(t, VersionThree, u.Version(), "Expected correct version")
	assert.Equal(t, VariantRFC4122, u.Variant(), "Expected correct variant")
	assert.True(t, parseUUIDRegex.MatchString(u.String()), "Expected string representation to be valid")

	assert.True(t, didNewHexPanic(), "Hex string should panic when invalid")
}

func didNewHexPanic() bool {
	return func() (didPanic bool) {
		defer func() {
			if recover() != nil {
				didPanic = true
			}
		}()

		NewHex("*********-------)()()()()(")
		return
	}()
}

func TestParse(t *testing.T) {
	for _, v := range invalidHexStrings {
		_, err := Parse(v)
		assert.Error(t, err, "Expected error due to invalid UUID string")
	}
	for _, v := range validHexStrings {
		_, err := Parse(v)
		assert.NoError(t, err, "Expected valid UUID string but got error")
	}
	for _, v := range namespaces {
		_, err := Parse(v)
		assert.NoError(t, err, "Expected valid UUID string but got error")
	}
}

func TestNew(t *testing.T) {
	for k := range namespaces {

		u := New(k.Bytes())

		assert.NotNil(t, u, "Expected a valid non nil UUID")
		assert.Equal(t, VersionOne, u.Version(), "Expected correct version %d, but got %d", VersionOne, u.Version())
		assert.Equal(t, VariantRFC4122, u.Variant(), "Expected ReservedNCS variant %x, but got %x", VariantNCS, u.Variant())
		assert.Equal(t, k.String(), u.String(), "Stringer versions should equal")
	}
}

func TestNew_Bulk(t *testing.T) {
	for i := 0; i < 1000000; i++ {
		New(uuidBytes[:])
	}
}

func TestNewHex_Bulk(t *testing.T) {
	for i := 0; i < 1000000; i++ {
		s := "f3593cffee9240df408687825b523f13"
		NewHex(s)
	}
}

func TestNewHash(t *testing.T) {
	ids := make([]UUID, 5)
	for i, v := range []hash.Hash{
		md5.New(), sha1.New(), sha512.New(), sha256.New(),
	} {
		var id string
		id = "this is a unique id"
		ids[i] = NewHash(v, NameSpaceDNS, goLang, NameSpaceDNS.Bytes(), &id)
		assert.False(t, IsNil(ids[i]))
		assert.Equal(t, VersionUnknown, ids[i].Version(), "Expected correct version")
		assert.Equal(t, VariantFuture, ids[i].Variant(), "Expected correct variant")
	}

	assert.True(t, didNewHashPanic())
}

func didNewHashPanic() bool {
	return func() (didPanic bool) {
		defer func() {
			if recover() != nil {
				didPanic = true
			}
		}()
		NewHash(md5.New(), NameSpaceDNS, 0)
		return
	}()
}

func TestNewV1(t *testing.T) {
	id := NewV1()
	assert.Equal(t, VersionOne, id.Version(), "Expected correct version")
	assert.Equal(t, VariantRFC4122, id.Variant(), "Expected correct variant")
	assert.True(t, parseUUIDRegex.MatchString(id.String()), "Expected string representation to be valid")
}

func TestNewV2(t *testing.T) {
	id := NewV2(SystemIdGroup)
	assert.Equal(t, VersionTwo, id.Version(), "Expected correct version")
	assert.Equal(t, VariantRFC4122, id.Variant(), "Expected correct variant")
	assert.True(t, parseUUIDRegex.MatchString(id.String()), "Expected string representation to be valid")
}

func TestNewV3(t *testing.T) {
	id := NewV3(NameSpaceURL, goLang)
	assert.Equal(t, VersionThree, id.Version(), "Expected correct version")
	assert.Equal(t, VariantRFC4122, id.Variant(), "Expected correct variant")
	assert.True(t, parseUUIDRegex.MatchString(id.String()), "Expected string representation to be valid")

	ur, _ := url.Parse(string(goLang))

	// Same NS same name MUST be equal
	id2 := NewV3(NameSpaceURL, ur)
	assert.Equal(t, id, id2, "Expected UUIDs generated with same namespace and name to equal")

	// Different NS same name MUST NOT be equal
	id3 := NewV3(NameSpaceDNS, ur)
	assert.NotEqual(t, id, id3, "Expected UUIDs generated with different namespace and same name to be different")

	// Same NS different name MUST NOT be equal
	id4 := NewV3(NameSpaceURL, id)
	assert.NotEqual(t, id, id4, "Expected UUIDs generated with the same namespace and different names to be different")

	ids := []Implementation{
		id, id2, id3, id4,
	}

	for j, id := range ids {
		i := NewV3(NameSpaceURL, string(j), id)
		assert.NotEqual(t, id, i, "Expected UUIDs generated with the same namespace and different names to be different")
	}

	id = NewV3(NameSpaceDNS, "www.example.com")
	assert.Equal(t, "5df41881-3aed-3515-88a7-2f4a814cf09e", id.String())

	id = NewV3(NameSpaceDNS, "python.org")
	assert.Equal(t, "6fa459ea-ee8a-3ca4-894e-db77e160355e", id.String())
}

func TestNewV4(t *testing.T) {
	id := NewV4()
	assert.Equal(t, VersionFour, id.Version(), "Expected correct version")
	assert.Equal(t, VariantRFC4122, id.Variant(), "Expected correct variant")
	assert.True(t, parseUUIDRegex.MatchString(id.String()), "Expected string representation to be valid")
}

func TestNewV5(t *testing.T) {
	u := NewV5(NameSpaceURL, goLang)

	assert.Equal(t, VersionFive, u.Version(), "Expected correct version")
	assert.Equal(t, VariantRFC4122, u.Variant(), "Expected correct variant")
	assert.True(t, parseUUIDRegex.MatchString(u.String()), "Expected string representation to be valid")

	ur, _ := url.Parse(string(goLang))

	// Same NS same name MUST be equal
	u2 := NewV5(NameSpaceURL, ur)
	assert.Equal(t, u, u2, "Expected UUIDs generated with same namespace and name to equal")

	// Different NS same name MUST NOT be equal
	u3 := NewV5(NameSpaceDNS, ur)
	assert.NotEqual(t, u, u3, "Expected UUIDs generated with different namespace and same name to be different")

	// Same NS different name MUST NOT be equal
	u4 := NewV5(NameSpaceURL, u)
	assert.NotEqual(t, u, u4, "Expected UUIDs generated with the same namespace and different names to be different")

	ids := []Implementation{
		u, u2, u3, u4,
	}

	for j, id := range ids {
		i := NewV5(NameSpaceURL, string(j), id)
		assert.NotEqual(t, i, id, "Expected UUIDs generated with the same namespace and different names to be different")
	}

	u = NewV5(NameSpaceDNS, "python.org")
	assert.Equal(t, "886313e1-3b8a-5372-9b90-0c9aee199e5d", u.String())
}

func TestBulkV1(t *testing.T) {
	eachIsUnique(t, BulkV1(100))
}

func TestBulkV4(t *testing.T) {
	eachIsUnique(t, BulkV4(100))
}

func Test_EachIsUnique(t *testing.T) {

	// Run half way through to avoid running within default resolution only
	BulkV1(int(defaultSpinResolution / 2))

	spin := int(defaultSpinResolution)

	// Test V1
	eachIsUnique(t, BulkV1(spin))

	// Test V2
	if runtime.GOOS != "windows" {
		id := NewV2(SystemIdUser)
		id2 := NewV2(SystemIdGroup)
		assert.NotEqual(t, id, id2)
	}

	// Test V4
	ids := BulkV4(spin)

	eachIsUnique(t, ids)

	// Test V3
	eachHashableIsUnique(t, spin, func(i int) UUID { return NewV3(NameSpaceDNS, string(i)) })

	// Test V5
	eachHashableIsUnique(t, spin, func(i int) UUID { return NewV5(NameSpaceDNS, string(i)) })
}

func eachIsUnique(t *testing.T, ids []UUID) {
	for i, v := range ids {
		for j := 0; j < len(ids); j++ {
			if i == j {
				continue
			}
			if b := assert.NotEqual(t, v.String(), ids[j].String(), "Should not create the same UUID"); !b {
				break
			}
		}
	}
}

func eachHashableIsUnique(t *testing.T, spin int, id func(int) UUID) {
	ids := make([]UUID, spin)
	for i := range ids {
		ids[i] = id(i)
	}
	eachIsUnique(t, ids)
}

func Test_NameSpaceUUIDs(t *testing.T) {
	for k, v := range namespaces {
		arrayId, _ := Parse(v)
		uuidId := UUID{}
		uuidId.unmarshal(arrayId.Bytes())
		assert.Equal(t, v, arrayId.String())
		assert.Equal(t, v, k.String())
	}
}

func TestIsNil(t *testing.T) {
	assert.True(t, IsNil(nil))
	assert.True(t, IsNil(Nil))
	assert.True(t, IsNil(UUID{}))
	assert.False(t, IsNil(NameSpaceDNS))
	assert.False(t, IsNil(NewV1()))
	assert.False(t, IsNil(NewV2(SystemIdGroup)))
	assert.False(t, IsNil(NewV3(NameSpaceDNS, "www.example.com")))
	assert.False(t, IsNil(NewV4()))
	assert.False(t, IsNil(NewV5(NameSpaceDNS, "www.example.com")))
}

func TestReadV1(t *testing.T) {
	ids := make([]UUID, 100)
	ReadV1(ids)
	eachIsUnique(t, ids)
}

func TestReadV4(t *testing.T) {
	ids := make([]UUID, 100)
	ReadV4(ids)
	eachIsUnique(t, ids)
}
