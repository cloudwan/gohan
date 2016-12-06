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
	"net"
	"regexp"
	"strconv"

	"github.com/twinj/uuid"
	"github.com/xeipuuv/gojsonschema"
)

type macFormatChecker struct{}
type cidrFormatChecker struct{}
type cidrOrIPv4FormatChecker struct{}
type regexFormatChecker struct{}
type uuidFormatChecker struct{}
type hyphenatedUUIDFormatChecker struct{}
type nonHyphenatedUUIDFormatChecker struct{}
type portFormatChecker struct{}
type yamlFormatChecker struct{}
type textFormatChecker struct{}

func (f macFormatChecker) IsFormat(input string) bool {
	match, _ := regexp.MatchString(`^([0-9A-Fa-f]{2}:){5}[0-9A-Fa-f]{2}$`, input)
	return match
}

func (f cidrFormatChecker) IsFormat(input string) bool {
	_, _, err := net.ParseCIDR(input)
	return err == nil
}

func (f cidrOrIPv4FormatChecker) IsFormat(input string) bool {
	cidrIP, _, cidrErr := net.ParseCIDR(input)
	ip := net.ParseIP(input)
	return (cidrErr == nil && cidrIP.To4() != nil) || (ip != nil && ip.To4() != nil)
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

func registerGohanFormats(checkers gojsonschema.FormatCheckerChain) {
	checkers.Add("mac", macFormatChecker{})
	checkers.Add("cidr", cidrFormatChecker{})
	checkers.Add("cidr-or-ipv4", cidrOrIPv4FormatChecker{})
	checkers.Add("regex", regexFormatChecker{})
	checkers.Add("uuid", uuidFormatChecker{})
	checkers.Add("hyph-uuid", hyphenatedUUIDFormatChecker{})
	checkers.Add("non-hyph-uuid", nonHyphenatedUUIDFormatChecker{})
	checkers.Add("port", portFormatChecker{})
	checkers.Add("yaml", yamlFormatChecker{})
	checkers.Add("text", textFormatChecker{})
}
