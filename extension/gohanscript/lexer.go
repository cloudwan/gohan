// Copyright (C) 2016  Juniper Networks, Inc.
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

package gohanscript

import (
	"fmt"
	"strconv"
	"unicode/utf8"

	"github.com/cloudwan/gohan/util"
)

//Additonal lexer code for support Ansible-like yaml notion.

const (
	eof = -1
)

type lexer struct {
	code     []byte
	position int
	err      error
}

func newLexer(src string) *lexer {
	return &lexer{
		position: 0,
		code:     []byte(src),
	}
}

func isLetter(c rune) bool {
	return 'a' <= c && c <= 'z' || 'A' <= c && c <= 'Z' || isDigit(c) || c == '_'
}

func isDigit(c rune) bool {
	return '0' <= c && c <= '9'
}

func isWhiteSpace(c rune) bool {
	return c == ' ' || c == '\t' || c == '\n'
}

func isNotWhiteSpace(c rune) bool {
	return !(c == ' ' || c == '\t' || c == '\n' || c == eof)
}

func (l *lexer) next() rune {
	if l.isEOF() {
		return eof
	}
	c, width := utf8.DecodeRune(l.code[l.position:])
	l.position += width
	return c
}

func (l *lexer) peek() rune {
	if l.isEOF() {
		return eof
	}
	c, _ := utf8.DecodeRune(l.code[l.position:])
	return c
}

func (l *lexer) isEOF() bool {
	return l.position >= len(l.code) || l.err != nil
}

func (l *lexer) scanWhiteSpace() {
	for isWhiteSpace(l.peek()) {
		l.next()
	}
}

func (l *lexer) error(error string) {
	l.err = fmt.Errorf("Error %s at position %d", error, l.position)
}

func (l *lexer) clearError() {
	l.err = nil
}

func (l *lexer) scanIdentifier() string {
	var result []rune
	firstChar := l.peek()
	if !isLetter(firstChar) {
		l.error("Identifier expected1")
	}
	for isLetter(l.peek()) {
		result = append(result, l.next())
	}
	return string(result)
}

func (l *lexer) scanNumber() int {
	var result int
	firstChar := l.peek()
	if !isDigit(firstChar) {
		l.error("Digit expected")
	}
	for isDigit(l.peek()) {
		c := l.next()
		result = result*10 + (int(c) - '0')
	}
	return result
}

func (l *lexer) scanQuotedString() string {
	var result []rune
	firstChar := l.peek()
	if firstChar != '"' {
		l.error("value should be quoted")
		return ""
	}
	l.next()
	for {
		c := l.peek()
		switch c {
		case '\\':
			l.next()
			result = append(result, l.next())
		case '"':
			l.next()
			return string(result)
		case eof:
			l.error("string isn't terminated")
			return ""
		default:
			result = append(result, l.next())
		}
	}
}

func (l *lexer) scanString() string {
	var result []rune
	for isNotWhiteSpace(l.peek()) {
		result = append(result, l.next())
	}
	return string(result)
}

func (l *lexer) parseDict() map[string]interface{} {
	result := map[string]interface{}{}
	for {
		l.scanWhiteSpace()
		key := l.scanIdentifier()
		var value interface{}
		if l.err != nil {
			return nil
		}
		l.scanWhiteSpace()
		c := l.next()
		if c != '=' {
			l.error(" = expected")
			return nil
		}
		l.scanWhiteSpace()
		c = l.peek()
		switch c {
		case '"':
			value = l.scanQuotedString()
			if l.err != nil {
				return nil
			}
		default:
			s := l.scanString()
			if l.err != nil {
				return nil
			}
			i, err := strconv.Atoi(s)
			if err == nil {
				value = i
			} else {
				value = s
			}

		}

		result[key] = value
		l.scanWhiteSpace()
		c = l.peek()
		if c == eof {
			return result
		}
	}
}

func parseCode(key string, code interface{}) map[string]interface{} {
	switch c := code.(type) {
	case string:
		l := newLexer(c)
		result := l.parseDict()
		if l.err != nil {
			args := map[string]interface{}{}
			args[key] = c
			return args
		}
		return result
	case map[string]interface{}:
		return c
	case map[interface{}]interface{}:
		return util.MaybeMap(c)
	}
	return map[string]interface{}{key: code}
}
