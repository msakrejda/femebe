package femebe

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
