package cliexpect_test

import (
	"testing"

	"github.com/nu11ptr/cliexpect"
	"github.com/stretchr/testify/assert"
)

type rw struct {
	data string
}

func (rw *rw) Read(b []byte) (int, error) {
	copy(b, rw.data)
	defer func() { rw.data = "" }()
	if rw.data == "" {
		select {} // block forever when there is no data
	}
	return len(rw.data), nil
}

func (rw *rw) Write(b []byte) (int, error) {
	return 0, nil
}

func TestBasicMatch(t *testing.T) {
	data := "test\nrouter#"
	rw := &rw{data: data}
	param := cliexpect.ShellParam{Prompt: "(.+)(.)"} // Capture the prompt and the last char of it
	sh := cliexpect.NewWithParam(rw, rw, param)

	full, groups, err := sh.ExpectRegex("test.+")
	assert.NoError(t, err)
	assert.Equal(t, data, full)
	assert.Equal(t, []string{"test\n", "router#", "router", "#"}, groups)
}

func TestRetrieve(t *testing.T) {
	data := "test\nrouter#"
	rw := &rw{data: data}
	sh := cliexpect.New(rw, rw)

	full, groups, err := sh.Retrieve()
	assert.NoError(t, err)
	assert.Equal(t, data, full)
	assert.Equal(t, []string{"test\n", "router#"}, groups)
}

func TestMultiRetrieve(t *testing.T) {
	data := "test\nrouter#\nrouter#\nblah blah\nbogus bogus\nrouter>"
	rw := &rw{data: data}
	sh := cliexpect.New(rw, rw)
	sh.SetPrompt("([^\n]+)[#>]") // Capture the base prompt - must end with # or >

	full, groups, err := sh.Retrieve()
	assert.NoError(t, err)
	assert.Equal(t, "test\nrouter#", full)
	assert.Equal(t, []string{"test\n", "router#", "router"}, groups)

	full, groups, err = sh.Retrieve()
	assert.NoError(t, err)
	assert.Equal(t, "\nrouter#", full)
	assert.Equal(t, []string{"\n", "router#", "router"}, groups)

	full, groups, err = sh.Retrieve()
	assert.NoError(t, err)
	assert.Equal(t, "\nblah blah\nbogus bogus\nrouter>", full)
	assert.Equal(t, []string{"\nblah blah\nbogus bogus\n", "router>", "router"}, groups)
}
