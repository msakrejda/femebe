package femebe

import (
	"bytes"
	"io"
	"testing"
)

func newTestMessageStream(t *testing.T) *MessageStream {
	buf := bytes.NewBuffer(make([]byte, 0, 1024))
	ms := MessageStream{
		Name:         "Test",
		state:        CONN_NORMAL,
		msgRemainder: *buf,
		be:           &binEnc{},
	}

	return &ms
}

func TestFullyBuffered(t *testing.T) {
	ms := newTestMessageStream(t)
	var m Message

	InitAuthenticationOk(&m)

	const NUM_MSG = 10

	// Buffer some messages.
	for i := 0; i < NUM_MSG; i += 1 {
		m.WriteTo(&ms.msgRemainder)
	}

	// Read them back out
	for i := 0; i < NUM_MSG-1; i += 1 {
		if !ms.HasNext() {
			t.Fatal()
		}

		ms.Next(&m)
	}

	// The very last HasNext call must return false
	if ms.HasNext() {
		t.Fatal()
	}
}

func TestPromise(t *testing.T) {
	// Write a complete AuthenticationOk message to a buffer.
	var m Message
	InitAuthenticationOk(&m)

	buf := bytes.NewBuffer(make([]byte, 0, 1024))
	m.WriteTo(buf)

	// Slice it apart: the five-byte prefix is a minimal message
	// header.
	header := make([]byte, 5)
	buf.Read(header)

	// Conjure a MessageStream where only those five bytes have
	// been received from the network.
	ms := newTestMessageStream(t)
	ms.msgRemainder.Write(header)

	// 'buf' is in the right spot to form the rest of the
	// Promise-style message.
	ms.r = buf

	// 5 bytes is enough to have a next message.
	if !ms.HasNext() {
		t.Fatal()
	}

	// Grab the message and check the contents of the payload.
	ms.Next(&m)

	checkBuf := bytes.NewBuffer(make([]byte, 0, 1024))
	io.Copy(checkBuf, m.Payload())

	if !bytes.Equal(checkBuf.Bytes(), []byte{0x0, 0x0, 0x0, 0x0}) {
		t.Fatal()
	}
}

func TestIncompleteMessage(t *testing.T) {
	// Write a complete AuthenticationOk message to a buffer.
	var m Message
	InitAuthenticationOk(&m)

	buf := bytes.NewBuffer(make([]byte, 0, 1024))
	m.WriteTo(buf)

	// Slice it apart: a four-byte prefix is not enough to form a
	// complete message.
	header := make([]byte, 4)
	buf.Read(header)

	ms := newTestMessageStream(t)
	ms.msgRemainder.Write(header)

	ms.r = buf

	// Only four bytes are in the MessageStream's buffer, and
	// that's not enough to form a Message without blocking.
	if ms.HasNext() {
		t.Fatal()
	}

	// However, a-priori this test ensures there are enough bytes
	// in the future stream to produce a message.
	ms.Next(&m)

	// Grab the message and check the contents of the payload.
	checkBuf := bytes.NewBuffer(make([]byte, 0, 1024))
	io.Copy(checkBuf, m.Payload())

	if !bytes.Equal(checkBuf.Bytes(), []byte{0x0, 0x0, 0x0, 0x0}) {
		t.Fatal()
	}
}
