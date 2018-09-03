package cliexpect_test

import (
	"errors"
	"testing"
	"time"

	"github.com/nu11ptr/cliexpect"
	"github.com/stretchr/testify/assert"
)

type reader struct {
	data string
	err  error
}

func (r *reader) Read(b []byte) (int, error) {
	copy(b, r.data)
	defer func() { r.data = "" }()
	if r.data == "" {
		select {} // block forever when there is no data
	}
	return len(r.data), r.err
}

type errReader struct{}

func (r errReader) Read(b []byte) (int, error) {
	return 0, errors.New("Bad read")
}

type writer struct {
	data []byte
}

func (w *writer) Write(b []byte) (int, error) {
	w.data = b
	return len(b), nil
}

func TestBasicMatch(t *testing.T) {
	data := "test\nrouter#"
	param := cliexpect.ShellParam{Prompt: "(.+)(.)"} // Capture the prompt and the last char of it
	sh := cliexpect.NewWithParam(new(writer), &reader{data: data}, param)

	full, groups, err := sh.ExpectRegex("test.+")
	assert.NoError(t, err)
	assert.Equal(t, data, full)
	assert.Equal(t, []string{"test\n", "router#", "router", "#"}, groups)
}

func TestRetrieve(t *testing.T) {
	data := "test\nrouter#"
	sh := cliexpect.New(new(writer), &reader{data: data})

	full, groups, err := sh.Retrieve()
	assert.NoError(t, err)
	assert.Equal(t, data, full)
	assert.Equal(t, []string{"test\n", "router#"}, groups)
}

func TestMultiRetrieve(t *testing.T) {
	data := "test\nrouter#\nrouter#\nblah blah\nbogus bogus\nrouter>"
	sh := cliexpect.New(new(writer), &reader{data: data})
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

func TestTimeout(t *testing.T) {
	param := cliexpect.ShellParam{Timeout: 1 * time.Nanosecond}
	sh := cliexpect.NewWithParam(new(writer), new(reader), param)

	full, groups, err := sh.ExpectStr("testing\n")
	assert.Error(t, err)
	assert.Equal(t, "", full)
	assert.Nil(t, groups)
}

func TestReadError(t *testing.T) {
	sh := cliexpect.New(new(writer), new(errReader))

	full, groups, err := sh.ExpectStr("testing\n")
	assert.Error(t, err)
	assert.Equal(t, "", full)
	assert.Nil(t, groups)
}

func TestSendBytes(t *testing.T) {
	w := new(writer)
	sh := cliexpect.New(w, new(reader))
	data := []byte("bogus")

	assert.NoError(t, sh.SendBytes(data))
	assert.Equal(t, data, w.data)
}

func TestSend(t *testing.T) {
	w := new(writer)
	sh := cliexpect.New(w, new(reader))
	data := "bogus"

	assert.NoError(t, sh.Send(data))
	assert.Equal(t, []byte(data), w.data)
}

func TestSendLine(t *testing.T) {
	w := new(writer)
	sh := cliexpect.New(w, new(reader))
	data := "bogus"

	assert.NoError(t, sh.SendLine(data))
	assert.Equal(t, []byte(data+"\n"), w.data)
}
