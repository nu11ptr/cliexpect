package cliexpect_test

import (
	"testing"

	"github.com/nu11ptr/cliexpect"
	"github.com/stretchr/testify/assert"
)

func TestRegexMatcher(t *testing.T) {
	data := "blah blah\n"
	m := cliexpect.RegexMatcher(".+")
	result := m(data)
	assert.Equal(t, []int{0, 10}, result)
}

func TestStrMatcher(t *testing.T) {
	data := "blah blah\n"
	m := cliexpect.StrMatcher("blah blah\n")
	result := m(data)
	assert.Equal(t, []int{0, 10}, result)
}
