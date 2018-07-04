package main

import (
	"errors"
	"flag"
	"fmt"
	"github.com/cdevr/WapSNMP"
	"time"
)

var target = flag.String("target", "", "The host to connect to")
var community = flag.String("community", "", "The community to use")
var timeout = flag.Duration("timeout", 2*time.Second, "timeout for packets")
var retries = flag.Int("retries", 5, "how many times to retry sending a packet before giving up")

var oidasstring = ".1.3.6.1.4.1.2636.3.2.5.1.30"
var oid = wapSnmp.MustParseOid(oidasstring)

func decodeOidToLSPName(lspOid wapSnmp.Oid) (*string, error) {
	if !lspOid.Within(oid) {
		return nil, errors.New("Oid must be within RRO table")
	}
	result := ""
	for _, i := range lspOid[len(oid):] {
		if i == 0 {
			break
		}
		result += fmt.Sprintf("%c", i)
	}
	return &result, nil
}

func doGetRROs() {
	flag.Parse()

	fmt.Printf("target=%v\ncommunity=%v\noid=%v\n", *target, *community, oidasstring)
	version := wapSnmp.SNMPv2c

	fmt.Printf("Contacting %v %v %v\n", *target, *community, version)
	wsnmp, err := wapSnmp.NewWapSNMP(*target, *community, version, *timeout, *retries)
	if err != nil {
		fmt.Printf("Error creating wsnmp => %v\n", err)
		return
	}
	defer wsnmp.Close()

	table, err := wsnmp.GetTable(oid)
	if err != nil {
		fmt.Printf("Error getting table => %v\n", err)
		return
	}
	for k, v := range table {
		decoded, err := decodeOidToLSPName(wapSnmp.MustParseOid(k))
		if err != nil {
			fmt.Printf("Faulty oid returned : %v", k)
			return
		}
		fmt.Printf("%v => '%v'\n", *decoded, v)
	}
}

func main() {
	doGetRROs()
}
