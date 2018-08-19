package cliexpect

import (
	"io"
	"strings"
	"sync"
	"time"
)

const (
	defaultTimeout  = 10 * time.Second
	defaultBuffSize = 16384
	readBuffSize    = defaultBuffSize // Must always be lte size of defaultBuffSize
)

// Shell represents a structure used in expect-like interactions
type Shell struct {
	in       io.WriteCloser
	out      io.Reader
	ch       chan error
	timeout  time.Duration
	buffSize int

	lock   sync.Mutex
	buffer strings.Builder
}

// New creates an expect struct using the specified Writer/Reader with default parameters
func New(in io.WriteCloser, out io.Reader) *Shell {
	return NewParam(in, out, defaultTimeout, defaultBuffSize)
}

// NewParam creates an expect struct using the specified Writer/Reader with the specified parameters
func NewParam(in io.WriteCloser, out io.Reader, timeout time.Duration, minBuffSize int) *Shell {
	// Can't set buffer size smaller than default
	if minBuffSize < defaultBuffSize {
		minBuffSize = defaultBuffSize
	}
	sh := &Shell{in: in, out: out, timeout: timeout, buffSize: minBuffSize}
	// We try an size the channel based on expected number of data chunks to fill a size target of minBuffSize
	chanSize := minBuffSize / readBuffSize
	sh.ch = make(chan error, chanSize)
	sh.resetBuff()
	go sh.reader()
	return sh
}

// resetBuff clears buffer and resizes to minBuffSize
func (s *Shell) resetBuff() {
	s.buffer.Reset()
	s.buffer.Grow(s.buffSize)
}

// reader loops reading data from reader storing data in strings.Builder and notifying of
// each operation and errors status via channel
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
