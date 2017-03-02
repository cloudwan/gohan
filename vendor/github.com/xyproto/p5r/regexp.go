/*
Package p5r is a regexp package that has an interface similar to Go's framework regexp engine but uses a
more feature full regex engine behind the scenes.

It doesn't have constant time guarantees, but it allows backtracking and is compatible with Perl5 and .NET.
You'll likely be better off with the RE2 engine from the regexp package and should only use this if you
need to write very complex patterns or require compatibility with .NET.
*/
package p5r

import (
	"errors"
	"math"
	"strconv"
	"sync"
	"time"

	"github.com/xyproto/p5r/syntax"
	"bytes"
)

// Default timeout used when running regexp matches -- "forever"
var DefaultMatchTimeout = time.Duration(math.MaxInt64)

// Regexp is the representation of a compiled regular expression.
// A Regexp is safe for concurrent use by multiple goroutines.
type Regexp struct {
	//timeout when trying to find matches
	MatchTimeout time.Duration

	// read-only after Compile
	pattern string       // as passed to Compile
	options RegexOptions // options

	caps     map[int]int    // capnum->index
	capnames map[string]int //capture group name -> index
	capslist []string       //sorted list of capture group names
	capsize  int            // size of the capture array

	code *syntax.Code // compiled program

	// cache of machines for running regexp
	muRun  sync.Mutex
	runner []*runner
}

// Compile parses a regular expression and returns, if successful,
// a Regexp object that can be used to match against text.
func Compile2(expr string, opt RegexOptions) (*Regexp, error) {
	// parse it
	tree, err := syntax.Parse(expr, syntax.RegexOptions(opt))
	if err != nil {
		return nil, err
	}

	// translate it to code
	code, err := syntax.Write(tree)
	if err != nil {
		return nil, err
	}

	// return it
	return &Regexp{
		pattern:      expr,
		options:      opt,
		caps:         code.Caps,
		capnames:     tree.Capnames,
		capslist:     tree.Caplist,
		capsize:      code.Capsize,
		code:         code,
		MatchTimeout: DefaultMatchTimeout,
	}, nil
}

// MustCompile2 is like Compile but panics if the expression cannot be parsed.
// It simplifies safe initialization of global variables holding compiled regular
// expressions.
func MustCompile2(str string, opt RegexOptions) *Regexp {
	regexp, error := Compile2(str, opt)
	if error != nil {
		panic(`regexp2: Compile(` + quote(str) + `): ` + error.Error())
	}
	return regexp
}

// Escape adds backslashes to any special characters in the input string
func Escape(input string) string {
	return syntax.Escape(input)
}

// Unescape removes any backslashes from previously-escaped special characters in the input string
func Unescape(input string) (string, error) {
	return syntax.Unescape(input)
}

// String returns the source text used to compile the regular expression.
func (re *Regexp) String() string {
	return re.pattern
}

func quote(s string) string {
	if strconv.CanBackquote(s) {
		return "`" + s + "`"
	}
	return strconv.Quote(s)
}

// RegexOptions impact the runtime and parsing behavior
// for each specific regex.  They are setable in code as well
// as in the regex pattern itself.
type RegexOptions int32

const (
	None                    RegexOptions = 0x0
	IgnoreCase                           = 0x0001 // "i"
	Multiline                            = 0x0002 // "m"
	ExplicitCapture                      = 0x0004 // "n"
	Compiled                             = 0x0008 // "c"
	Singleline                           = 0x0010 // "s"
	IgnorePatternWhitespace              = 0x0020 // "x"
	RightToLeft                          = 0x0040 // "r"
	Debug                                = 0x0080 // "d"
	ECMAScript                           = 0x0100 // "e"
)

func (re *Regexp) RightToLeft() bool {
	return re.options&RightToLeft != 0
}

func (re *Regexp) Debug() bool {
	return re.options&Debug != 0
}

