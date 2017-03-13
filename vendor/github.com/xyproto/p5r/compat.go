package p5r

import (
	goregexp "regexp"
)

// Compatability functions for the "regexp" package

// MatchString return true if the string matches the regex
// Returns false if an error/timeout occurs
func (re *Regexp) MatchString(s string) bool {
	m, err := re.run(true, -1, getRunes(s))
	if err != nil {
		return false
	}
	return m != nil
}

// ReplaceAllString returns a modified string if the replacement worked
// Returns the original string if an error occured
func (re *Regexp) ReplaceAllString(input, replacement string) string {
	output, err := re.Replace(input, replacement, -1, -1)
	if err != nil {
		// Return the original string if something went wrong
		return input
	}
	// Return the string with replacements
	return output
}

// Compile a string to a Regexp, returns an error if something went wrong
func Compile(input string) (*Regexp, error) {
	return Compile2(input, 0)
}

// Compile a string to a Regexp
func MustCompile(input string) *Regexp {
	return MustCompile2(input, 0)
}

// Convert a p5r.Regex to a regexp.Regexp
func (re *Regexp) Convert() (*goregexp.Regexp, error) {
	return goregexp.Compile(re.pattern)
}

// Convert a p5r.Regex to a regexp.Regexp
func (re *Regexp) MustConvert() *goregexp.Regexp {
	return goregexp.MustCompile(re.pattern)
}

func QuoteMeta(s string) string {
	return goregexp.QuoteMeta(s)
}
