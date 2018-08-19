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

	retrieveFmt = `(.*\n)(%s)\z`
)

// Shell represents a structure used in expect-like interactions
type Shell struct {
	// Mandatory parameters
	in     io.WriteCloser
	out    io.Reader
	prompt string

	// Options parameters
	timeout  time.Duration
	buffSize int

	// Reader loop vars
	ch     chan error
	lock   sync.Mutex
	buffer strings.Builder

	retrieve Matcher
}

// New creates an expect struct using the specified Writer/Reader with default parameters
func New(in io.WriteCloser, out io.Reader, prompt string) *Shell {
	return NewParam(in, out, prompt, defaultTimeout, defaultBuffSize)
}

// NewParam creates an expect struct using the specified Writer/Reader with the specified parameters
func NewParam(in io.WriteCloser, out io.Reader, prompt string, timeout time.Duration, minBuffSize int) *Shell {
	// Can't set buffer size smaller than default
	if minBuffSize < defaultBuffSize {
		minBuffSize = defaultBuffSize
	}
	sh := &Shell{in: in, out: out, timeout: timeout, buffSize: minBuffSize}
	sh.SetPromptRE(prompt)
	// We try an size the channel based on expected number of data chunks to fill a size target of minBuffSize
	chanSize := minBuffSize / readBuffSize
	sh.ch = make(chan error, chanSize)
	sh.resetBuff()
	go sh.reader()
	return sh
}

// SetPromptRE sets the underlying prompt regex used to match end out output in every expect operation
func (s *Shell) SetPromptRE(prompt string) {
	s.prompt = prompt
	s.retrieve = s.RegexMatcher(fmt.Sprintf(retrieveFmt, prompt))
}

// resetBuff clears buffer and resizes to minBuffSize
func (s *Shell) resetBuff() {
	s.buffer.Reset()
	s.buffer.Grow(s.buffSize)
}

// reader loops reading data from reader storing data in a strings.Builder and notifying of
// each operation error outcome via channel
func (s *Shell) reader() {
	buff := make([]byte, readBuffSize, readBuffSize)
	for {
		n, err := s.out.Read(buff)
		if n > 0 {
			s.lock.Lock()
			s.buffer.Write(buff)
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

// Expect takes a matcher and tries to match it against the current data that was received. If it
// cannot make a match, it will try again waiting up to timeout to make the match.  It returns the
// entire match, all submatches, and an error, if any occured. If err is set then no matches will
// be returned
func (s *Shell) Expect(m Matcher) (string, []string, error) {
	s.lock.Lock()
	defer s.lock.Unlock()

	// Start by trying to match against existing data (will block if no data, however)
	// TODO: Track how much time we spent wait to read - subtract from first wait, if needed
	data, err := s.read()
	if err != nil {
		return "", nil, err
	}
	result := m(data)
	// No matches? try waiting one timeout and trying again
	if len(result) < 2 {
		// TODO: Track how much time we actually waited and keep looping up until timeout has expired
		if _, err := s.waitForData(s.timeout); err != nil {
			return "", nil, err
		}
		// NOTE: data is a cast to underlying []byte - no need to re-read
		// Try the match again...
		if result = m(data); len(result) < 2 {
			return "", nil, errors.New("No matches")
		}
	}
	// Prepare for the next operation
	s.resetBuff()
	// Did we match everything? No, then save that data for next time
	if result[1] < len(data) {
		// Write the remaining data back to the buffer
		s.buffer.WriteString(data[result[1]:])
	}
	// Take the index slice and convert it into all the matched strings
	subMatchPairs := len(result) - 2
	matches := make([]string, 0, subMatchPairs/2)
	for i, j := 0, 0; i < subMatchPairs; i, j = i+2, j+1 {
		matches[j] = data[i : i+1]
	}
	return data[result[0]:result[1]], matches, nil
}

// read data from the buffer and return it, waiting up to timeout if no data present
func (s *Shell) read() (string, error) {
	if err := ackReads(s.ch); err != nil {
		return "", err
	}
	data := s.buffer.String()
	if data == "" {
		if _, err := s.waitForData(s.timeout); err != nil {
			return "", err
		}
		// String() above is just a cast of underlying []byte - no need to update data
	}
	return data, nil
}

// ackReads acknowledges all outstanding read operations done by reader and return an error if there is one
func ackReads(ch chan error) error {
	var err error
	for {
		select {
		case err = <-ch:
		default:
			return err
		}
	}
}

// waitForData waits for the next read operation to complete by the reader waiting up to timeout
// in duration. If it doesn't time out, it will return the duration that it waited
func (s *Shell) waitForData(timeout time.Duration) (time.Duration, error) {
	oldTime := time.Now()

	// Note the inverted ordering - this is always called under lock, so undo lock so our reader
	// goroutine can write new data to the builder
	s.lock.Unlock()
	defer s.lock.Lock()

	select {
	case err := <-s.ch:
		if err != nil {
			return time.Duration(0), err
		}
		// Return now long we waited
		return oldTime.Sub(time.Now()), nil
	case <-time.After(timeout):
		return time.Duration(0), errors.New("Read timed out")
	}
}
