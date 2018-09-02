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
func (s *Shell) RegexMatcher(regex string) Matcher {
	// TODO: Should we move this into the function? Right now if prompt changes... it won't match
	re := regexp.MustCompile(fmt.Sprintf(matchFmt, regex, s.prompt))

	return func(input string) []int {
		return re.FindStringSubmatchIndex(input)
	}
}