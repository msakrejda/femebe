package femebe

// N.B.: these are test utilities, but they can't easily be shared
// across *just* tests across different subpackages, so they're in a
// common package. There is nothing of value outside of testing here
// (and if there is, it should be moved to another file). Hopefully we
// can find a better way to share test code at some point.

import (
	"bytes"
	"io"
)

type closableBuffer struct {
	io.ReadWriter
}

func (c *closableBuffer) Close() error {
	// noop, to satisfy interface
	return nil
}

func newClosableBuffer(buf *bytes.Buffer) *closableBuffer {
	return &closableBuffer{buf}
}

type inMemRwc struct {
	io.ReadWriter
	Contents *bytes.Buffer
}

func NewInMemRwc() io.ReadWriteCloser {
	buf := bytes.NewBuffer(make([]byte, 0, 1024))
	return &inMemRwc{buf, buf}
}

func (rwc *inMemRwc) Close() error {
	return nil
}

func NewTestFrontendStream() (*MessageStream, io.ReadWriteCloser) {
	rwc := NewInMemRwc()
	return NewFrontendMessageStream(rwc), rwc
}

func NewTestBackendStream() (*MessageStream, io.ReadWriteCloser) {
	rwc := NewInMemRwc()
	return NewBackendMessageStream(rwc), rwc
}
