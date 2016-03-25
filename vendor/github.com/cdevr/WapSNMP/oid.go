package wapsnmp

/* Encode decode OIDs.

   References : http://rane.com/note161.html
*/

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// Oid is the SNMP object identifier type.
type Oid []int

// String returns the string representation for this oid object.
func (o Oid) String() string {
	/* A zero-length Oid has to be valid as it's often used as the start of a
	   Walk. */
	if len(o) == 0 {
		return "."
	}
	var result string
	for _, val := range o {
		result += fmt.Sprintf(".%d", val)
	}
	return result
}

// MustParseOid parses a string oid to an Oid instance. Panics on error.
func MustParseOid(o string) Oid {
	result, err := ParseOid(o)
	if err != nil {
		panic(err)
	}
	return result
}

// ParseOid a text format oid into an Oid instance.
func ParseOid(oid string) (Oid, error) {
	// Special case "." = [], "" = []
	if oid == "." || oid == "" {
		return Oid{}, nil
	}
	if oid[0] == '.' {
		oid = oid[1:]
	}
	oidParts := strings.Split(oid, ".")
	res := make([]int, len(oidParts))
	for idx, val := range oidParts {
		parsedVal, err := strconv.Atoi(val)
		if err != nil {
			return nil, err
		}
		res[idx] = parsedVal
	}
	result := Oid(res)

	return result, nil
}

// DecodeOid decodes a ASN.1 BER raw oid into an Oid instance.
func DecodeOid(raw []byte) (*Oid, error) {
	if len(raw) < 1 {
		return nil, errors.New("0 byte oid doesn't exist")
	}

	result := make([]int, 2)
	result[0] = int(raw[0] / 40)
	result[1] = int(raw[0] % 40)
	val := 0
	for idx, b := range raw {
		if idx == 0 {
			continue
		}
		if b < 128 {
			val = val*128 + int(b)
			result = append(result, val)
			val = 0
		} else {
			val = val*128 + int(b%128)
		}
	}
	r := Oid(result)
	return &r, nil
}

// Encode encodes the oid into an ASN.1 BER byte array.
func (o Oid) Encode() ([]byte, error) {
	if len(o) < 2 {
		return nil, errors.New("oid needs to be at least 2 long")
	}
	var result []byte
	/* Every o is supposed to start with 40 * first_byte + second
	   byte */
	start := (40 * o[0]) + o[1]
	result = append(result, byte(start))
	for i := 2; i < len(o); i++ {
		val := o[i]

		var toadd []int
		if val == 0 {
			toadd = append(toadd, 0)
		}
		for val > 0 {
			toadd = append(toadd, val%128)
			val /= 128
		}

		for i := len(toadd) - 1; i >= 0; i-- {
			sevenbits := toadd[i]
			if i != 0 {
				result = append(result, 128+byte(sevenbits))
			} else {
				result = append(result, byte(sevenbits))
			}
		}
	}
	return result, nil
}

// Copy copies an oid into a new object instance.
func (o Oid) Copy() Oid {
	dest := make([]int, len(o))
	copy(dest, o)
	return Oid(dest)
}

// Within determines if an oid has this oid instance as a prefix.
//
// E.g. MustParseOid("1.2.3").Within(MustParseOid("1.2")) => true.
func (o Oid) Within(other Oid) bool {
	if len(other) > len(o) {
		return false
	}
	for idx, val := range other {
		if o[idx] != val {
			return false
		}
	}
	return true
}
