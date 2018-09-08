// Package cliexpect is designed to work specifically with CLI shell interfaces. Specifically, it
// always assumes a prompt will separate the data allowing easy traversal of multiple outputs.
package cliexpect

import (
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"
)

const (
	defaultTimeout  = 10 * time.Second
	defaultBuffSize = 16384
	readBuffSize    = defaultBuffSize // Must always be lte size of defaultBuffSize

	matchFmt           = `(?ms)`
	retrieveRegex      = `(.*?)(^%s$)`
	defaultPromptRegex = `\S+` // Prompt is one or more chars that are NOT whitespace
)

// ErrNoMatches represents the error returned when the expected matcher is not matched and
// the reader returns an error (if it doesn't eventually it just times out)
var ErrNoMatches = errors.New("No matches")

// ShellParam defines optional parameters for the expect shell
type ShellParam struct {
	Timeout  time.Duration
	BuffSize int

	retrieve Matcher
}

// Shell represents a structure used in expect-like interactions
type Shell struct {
	// Mandatory parameters
	in  io.Writer
	out io.Reader

	// Options parameters
	param ShellParam

	// Reader loop vars
	ch     chan error
	lock   sync.Mutex
	buffer strings.Builder
}

// New creates an expect struct using the specified Writer/Reader with default parameters
func New(in io.Writer, out io.Reader) *Shell {
	return NewWithParam(in, out, ShellParam{})
}

// validateParams validates parameters are in the appropriate range adjusting them if necessary
func validateParams(param *ShellParam) {
	if param.BuffSize < defaultBuffSize {
		param.BuffSize = defaultBuffSize
	}
	if param.Timeout < 1 {
		param.Timeout = defaultTimeout
	}
}

// NewWithParam creates an expect struct using the specified Writer/Reader with the specified parameters
func NewWithParam(in io.Writer, out io.Reader, param ShellParam) *Shell {
	validateParams(&param)

	sh := &Shell{in: in, out: out, param: param}
	sh.SetPromptRegex(defaultPromptRegex)
	// We try an size the channel based on expected number of data chunks to fill a size target of minBuffSize
	chanSize := param.BuffSize / readBuffSize
	sh.ch = make(chan error, chanSize)
	sh.resetBuff()
	go sh.reader()

	return sh
}

// SetPromptRegex sets the underlying prompt regex used to match the end of output in every expect operation
func (s *Shell) SetPromptRegex(re string) {
	s.param.retrieve = RegexMatcher(fmt.Sprintf(retrieveRegex, re))
}

// SetPrompt sets the underlying prompt to match based on a literal string and is used to match
// the end of output in every expect operation
func (s *Shell) SetPrompt(prompt string) {
	s.SetPromptRegex(fmt.Sprintf(`\Q%s\E`, prompt))
}

// resetBuff clears buffer and resizes to minBuffSize
func (s *Shell) resetBuff() {
	s.buffer.Reset()
	s.buffer.Grow(s.param.BuffSize)
}

// reader loops reading data from reader storing data in a strings.Builder and notifying of
// each operation error outcome via channel
func (s *Shell) reader() {
	buff := make([]byte, readBuffSize, readBuffSize)
	for {
		n, err := s.out.Read(buff)
		if n > 0 {
			s.lock.Lock()
			s.buffer.Write(buff[:n])
			s.lock.Unlock()
		}
		// Notify that a read operation was completed and the resulting error, if any
		s.ch <- err
		if err != nil {
			return
		}
	}
}

// SendBytes sends a byte slice to the shell
func (s *Shell) SendBytes(b []byte) error {
	_, err := s.in.Write(b)
	return err
}

// Send sends a string to the shell
func (s *Shell) Send(str string) error {
	return s.SendBytes([]byte(str))
}

// SendLine sends a string followed by a newline to the shell
func (s *Shell) SendLine(str string) error {
	return s.SendBytes([]byte(str + "\n"))
}

