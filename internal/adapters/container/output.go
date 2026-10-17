package container

import (
	"errors"
	"io"
	"sync"
)

// maxBufferedOutput bounds provider output that the consumer has not read
// yet. Buffering keeps control replies, such as a census or an exit, from
// waiting for the consumer; a consumer that falls this far behind fails the
// session instead of growing memory without bound.
const maxBufferedOutput = 32 << 20

var errOutputOverflow = errors.New("provider output exceeds the unread buffer; the consumer is not reading")

// outputBuffer is an in-memory pipe whose writer never waits for the reader.
type outputBuffer struct {
	mu       sync.Mutex
	ready    *sync.Cond
	data     []byte
	closed   bool // the writer finished; the reader drains, then sees EOF
	detached bool // the reader left; writes are discarded
}

func newOutputBuffer() *outputBuffer {
	b := &outputBuffer{}
	b.ready = sync.NewCond(&b.mu)
	return b
}

func (b *outputBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	switch {
	case b.detached:
		return len(p), nil
	case b.closed:
		return 0, io.ErrClosedPipe
	case len(b.data)+len(p) > maxBufferedOutput:
		return 0, errOutputOverflow
	}
	b.data = append(b.data, p...)
	b.ready.Broadcast()
	return len(p), nil
}

func (b *outputBuffer) Read(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	for len(b.data) == 0 && !b.closed && !b.detached {
		b.ready.Wait()
	}
	if len(b.data) == 0 {
		return 0, io.EOF
	}
	n := copy(p, b.data)
	b.data = b.data[n:]
	if len(b.data) == 0 {
		b.data = nil
	}
	return n, nil
}

// close ends the stream after the remaining data is read.
func (b *outputBuffer) close() {
	b.mu.Lock()
	b.closed = true
	b.ready.Broadcast()
	b.mu.Unlock()
}

// detach drops unread data and discards later writes.
func (b *outputBuffer) detach() {
	b.mu.Lock()
	b.detached, b.data = true, nil
	b.ready.Broadcast()
	b.mu.Unlock()
}
