package femebe

import (
	"bytes"
	"testing"
)

func TestFullyBuffered(t *testing.T) {
	buf := bytes.NewBuffer(make([]byte, 0, 1024))
	ms := MessageStream{
		Name:         "Test",
		state:        CONN_NORMAL,
		msgRemainder: *buf,
		be:           &binEnc{},
	}

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
			t.FailNow()
		}

		ms.Next(&m)
	}

	// The very last HasNext call must return false
	if ms.HasNext() {
		t.FailNow()
	}
}
