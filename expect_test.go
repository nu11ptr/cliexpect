package cliexpect_test

import (
	"errors"
	"io"
	"strings"
	"testing"
	"testing/iotest"
	"time"

	"github.com/nu11ptr/cliexpect"
	"github.com/stretchr/testify/assert"
)

type blockingReader struct {
	data string
}

func (r *blockingReader) Read(b []byte) (int, error) {
	copy(b, r.data)
	defer func() { r.data = "" }()
	if r.data == "" {
		select {} // block forever when there is no data
	}
	return len(r.data), nil
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

func TestMatch(t *testing.T) {
	data := "test\nrouter#"

	tests := []struct {
		name, regex, full string
		groups            []string
		err               error
		reader            func(string) io.Reader
	}{
		{"Fake", "test.+", data, []string{"test\n", "router#"}, nil,
			func(s string) io.Reader { return &blockingReader{data: s} }},
		{"Strings", "test.+", data, []string{"test\n", "router#"}, nil,
			func(s string) io.Reader { return strings.NewReader(s) }},
		{"StringsNoMatch", "testing.+", "", nil, cliexpect.ErrNoMatches,
			func(s string) io.Reader { return strings.NewReader(s) }},
		{"DataErrReader", "test.+", data, []string{"test\n", "router#"}, io.EOF,
			func(s string) io.Reader { return iotest.DataErrReader(strings.NewReader(s)) }},
		{"DataErrReaderNoMatch", "testing.+", "", nil, cliexpect.ErrNoMatches,
			func(s string) io.Reader { return iotest.DataErrReader(strings.NewReader(s)) }},
		{"OneByteReader", "test.+", data, []string{"test\n", "router#"}, nil,
			func(s string) io.Reader { return iotest.OneByteReader(strings.NewReader(s)) }},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			sh := cliexpect.New(new(writer), test.reader(data))
			sh.SetPromptRegex(`\S+#`) // Must end in pound so OneByteReader knows it needs more data
			full, groups, err := sh.ExpectRegex(test.regex)
			// NOTE: Running the test with "-race" alters the ordering the goroutines are
			// performed and the error on 'Strings' fluctuates between nil and EOF...however, we
			// don't really care whether or not we get an EOF or nil and both are correct
			if test.name == "Strings" {
				if err != io.EOF && err != nil {
					t.Fail()
				}
			} else { // All other tests
				assert.Equal(t, test.err, err)
			}
			assert.Equal(t, test.full, full)
			assert.Equal(t, test.groups, groups)
		})
	}
}

func TestSubMatches(t *testing.T) {
	data := "test\nrouter#"
	sh := cliexpect.New(new(writer), &blockingReader{data: data})
	sh.SetPromptRegex(`(\w+)([#|>])`) // Capture the prompt and the last char of it

	full, groups, err := sh.ExpectRegex("test.+")
	assert.NoError(t, err)
	assert.Equal(t, data, full)
	assert.Equal(t, []string{"test\n", "router#", "router", "#"}, groups)
}

func TestRetrieve(t *testing.T) {
	data := "test\nrouter#"
	sh := cliexpect.New(new(writer), &blockingReader{data: data})
	sh.SetPromptRegex("[^\n]+#") // Prompt must end with hash/pound

	full, groups, err := sh.Retrieve()
	assert.NoError(t, err)
	assert.Equal(t, data, full)
	assert.Equal(t, []string{"test\n", "router#"}, groups)
}

func TestMultiRetrieve(t *testing.T) {
	data := "test\nrouter#\nrouter#\nblah blah\nbogus bogus\nrouter>"
	sh := cliexpect.New(new(writer), &blockingReader{data: data})
	sh.SetPromptRegex("([^\n]+)[#>]") // Capture the base prompt - must end with # or >

	tests := []struct {
		name   string
		full   string
		groups []string
		err    error
	}{
		{"Retrieve1", "test\nrouter#", []string{"test\n", "router#", "router"}, nil},
		{"Retrieve2", "\nrouter#", []string{"\n", "router#", "router"}, nil},
		{"Retrieve3", "\nblah blah\nbogus bogus\nrouter>",
			[]string{"\nblah blah\nbogus bogus\n", "router>", "router"}, nil},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			full, groups, err := sh.Retrieve()
			assert.Equal(t, test.err, err)
			assert.Equal(t, test.full, full)
			assert.Equal(t, test.groups, groups)
		})
	}
}

func TestTimeout(t *testing.T) {
	param := cliexpect.ShellParam{Timeout: 1 * time.Nanosecond}
	sh := cliexpect.NewWithParam(new(writer), new(blockingReader), param)

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
	sh := cliexpect.New(w, new(blockingReader))
	data := []byte("bogus")

	assert.NoError(t, sh.SendBytes(data))
	assert.Equal(t, data, w.data)
}

func TestSend(t *testing.T) {
	w := new(writer)
	sh := cliexpect.New(w, new(blockingReader))
	data := "bogus"

	assert.NoError(t, sh.Send(data))
	assert.Equal(t, []byte(data), w.data)
}

func TestSendLine(t *testing.T) {
	w := new(writer)
	sh := cliexpect.New(w, new(blockingReader))
	data := "bogus"

	assert.NoError(t, sh.SendLine(data))
	assert.Equal(t, []byte(data+"\n"), w.data)
}
