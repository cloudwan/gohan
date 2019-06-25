// Copyright (C) 2015 NTT Innovation Institute, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or
// implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package schema

import (
	"fmt"
	"net"
	"regexp"
	"strconv"
	"strings"

	"github.com/Masterminds/semver"
	"github.com/twinj/uuid"
	"github.com/xeipuuv/gojsonschema"
)

type macFormatChecker struct{}
type ipv4FormatChecker struct{}
type cidrFormatChecker struct{}
type cidrOrIPv4FormatChecker struct{}
type ipv4NetworkFormatChecker struct{}
type ipv4AddressWithCidrFormatChecker struct{}
type regexFormatChecker struct{}
type uuidFormatChecker struct{}
type hyphenatedUUIDFormatChecker struct{}
type nonHyphenatedUUIDFormatChecker struct{}
type portFormatChecker struct{}
type yamlFormatChecker struct{}
type textFormatChecker struct{}
type versionFormatChecker struct{}
type versionConstraintFormatChecker struct{}

var yangIPv4FormatRegexp = regexp.MustCompile(`^(([0-9]|[1-9][0-9]|1[0-9][0-9]|2[0-4][0-9]|25[0-5])\.){3}([0-9]|[1-9][0-9]|1[0-9][0-9]|2[0-4][0-9]|25[0-5])(%[\p{N}\p{L}]+)?$`)

func (f macFormatChecker) IsFormat(input string) bool {
	match, _ := regexp.MatchString(`^([0-9A-Fa-f]{2}:){5}[0-9A-Fa-f]{2}$`, input)
	return match
}

func matchYangIPv4AddressPattern(input string) bool {
	return yangIPv4FormatRegexp.MatchString(input)
}

func isValidIP(ip net.IP) bool {
	return ip != nil
}

func isV4(ip net.IP) bool {
	return ip.To4() != nil
}

func (f ipv4FormatChecker) IsFormat(input string) bool {
	ip := net.ParseIP(input)
	if !isValidIP(ip) || !isV4(ip) {
		return false
	}
	return matchYangIPv4AddressPattern(input)
}

func (f cidrFormatChecker) IsFormat(input string) bool {
	ip, _, err := net.ParseCIDR(input)
	if err != nil {
		return false
	}
	if isV4(ip) {
		inputIp, mask := extractIpAndMaskFromCidrFormattedString(input)
		if !maskIsValid(mask) {
			return false
		}
		return matchYangIPv4AddressPattern(inputIp)
	}
	return true
}

func (f cidrOrIPv4FormatChecker) IsFormat(input string) bool {
	ipv4 := ipv4FormatChecker{}
	cidr := ipv4AddressWithCidrFormatChecker{}
	net := ipv4NetworkFormatChecker{}
	return ipv4.IsFormat(input) || cidr.IsFormat(input) || net.IsFormat(input)
}

func (f ipv4NetworkFormatChecker) IsFormat(input string) bool {
	hostIP, netIP, _, cidrErr := extractHostAndNet(input)
	if cidrErr != nil {
		return false
	}

	inputIp, mask := extractIpAndMaskFromCidrFormattedString(input)
	if !matchYangIPv4AddressPattern(inputIp) {
		return false
	}
	if !maskIsValid(mask) {
		return false
	}
	return hostIP.Equal(netIP)
}

func maskIsValid(mask string) bool {
	if len(mask) > 2 {
		return false
	}
	if len(mask) == 2 && mask[0] == '0' {
		return false
	}
	return true
}

func (f ipv4AddressWithCidrFormatChecker) IsFormat(input string) bool {
	hostIP, netIP, mask, cidrErr := extractHostAndNet(input)
	if cidrErr != nil {
		return false
	}

	if isBroadcast(hostIP, mask) {
		return false
	}

	inputIp := extractIpFromCidrFormattedString(input)
	if !matchYangIPv4AddressPattern(inputIp) {
		return false
	}
	return !hostIP.Equal(netIP)
}

