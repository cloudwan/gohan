package wapsnmp

import (
	"fmt"
	"math/rand" // Needed to set Seed, so a consistent request ID will be chosen.
	"testing"
	"time"
)

func ExampleGetTable() {
	target := "some_host"
	community := "public"
	version := SNMPv2c

	oid := MustParseOid(".1.3.6.1.4.1.2636.3.2.3.1.20")

	fmt.Printf("Contacting %v %v %v\n", target, community, version)
	wsnmp, err := NewWapSNMP(target, community, version, 2*time.Second, 5)
	defer wsnmp.Close()
	if err != nil {
		fmt.Printf("Error creating wsnmp => %v\n", wsnmp)
		return
	}

	table, err := wsnmp.GetTable(oid)
	if err != nil {
		fmt.Printf("Error getting table => %v\n", wsnmp)
		return
	}
	for k, v := range table {
		fmt.Printf("%v => %v\n", k, v)
	}
}

func ExampleGetBulk() {
	target := "some_host"
	community := "public"
	version := SNMPv2c

	oid := MustParseOid(".1.3.6.1.2.1")

	fmt.Printf("Contacting %v %v %v\n", target, community, version)
	wsnmp, err := NewWapSNMP(target, community, version, 2*time.Second, 5)
	defer wsnmp.Close()
	if err != nil {
		fmt.Printf("Error creating wsnmp => %v\n", wsnmp)
		return
	}
	defer wsnmp.Close()
	for {
		results, err := wsnmp.GetBulk(oid, 50)
		if err != nil {
			fmt.Printf("GetBulk error => %v\n", err)
			return
		}
		for o, v := range results {
			fmt.Printf("%v => %v\n", o, v)

			oid = MustParseOid(o)
		}
		/*  Old version without GETBULK
		    result_oid, val, err := wsnmp.GetNext(oid)
		    if err != nil {
		      fmt.Printf("GetNext error => %v\n", err)
		      return
		    }
		    fmt.Printf("GetNext(%v, %v, %v, %v) => %s, %v\n", target, community, version, oid, result_oid, val)
		    oid = *result_oid
		*/
	}
}

func ExampleGet() {
	target := "some_host"
	community := "public"
	version := SNMPv2c

	oids := []Oid{
		MustParseOid(".1.3.6.1.2.1.1.1.0"),
		MustParseOid(".1.3.6.1.2.1.1.2.0"),
		MustParseOid(".1.3.6.1.2.1.2.1.0"),
	}

	wsnmp, err := NewWapSNMP(target, community, version, 2*time.Second, 5)
	defer wsnmp.Close()
	if err != nil {
		fmt.Printf("Error creating wsnmp => %v\n", wsnmp)
		return
	}

	for _, oid := range oids {
		val, err := wsnmp.Get(oid)
		fmt.Printf("Getting %v\n", oid)
		if err != nil {
			fmt.Printf("Get error => %v\n", err)
			return
		}
		fmt.Printf("Get(%v, %v, %v, %v) => %v\n", target, community, version, oid, val)
	}
}

func TestGet(t *testing.T) {
	rand.Seed(0)

	target := "magic_host"
	community := "[R0_C@cti!]"
	version := SNMPv2c

	oid := MustParseOid("1.3.6.1.2.1.1.3.0")

	udpStub := NewUdpStub(t)
	defer udpStub.CheckClosed()
	// Expect a UDP SNMP GET packet.
	udpStub.Expect("302e020101040b5b52305f4340637469215da01c020478fc2ffa020100020100300e300c06082b060102010103000500").AndRespond([]string{"3032020101040b5b52305f4340637469215da220020421182cd70201000201003012301006082b06010201010300430404926fa4"})

	wsnmp := NewWapSNMPOnConn(target, community, version, 2*time.Second, 5, udpStub)
	//wsnmp, err := NewWapSNMP(target, community, version, 2*time.Second, 5)
	defer wsnmp.Close()
	val, err := wsnmp.Get(oid)

	if err != nil {
		t.Errorf("Error testing to get a value: %v.", err)
	}

	if val != time.Duration(76705700)*10*time.Millisecond {
		t.Errorf("Received wrong value: %v", val)
	}
}

func TestGetTable(t *testing.T) {
	rand.Seed(0)

	target := "magic_host"
	community := "examplcommunity"
	version := SNMPv2c

	oid := MustParseOid("1.3.6.1.2.1.2.2.1.21")

	udpStub := NewUdpStub(t)
	defer udpStub.CheckClosed()
	// Expect a UDP SNMP GETBULK packet.
	udpStub.Expect("3033020101040f6578616d706c636f6d6d756e697479a51d020478fc2ffa020100020132300f300d06092b06010201020201150500").AndRespond([]string{"3082039c020101040f6578616d706c636f6d6d756e697479a282038402046a4eef7a020100020100308203743010060b2b060102010202011585064201003010060b2b060102010202011585074201003010060b2b060102010202011585084201003010060b2b060102010202011585094201003010060b2b0601020102020115850a4201003010060b2b0601020102020115850b4201003010060b2b0601020102020115850c4201003010060b2b0601020102020115850d4201003010060b2b0601020102020115850e4201003010060b2b0601020102020115850f4201003010060b2b060102010202011585104201003010060b2b060102010202011585114201003010060b2b060102010202011585124201003010060b2b060102010202011585134201003010060b2b060102010202011585144201003010060b2b060102010202011585154201003010060b2b060102010202011585164201003010060b2b060102010202011585174201003010060b2b060102010202011585184201003010060b2b060102010202011585194201003010060b2b0601020102020115851a4201003010060b2b0601020102020115851b4201003010060b2b0601020102020115851c4201003010060b2b0601020102020115851d4201003010060b2b0601020102020115851e4201003010060b2b0601020102020115851f4201003010060b2b060102010202011585204201003010060b2b060102010202011585214201003010060b2b060102010202011585224201003010060b2b060102010202011585234201003010060b2b060102010202011585244201003010060b2b060102010202011585254201003010060b2b060102010202011585264201003010060b2b06010201020201158527420100300f060a2b060102010202011601060100300f060a2b060102010202011604060100300f060a2b060102010202011605060100300f060a2b060102010202011606060100300f060a2b060102010202011607060100300f060a2b060102010202011608060100300f060a2b060102010202011609060100300f060a2b06010201020201160a060100300f060a2b06010201020201160b060100300f060a2b06010201020201160c060100300f060a2b06010201020201160d060100300f060a2b060102010202011610060100300f060a2b060102010202011611060100300f060a2b060102010202011612060100300f060a2b060102010202011615060100300f060a2b060102010202011616060100"})

	wsnmp := NewWapSNMPOnConn(target, community, version, 2*time.Second, 5, udpStub)
	defer wsnmp.Close()

	val, err := wsnmp.GetTable(oid)

	if err != nil {
		t.Fatalf("Error testing to get a table: %v.", err)
	}

	gValue, ok := val[".1.3.6.1.2.1.2.2.1.21.646"].(Gauge)
	if !ok {
		v := val[".1.3.6.1.2.1.2.2.1.21.646"]
		t.Fatalf("Expected a zero int value for this table request, got wrong type %T for value '%v'", v, v)
	}
	if gValue != 0 {
		t.Errorf("Expected a zero value in this table request, got %v", val["1.3.6.1.2.1.2.2.1.21.646"])
	}
}
