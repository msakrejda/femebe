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

func newTestFrontendStream(t *testing.T) (*femebe.MessageStream, *inMemRwc) {
	rwc := NewInMemRwc()
	return femebe.NewFrontendMessageStream("TestClientConn", rwc), rwc
}

func newTestBackendStream(t *testing.T) (*femebe.MessageStream, *inMemRwc) {
	rwc := NewInMemRwc()
	return femebe.NewBackendMessageStream("TestServerConn", rwc), rwc
}