func extractIpFromCidrFormattedString(input string) string {
	ip, _ := extractIpAndMaskFromCidrFormattedString(input)
	return ip
}

func extractIpAndMaskFromCidrFormattedString(input string) (string, string) {
	ipAndMask := strings.Split(input, "/")
	return ipAndMask[0], ipAndMask[1]
}

func extractHostAndNet(input string) (hostIP net.IP, netIP net.IP, mask net.IPMask, err error) {
	cidrIP, cidrNet, cidrErr := net.ParseCIDR(input)
	if cidrErr != nil {
		return nil, nil, nil, cidrErr
	}
	hostIP = cidrIP.To4()
	netIP = cidrNet.IP.To4()
	mask = cidrNet.Mask
	if hostIP == nil || netIP == nil {
		return nil, nil, nil, fmt.Errorf("Invalid address: host or network ip empty")
	}
	return
}

func isBroadcast(host net.IP, mask net.IPMask) bool {
	for i := range host {
		if (host[i] | mask[i]) != 255 {
			return false
		}
	}
	return true
}

func (f regexFormatChecker) IsFormat(input string) bool {
	_, err := regexp.Compile(input)
	return err == nil
}

func (f uuidFormatChecker) IsFormat(input string) bool {
	_, err := uuid.Parse(input)
	return err == nil
}

func (f hyphenatedUUIDFormatChecker) IsFormat(input string) bool {
	match, _ := regexp.MatchString(`^[A-Fa-f0-9]{8}-[A-Fa-f0-9]{4}-[1-5][A-Fa-f0-9]{3}-[A-Fa-f0-9]{4}-[A-Fa-f0-9]{12}$`, input)
	return match
}

func (f nonHyphenatedUUIDFormatChecker) IsFormat(input string) bool {
	match, _ := regexp.MatchString(`^[A-Fa-f0-9]{8}[A-Fa-f0-9]{4}[1-5][A-Fa-f0-9]{3}[A-Fa-f0-9]{4}[A-Fa-f0-9]{12}$`, input)
	return match
}

func (f portFormatChecker) IsFormat(input string) bool {
	port, err := strconv.ParseInt(input, 10, 0)
	return err == nil && 1 <= port && port <= 65535
}

func (f yamlFormatChecker) IsFormat(input string) bool {
	return true
}

func (f textFormatChecker) IsFormat(input string) bool {
	return true
}

func (f versionFormatChecker) IsFormat(input string) bool {
	_, err := semver.NewVersion(input)
	return err == nil
}

func (f versionConstraintFormatChecker) IsFormat(input string) bool {
	_, err := semver.NewConstraint(input)
	return err == nil
}

func registerGohanFormats(checkers gojsonschema.FormatCheckerChain) {
	checkers.Add("mac", macFormatChecker{})
	checkers.Add("ipv4", ipv4FormatChecker{})
	checkers.Add("cidr", cidrFormatChecker{})
	checkers.Add("cidr-or-ipv4", cidrOrIPv4FormatChecker{})
	checkers.Add("ipv4-network", ipv4NetworkFormatChecker{})
	checkers.Add("ipv4-address-with-cidr", ipv4AddressWithCidrFormatChecker{})
	checkers.Add("regex", regexFormatChecker{})
	checkers.Add("uuid", uuidFormatChecker{})
	checkers.Add("hyph-uuid", hyphenatedUUIDFormatChecker{})
	checkers.Add("non-hyph-uuid", nonHyphenatedUUIDFormatChecker{})
	checkers.Add("port", portFormatChecker{})
	checkers.Add("yaml", yamlFormatChecker{})
	checkers.Add("text", textFormatChecker{})
	checkers.Add("version", versionFormatChecker{})
	checkers.Add("version-constraint", versionConstraintFormatChecker{})
}
