package pgproto

import (
	"bytes"
	"femebe"
	"io"
	"testing"
)

type inMemRwc struct {
	io.ReadWriter
	Contents *bytes.Buffer
}

func NewInMemRwc() *inMemRwc {
	buf := bytes.NewBuffer(make([]byte, 0, 1024))
	return &inMemRwc{buf, buf}
}

func (rwc *inMemRwc) Close() error {
	return nil
}

func newTestClientStream(t *testing.T) (*femebe.MessageStream, *inMemRwc) {
	rwc := NewInMemRwc()
	return femebe.NewClientMessageStream("TestClientConn", rwc), rwc
}

func newTestServerStream(t *testing.T) (*femebe.MessageStream, *inMemRwc) {
	rwc := NewInMemRwc()
	return femebe.NewServerMessageStream("TestServerConn", rwc), rwc
}
