WapSnmp : SNMP client for golang
--------------------------------

This is an open-source SNMP client library for Go. This allows you to query SNMP servers for any variable, given it's OID (no MIB resolution). It is released under the Apache 2.0 licence.

This library has been written to be in Go style and that means it should be very resistent to all error conditions. It's entirely non-blocking/asynchronous and very, very fast. It's also surprisingly small and easy to understand. Excellent test coverage is provided.

It supports the following SNMP operations:

* Get
* GetMultiple
* Set
* SetMultiple
* GetNext
* GetBulk
* GetTable (use getBulk to get an entire subtree)

All of these are implemented with timeout support, and correct error/retry handling.

It supports SNMPv2c or lower (not 3, due to it's complexity), and supports all methods provided as part of that standard. Get, GetMultiple (which are really the same request, but ...), GetNext and GetBulk.

It has been tested on juniper and cisco devices and has proven to remain stable over long periods of time.

Example usage of the library:

    func DoGetTableTest(target string) {
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

This library can also be used as a ASN1 BER parser.

The library has native support for a whole lot of SNMP types :

* Boolean
* Integer
* OctetString (string)
* Oids
* Null
* Counter32
* Counter64
* Gauge32
* TimeTicks
* EndOfMibView

And this should be easy to expand.
