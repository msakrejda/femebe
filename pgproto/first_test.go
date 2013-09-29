package pgproto

import (
	"bytes"
	"github.com/deafbybeheading/femebe"
	e "github.com/deafbybeheading/femebe/error"
	"testing"
)

// A helper that initializes a message, writes it into and then then
// reads it back out of femebe.
func firstMessageRoundTrip(t *testing.T,
	init func(m *femebe.Message)) (*femebe.Message, error) {
	// Pretend that a bad startup packet is being serialized
	// and sent to the server.
	sms, rwc := femebe.NewTestBackendStream()
	var m femebe.Message
	init(&m)
	sms.Send(&m)

	// Reuse the buffer that has been filled and pretend to be
	// serving a client connection isntead, which should result in
	// an error because the startup message is over-sized.
	cms := femebe.NewFrontendMessageStream(rwc)
	if err := cms.Next(&m); err != nil {
		return nil, err
	}

	return &m, nil
}

func TestHugeStartup(t *testing.T) {
	init := func(m *femebe.Message) {
		m.InitPromise(femebe.MsgTypeFirst, 10005, []byte{},
			&bytes.Buffer{})
	}

	m, err := firstMessageRoundTrip(t, init)
	if err != nil {
		t.Fatal()
	}

	_, err = ReadStartupMessage(m)
	if _, ok := err.(e.ErrTooBig); ok {
		// This is expected
	} else {
		t.Fatalf("Got error %#v, and it is not expected", err)
	}
}

func TestSmallStartup(t *testing.T) {
	init := func(m *femebe.Message) {
		m.InitPromise(femebe.MsgTypeFirst, 7, []byte{},
			&bytes.Buffer{})
	}

	m, err := firstMessageRoundTrip(t, init)
	if err != nil {
		t.Fatal()
	}

	_, err = ReadStartupMessage(m)
	if _, ok := err.(e.ErrWrongSize); ok {
		// This is expected
	} else {
		t.Fatalf("Got error %#v, and it is not expected", err)
	}
}