// Replace searches the input string and replaces each match found with the replacement text.
// Count will limit the number of matches attempted and startAt will allow
// us to skip past possible matches at the start of the input (left or right depending on RightToLeft option)
func (re *Regexp) Replace(input, replacement string, startAt, count int) (string, error) {
	data, err := syntax.NewReplacerData(replacement, re.caps, re.capsize, re.capnames, syntax.RegexOptions(re.options))
	if err != nil {
		return "", err
	}
	//TODO: cache ReplacerData

	return replace(re, data, input, startAt, count)
}

// FindStringMatch searches the input string for a Regexp match
func (re *Regexp) FindStringMatch(s string) (*Match, error) {
	// convert string to runes
	return re.run(false, -1, getRunes(s))
}

// FindRunesMatch searches the input rune slice for a Regexp match
func (re *Regexp) FindRunesMatch(r []rune) (*Match, error) {
	return re.run(false, -1, r)
}

// FindStringMatchStartingAt searches the input string for a Regexp match starting at the startAt index
func (re *Regexp) FindStringMatchStartingAt(s string, startAt int) (*Match, error) {
	if startAt > len(s) {
		return nil, errors.New("startAt must be less than the length of the input string")
	}
	r, startAt := re.getRunesAndStart(s, startAt)
	if startAt == -1 {
		// we didn't find our start index in the string -- that's a problem
		return nil, errors.New("startAt must align to the start of a valid rune in the input string")
	}

	return re.run(false, startAt, r)
}

// FindRunesMatchStartingAt searches the input rune slice for a Regexp match starting at the startAt index
func (re *Regexp) FindRunesMatchStartingAt(r []rune, startAt int) (*Match, error) {
	return re.run(false, startAt, r)
}

// FindNextMatch returns the next match in the same input string as the match parameter.
// Will return nil if there is no next match or if given a nil match.
func (re *Regexp) FindNextMatch(m *Match) (*Match, error) {
	if m == nil {
		return nil, nil
	}

	// If previous match was empty, advance by one before matching to prevent
	// infinite loop
	startAt := m.textpos
	if m.Length == 0 {
		if m.textpos == len(m.text) {
			return nil, nil
		}

		if re.RightToLeft() {
			startAt--
		} else {
			startAt++
		}
	}
	return re.run(false, startAt, m.text)
}

// MatchString return true if the string matches the regex
// error will be set if a timeout occurs
func (re *Regexp) MatchString2(s string) (bool, error) {
	m, err := re.run(true, -1, getRunes(s))
	if err != nil {
		return false, err
	}
	return m != nil, nil
}

func (re *Regexp) getRunesAndStart(s string, startAt int) ([]rune, int) {
	if startAt < 0 {
		if re.RightToLeft() {
			r := getRunes(s)
			return r, len(r)
		}
		return getRunes(s), 0
	}
	ret := make([]rune, len(s))
	i := 0
	runeIdx := -1
	for strIdx, r := range s {
		if strIdx == startAt {
			runeIdx = i
		}
		ret[i] = r
		i++
	}
	return ret[:i], runeIdx
}

func getRunes(s string) []rune {
	ret := make([]rune, len(s))
	i := 0
	for _, r := range s {
		ret[i] = r
		i++
	}
	return ret[:i]
}

// MatchRunes return true if the runes matches the regex
// error will be set if a timeout occurs
func (re *Regexp) MatchRunes(r []rune) (bool, error) {
	m, err := re.run(true, -1, r)
	if err != nil {
		return false, err
	}
	return m != nil, nil
}

// GetGroupNames Returns the set of strings used to name capturing groups in the expression.
func (re *Regexp) GetGroupNames() []string {
	var result []string

	if re.capslist == nil {
		result = make([]string, re.capsize)

		for i := 0; i < re.capsize; i++ {
			result[i] = strconv.Itoa(i)
		}
	} else {
		result = make([]string, len(re.capslist))
		copy(result, re.capslist)
	}

	return result
}

// GetGroupNumbers returns the integer group numbers corresponding to a group name.
func (re *Regexp) GetGroupNumbers() []int {
	var result []int

	if re.caps == nil {
		result = make([]int, re.capsize)

		for i := 0; i < re.capsize; i++ {
			result[i] = i
		}
	} else {
		result = make([]int, len(re.caps))

		for k, v := range re.caps {
			result[v] = k
		}
	}

	return result
}