// Retrieve returns all the text before the next prompt. The results returned from this function
// match those from the Expect function, but assume the text before the prompt is a single match
// group (the first one)
func (s *Shell) Retrieve() (string, []string, error) {
	s.lock.Lock()
	defer s.lock.Unlock()

	var result []int
	var timeSpent time.Duration

	// Start by just getting whatever data is in the buffer without waiting
	data, dur, err := s.read(0)

	for {
		result = s.param.retrieve(data)
		// If we got an error or matches then we are done...
		if err != nil || len(result) > 0 {
			break
		}
		timeSpent += dur
		data, dur, err = s.read(s.param.Timeout - timeSpent)
	}
	// If no results then we return early
	if len(result) < 6 { // Full match + body + prompt
		if err == nil || err == io.EOF {
			err = ErrNoMatches
		}
		return "", nil, err
	}
	// Prepare for the next operation
	s.resetBuff()
	// Did we match everything? No, then save that data for next time
	if result[1] < len(data) {
		// Write the remaining data back to the buffer
		s.buffer.WriteString(data[result[1]:])
	}
	results := processResults(result, data)
	return results[0], results[1:], err
}

// processResults takes the index slice and raw data and converts tem into a slice of matched strings
func processResults(result []int, data string) []string {
	subMatchPairs := len(result)
	matches := make([]string, subMatchPairs/2)
	for i, j := 0, 0; i < subMatchPairs; i, j = i+2, j+1 {
		matches[j] = data[result[i]:result[i+1]]
	}
	return matches
}

// Expect takes a matcher and tries to match it against the current data that was received. It returns the
// entire match, all submatches, and an error, if any occurred.
func (s *Shell) Expect(m Matcher) (string, []string, error) {
	full, groups, err := s.Retrieve()
	if len(groups) < 2 {
		return "", nil, err
	}
	// We base the 2nd match purely on the body we retrieved
	result := m(groups[0])
	if len(result) < 2 {
		if err == nil || err == io.EOF {
			err = ErrNoMatches
		}
		return "", nil, err
	}
	results := processResults(result, groups[0])
	// Add the original prompt matches back onto the results
	results = append(results, groups[1:]...)
	return full, results, err
}

// ExpectRegex takes a regex as a string, compiles it, and calls Expect looking for matches. The
// return values are identical to Expect.
func (s *Shell) ExpectRegex(re string) (string, []string, error) {
	return s.Expect(RegexMatcher(re))
}

// ExpectStr takes a string, converts it to a matcher, and calls Expect looking for matches. The
// return values are identical to Expect.
func (s *Shell) ExpectStr(str string) (string, []string, error) {
	return s.Expect(StrMatcher(str))
}

// read data from the buffer and return it, waiting up to timeout if no data present. In addition
// to a string of the actual data, the actual duration of time waited is returned
func (s *Shell) read(timeout time.Duration) (data string, d time.Duration, err error) {
	var reads int
	reads, err = ackReads(s.ch)
	data = s.buffer.String()

	// Only wait if we have a timeout, no error so far, and then only if we have no data OR we did zero reads
	if timeout > 0 && err == nil && (data == "" || reads == 0) {
		d, err = s.waitForData(timeout)
		data = s.buffer.String()
	}
	return
}

// ackReads acknowledges all outstanding read operations done by reader and returns number of
// channel reads and an error if there is one
func ackReads(ch chan error) (int, error) {
	var err error
	reads := 0
	for {
		select {
		case err = <-ch:
			reads++
		default:
			return reads, err
		}
	}
}

// waitForData waits for the next read operation to complete by the reader waiting up to timeout
// in duration. It returns the duration of time it actually waited in addition to a possible error
func (s *Shell) waitForData(timeout time.Duration) (time.Duration, error) {
	t := time.Now()

	// Note the inverted ordering - this is always called under lock, so undo lock so our reader
	// goroutine can write new data to the builder
	s.lock.Unlock()
	defer s.lock.Lock()

	select {
	case err := <-s.ch:
		return time.Since(t), err
	case <-time.After(timeout):
		return timeout, errors.New("Read timed out")
	}
}
