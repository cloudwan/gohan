function validateIpAddress(ipAddress, ipVersion) {
    var IPv4Seg  = "(25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9])"
    var IPv4Addr = "(" + IPv4Seg + "\.){3,3}" + IPv4Seg
    var IPv6Seg  = "[0-9a-fA-F]{1,4}"
    var IPv6Addr = "^(" +
               "(" + IPv6Seg + ":){7,7}" + IPv6Seg + "|" +          // 1:2:3:4:5:6:7:8
               "(" + IPv6Seg + ":){1,7}:|" +                        // 1::                                 1:2:3:4:5:6:7::
               "(" + IPv6Seg + ":){1,6}:" + IPv6Seg + "|" +         // 1::8               1:2:3:4:5:6::8   1:2:3:4:5:6::8
               "(" + IPv6Seg + ":){1,5}(:" + IPv6Seg + "){1,2}|" +  // 1::7:8             1:2:3:4:5::7:8   1:2:3:4:5::8
               "(" + IPv6Seg + ":){1,4}(:" + IPv6Seg + "){1,3}|" +  // 1::6:7:8           1:2:3:4::6:7:8   1:2:3:4::8
               "(" + IPv6Seg + ":){1,3}(:" + IPv6Seg + "){1,4}|" +  // 1::5:6:7:8         1:2:3::5:6:7:8   1:2:3::8
               "(" + IPv6Seg + ":){1,2}(:" + IPv6Seg + "){1,5}|" +  // 1::4:5:6:7:8       1:2::4:5:6:7:8   1:2::8
               "" + IPv6Seg + ":((:" + IPv6Seg + "){1,6})|" +       // 1::3:4:5:6:7:8     1::3:4:5:6:7:8   1::8
               ":((:" + IPv6Seg + "){1,7}|:)|" +                    // ::2:3:4:5:6:7:8    ::2:3:4:5:6:7:8  ::8       ::
               "fe80:(:" + IPv6Seg + "){0,4}%[0-9a-zA-Z]{1,}|" +    // fe80::7:8%eth0     fe80::7:8%1  (link-local IPv6 addresses with zone index)
               "::(ffff(:0{1,4}){0,1}:){0,1}" + IPv4Addr + "|" +    // ::255.255.255.255  ::ffff:255.255.255.255  ::ffff:0:255.255.255.255 (IPv4-mapped IPv6 addresses and IPv4-translated addresses)
               "(" + IPv6Seg + ":){1,4}:" + IPv4Addr +              // 2001:db8:3:4::192.0.2.33  64:ff9b::192.0.2.33 (IPv4-Embedded IPv6 Address)
               ")$"

    console.log("Playing with IP " + ipAddress + " which is of type " + typeof ipAddress);
    var re;
    if (!ipAddress) {
        return;
    }
    switch(ipVersion) {
        case 4:
            if(ipAddress.match("^" + IPv4Addr + "([01]?\d|2\d|3[12])?$") === null) {
                throw new ValidationException("'" + ipAddress + "' is not a valid IPv4 address.");
            }
            break;
        case 6:
            if(ipAddress.match(IPv6Addr) === null) {
                throw new ValidationException("'" + ipAddress + "' is not a valid IPv6 address.");
            }
            break;
        default:
            throw new ValidationException("'" + ipVersion + "' is not a valid IP address version.");
    }
}