// GroupNameFromNumber retrieves a group name that corresponds to a group number.
// It will return "" for and unknown group number.  Unnamed groups automatically
// receive a name that is the decimal string equivalent of its number.
func (re *Regexp) GroupNameFromNumber(i int) string {
	if re.capslist == nil {
		if i >= 0 && i < re.capsize {
			return strconv.Itoa(i)
		}

		return ""
	}

	if re.caps != nil {
		var ok bool
		if i, ok = re.caps[i]; !ok {
			return ""
		}
	}

	if i >= 0 && i < len(re.capslist) {
		return re.capslist[i]
	}

	return ""
}

// GroupNumberFromName returns a group number that corresponds to a group name.
// Returns -1 if the name is not a recognized group name.  Numbered groups
// automatically get a group name that is the decimal string equivalent of its number.
func (re *Regexp) GroupNumberFromName(name string) int {
	// look up name if we have a hashtable of names
	if re.capnames != nil {
		if k, ok := re.capnames[name]; ok {
			return k
		}

		return -1
	}

	// convert to an int if it looks like a number
	result := 0
	for i := 0; i < len(name); i++ {
		ch := name[i]

		if ch > '9' || ch < '0' {
			return -1
		}

		result *= 10
		result += int(ch - '0')
	}

	// return int if it's in range
	if result >= 0 && result < re.capsize {
		return result
	}

	return -1
}

// ReplaceAllFunc returns a copy of src in which all matches of the
// Regexp have been replaced by the return value of function repl applied
// to the matched byte slice. The replacement returned by repl is substituted
// directly, without using Expand.
func (regex *Regexp) ReplaceAllFunc(src []byte, repl func([]byte) []byte) []byte {
	return []byte(regex.ReplaceAllStringFunc(string(src), func(s string) string {
		return string(repl([]byte(s)))
	}))
}

// ReplaceAllStringFunc returns a copy of src in which all matches of the
// Regexp have been replaced by the return value of function repl applied
// to the matched substring. The replacement returned by repl is substituted
// directly, without using Expand.
func (regex *Regexp) ReplaceAllStringFunc(src string, repl func(string) string) string {
	match, err := regex.FindStringMatchStartingAt(src, -1)

	if err != nil {
		return ""
	}
	if match == nil {
		return src
	}
	buf := &bytes.Buffer{}
	text := match.text
	prevat := 0

	for match != nil {
		if match.Index != prevat {
			buf.WriteString(string(text[prevat:match.Index]))
		}
		prevat = match.Index + match.Length
		buf.WriteString(repl(src[match.Index:prevat]))
		match, err = regex.FindNextMatch(match)
		if err != nil {
			return ""
		}
	}

	if prevat < len(text) {
		buf.WriteString(string(text[prevat:]))
	}
	return buf.String()
}

// FindAllStringSubmatch is the 'All' version of FindStringSubmatch; it
// returns a slice of all successive matches of the expression, as defined by
// the 'All' description in the package comment.
// A return value of nil indicates no match.
func (regex *Regexp) FindAllStringSubmatch(source string, n int) [][]string {
	indices := regex.FindAllStringSubmatchIndex(source, n)
	if indices == nil {
		return nil
	}
	var result [][]string = make([][]string, len(indices))
	for i := range indices {
		result[i] = make([]string, len(indices[i]) / 2)
		for j := 0; j < len(result[i]); j++ {
			idx := 2 * j
			if indices[i][idx] == -1 || indices[i][idx + 1] == -1 {
				continue
			}
			result[i][j] = source[indices[i][idx]:indices[i][idx + 1]]
		}
	}
	return result
}

