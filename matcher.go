package cliexpect

import (
	"fmt"
	"regexp"
)

// Matcher is a function for matching data in expect operations. The returned slice matches the
// return value format of the regexp Index functions (first two positions = first/last index of whole
// match, 3rd and beyond correspond to match groups)
type Matcher func(string) []int

// RegexMatcher matches regexes in expect operations
func RegexMatcher(regex string) Matcher {
	re := regexp.MustCompile(matchFmt + regex)

	return func(input string) []int {
		return re.FindStringSubmatchIndex(input)
	}
}

// StrMatcher matches a string literal in expect operations, however, it matches the prompt as a regex
func StrMatcher(str string) Matcher {
	return RegexMatcher(fmt.Sprintf(`\Q%s\E`, str))
}
