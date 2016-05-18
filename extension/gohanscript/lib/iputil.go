package lib

import (
	"math/big"
	"net"
	"strings"
)

//IPToInt converts ip string to int
func IPToInt(ip string) int {
	i := big.NewInt(0)
	i.SetBytes(net.ParseIP(ip).To4())
	return int(i.Int64())
}

//IntToIP converts int to ip string
func IntToIP(value int) string {
	i := big.NewInt(0)
	i.SetInt64(int64(value))
	ip := net.IP(i.Bytes())
	return ip.String()
}

//IPAdd adds int for ip
func IPAdd(ip string, value int) string {
	i := big.NewInt(0)
	if strings.Contains(ip, ".") {
		i.SetBytes(net.ParseIP(ip).To4())
	}else{
		i.SetBytes(net.ParseIP(ip).To16())
	}
	j := big.NewInt(int64(value))
	i.Add(i, j)
	ipObj := net.IP(i.Bytes())
	return ipObj.String()
}

//ParseCidr parse cidr for start ip and number of ips
func ParseCidr(cidr string) (string, int, int) {
	_, n, err := net.ParseCIDR(cidr)
	if err != nil {
		return "", -1, -1
	}
	ones, bits := n.Mask.Size()
	return n.IP.String(), ones, bits
}

//FloatToInt converts float to int
func FloatToInt(value float64) int {
	return int(value)
}
