package cliexpect

// Matcher is a function for matching data in expect operations
type Matcher func(string) []int

// RegexMatcher matches regexes in expect operations
func (s *Shell) RegexMatcher(regex string) Matcher {
	return func(input string) []int {
		return nil
	}
}