// FindStringSubmatch returns a slice of strings holding the text of the
// leftmost match of the regular expression in s and the matches, if any, of
// its subexpressions, as defined by the 'Submatch' description in the
// package comment.
// A return value of nil indicates no match.
func (regex *Regexp) FindStringSubmatch(source string) []string {
	indices := regex.FindStringSubmatchIndex(source)
	if indices == nil {
		return nil
	}
	var result []string = make([]string, len(indices) / 2)
	for i := 0; i < len(result); i++ {
		idx := 2 * i
		if indices[idx] == -1 || indices[idx + 1] == -1 {
			continue
		}
		result[i] = source[indices[idx]:indices[idx + 1]]
	}
	return result
}

// FindAllStringIndex is the 'All' version of FindStringIndex; it returns a
// slice of all successive matches of the expression, as defined by the 'All'
// description in the package comment.
// Checks up to n characters or all if n is negative.
// A return value of nil indicates no match.
func (regex *Regexp) FindAllStringIndex(source string, n int) [][]int {
	if n < 0 {
		n = len(source) + 1
	}
	match, err  := regex.FindStringMatch(source)
	if err != nil {
		return nil
	}
	var result [][]int
	for match != nil && match.Index < n {
		indices := make([]int, 2)
		indices[0] = match.Index
		indices[1] = match.Index + match.Length
		result = append(result, indices)
		if indices[1] >= len(source) {
			break
		}
		match, err = regex.FindNextMatch(match)
		if err != nil {
			return nil
		}
	}
	return result
}

// FindStringIndex returns a two-element slice of integers defining the
// location of the leftmost match in s of the regular expression. The match
// itself is at s[loc[0]:loc[1]].
// A return value of nil indicates no match.
func (regex *Regexp) FindStringIndex(source string) []int {
	match, err  := regex.FindStringMatch(source)
	if err != nil || match == nil {
		return nil
	}
	result := make([]int, 2)
	result[0] = match.Index
	result[1] = match.Index + match.Length
	return result
}

// FindAllSubmatchIndex is the 'All' version of FindSubmatchIndex; it returns
// a slice of all successive matches of the expression, as defined by the
// 'All' description in the package comment.
// A return value of nil indicates no match.
func (regex *Regexp) FindAllSubmatchIndex(b []byte, n int) [][]int {
	return regex.FindAllStringSubmatchIndex(string(b), n)
}

// FindAllStringSubmatchIndex is the 'All' version of
// FindStringSubmatchIndex; it returns a slice of all successive matches of
// the expression, as defined by the 'All' description in the package
// comment.
// A return value of nil indicates no match.
func (regex *Regexp) FindAllStringSubmatchIndex(source string, n int) [][]int {
	if n < 0 {
		n = len(source) + 1
	}
	m, err  := regex.FindStringMatch(source)
	if err != nil || m == nil {
		return nil
	}
	var result [][]int
	for m != nil && m.Index < n {
		indices := getIndicesFromGroup(m)
		result = append(result, indices)
		if indices[1] >= len(source) {
			break
		}
		m, _ = regex.FindNextMatch(m)
	}
	return result
}

func getIndicesFromGroup(match *Match) []int {
	var indices []int
	for _, group := range match.Groups() {
		if len(group.Captures) > 0 {
			capture := group.Captures[len(group.Captures) - 1]
			indices = append(indices, capture.Index, capture.Index + capture.Length)
		} else {
			indices = append(indices, -1, -1)
		}
	}
	return indices
}

// FindStringSubmatchIndex returns a slice holding the index pairs
// identifying the leftmost match of the regular expression in s and the
// matches, if any, of its subexpressions, as defined by the 'Submatch' and
// 'Index' descriptions in the package comment.
// A return value of nil indicates no match.
func (regex *Regexp) FindStringSubmatchIndex(source string) []int {
	match, err  := regex.FindStringMatch(source)
	if err != nil || match == nil {
		return nil
	}
	var result []int
	for _, group := range match.Groups() {
		if len(group.Captures) > 0 {
			lastCapture := group.Captures[len(group.Captures) - 1]
			result = append(result, lastCapture.Index, lastCapture.Index + lastCapture.Length)
		} else {
			result = append(result, -1, -1)
		}
	}
	return result
}

// Match reports whether the Regexp matches the byte slice b.
func (regex *Regexp) Match(b []byte) bool {
	return regex.MatchString(string(b))
}
