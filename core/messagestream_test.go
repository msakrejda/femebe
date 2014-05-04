package core

import (
	"bytes"
	"io"
	"testing"
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

func InitBogon(m *Message) {
	m.InitFromBytes('B', []byte{0, 0, 0, 0})
}

func newTestMessageStream(t *testing.T) *MessageStream {
	buf := bytes.NewBuffer(make([]byte, 0, 1024))
	ms := MessageStream{
		state:        ConnNormal,
		msgRemainder: *buf,
	}

	return &ms
}

func TestFullyBuffered(t *testing.T) {
	ms := newTestMessageStream(t)
	var m Message
	InitBogon(&m)

	const NUM_MSG = 10

	// Buffer some messages.
	for i := 0; i < NUM_MSG; i += 1 {
		m.WriteTo(&ms.msgRemainder)
	}

	// Read them back out
	for i := 0; i < NUM_MSG; i += 1 {
		if !ms.HasNext() {
			t.Fail()
		}

		ms.Next(&m)
	}

	// The very last HasNext call must return false
	if ms.HasNext() {
		t.Fail()
	}
}

func TestPromise(t *testing.T) {
	var m Message
	InitBogon(&m)

	buf := newClosableBuffer(bytes.NewBuffer(make([]byte, 0, 1024)))
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
	ms.rw = buf

	// 5 bytes is enough to have a next message.
	if !ms.HasNext() {
		t.Fail()
	}

	// Grab the message and check the contents of the payload.
	ms.Next(&m)

	checkBuf := bytes.NewBuffer(make([]byte, 0, 1024))
	io.Copy(checkBuf, m.Payload())

	if !bytes.Equal(checkBuf.Bytes(), []byte{0x0, 0x0, 0x0, 0x0}) {
		t.Fail()
	}

	// More attempts to read from that stream should result in EOF
	for i := 0; i < 5; i += 1 {
		err := ms.Next(&m)
		if err != io.EOF {
			t.Fail()
		}
	}
}

func TestIncompleteMessage(t *testing.T) {
	var m Message
	InitBogon(&m)

	buf := newClosableBuffer(bytes.NewBuffer(make([]byte, 0, 1024)))
	m.WriteTo(buf)

	// Slice it apart: a four-byte prefix is not enough to form a
	// complete message.
	header := make([]byte, 4)
	buf.Read(header)

	ms := newTestMessageStream(t)
	ms.msgRemainder.Write(header)

	ms.rw = buf

	// Only four bytes are in the MessageStream's buffer, and
	// that's not enough to form a Message without blocking.
	if ms.HasNext() {
		t.Fail()
	}

	// However, a-priori this test ensures there are enough bytes
	// in the future stream to produce a message.
	ms.Next(&m)

	// Grab the message and check the contents of the payload.
	checkBuf := bytes.NewBuffer(make([]byte, 0, 1024))
	io.Copy(checkBuf, m.Payload())

	if !bytes.Equal(checkBuf.Bytes(), []byte{0x0, 0x0, 0x0, 0x0}) {
		t.Fail()
	}

	// More attempts to read from that stream should result in EOF
	for i := 0; i < 5; i += 1 {
		err := ms.Next(&m)
		if err != io.EOF {
			t.Fail()
		}
	}
}
