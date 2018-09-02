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

func TestMatch(t *testing.T) {
	rw := &rw{data: "test\nrouter#"}
	sh := cliexpect.New(rw, rw, "(.+)(.)$") // Capture the prompt and the last char of it in sub-group

	full, groups, err := sh.ExpectRegex("test.+")
	assert.NoError(t, err)
	assert.Equal(t, "test\nrouter#", full)
	assert.Equal(t, []string{"test\n", "router#", "router", "#"}, groups)
}
