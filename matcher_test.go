package cliexpect_test

import (
	"testing"

	"github.com/nu11ptr/cliexpect"
	"github.com/stretchr/testify/assert"
)

func TestRegexMatcher(t *testing.T) {
	sh := cliexpect.New(new(writer), new(blockingReader))
	data := "blah blah\nrouter#"
	m := sh.RegexMatcher(".+")
	result := m(data)
	assert.Equal(t, []int{0, 17, 0, 10, 10, 17}, result)
}

func TestStrMatcher(t *testing.T) {
	sh := cliexpect.New(new(writer), new(blockingReader))
	data := "blah blah\nrouter#"
	m := sh.StrMatcher("blah blah\n")
	result := m(data)
	assert.Equal(t, []int{0, 17, 0, 10, 10, 17}, result)
}
