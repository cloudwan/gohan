package wapsnmp

import (
	"fmt"
	"testing"
)

func TestParseOid(t *testing.T) {
	tests := []struct {
		toParse           string
		expectedCanonForm string
		expectFail        bool
	}{
		{"1.3.6.1.4.1.2636.3.2.3.1.20", ".1.3.6.1.4.1.2636.3.2.3.1.20", false},
		{".1.3.127.128", ".1.3.127.128", false},
		{"1.3", ".1.3", false},
		{".1.3.127.128.129", ".1.3.127.128.129", false},
		{"", ".", false},
		{".", ".", false},
		{"Donald Duck", "", true},
	}

	for _, test := range tests {
		oid, err := ParseOid(test.toParse)
		if (err != nil) != test.expectFail {
			t.Errorf("ParseOid '%s' got error '%s', expected '%t'", test.toParse, err, test.expectFail)
		}
		if !test.expectFail {
			if fmt.Sprintf("%s", oid) != test.expectedCanonForm {
				t.Errorf("ParseOid '%s' got '%s', expected '%s'", test.toParse, oid, test.expectedCanonForm)
			}
		}
	}
}

func TestOidEncode(t *testing.T) {
	tests := map[string][]byte{
		"1.3.6.1.4.1.2636.3.2.3.1.20": {0x2b, 0x06, 0x01, 0x04, 0x01, 0x94, 0x4c, 0x03, 0x02, 0x03, 0x01, 0x14},
		"1.3.6.1.2.1.1.5.0":           {0x2b, 0x06, 0x01, 0x02, 0x01, 0x01, 0x05, 0x00},
	}

	for oidString, expected := range tests {
		oid, err := ParseOid(oidString)
		if err != nil {
			t.Errorf("ParseOid '%s' error '%s'", oidString, err)
		}
		encode, err := oid.Encode()

		equal := len(encode) == len(expected)
		if equal {
			for idx, val := range encode {
				if val != expected[idx] {
					equal = false
				}
			}
		}
		if !equal {
			t.Errorf("ParseOid '%s' expected '%v', got '%v'\n", oidString, expected, encode)
		}
	}
}

func TestOidDecode(t *testing.T) {
	encodedOid := []byte{0x2b, 0x06, 0x01, 0x04, 0x01, 0x94, 0x4c, 0x03, 0x02, 0x03, 0x01, 0x14}
	oid, err := DecodeOid(encodedOid)
	if err != nil {
		t.Errorf("DecodeOid '1.3.6.1.4.1.2636.3.2.3.1.20' error '%s'", err)
	}
	if fmt.Sprintf("%s", oid) != ".1.3.6.1.4.1.2636.3.2.3.1.20" {
		t.Errorf("DecodeOid expected '1.3.6.1.4.1.2636.3.2.3.1.20', got '%s'", fmt.Sprintf("%s", oid))
	}
}

func TestWithin(t *testing.T) {
	if !MustParseOid("1.2.3").Within(MustParseOid("1.2")) {
		t.Errorf("Within is not working")
	}
}
