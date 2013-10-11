package proto

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
	cms := femebe.NewFrontendStream(rwc)
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

func TestStartupSerDes(t *testing.T) {
	ms, _ := femebe.NewTestFrontendStream()
	var m femebe.Message
	params := make(map[string]string)

	params["hello"] = "world"
	params["goodbye"] = "world"
	params["glory"] = "spite"

	InitStartupMessage(&m, params)

	ms.Send(&m)

	var deserM femebe.Message
	ms.Next(&deserM)

	serBytes, _ := m.Force()
	deserBytes, _ := deserM.Force()
	if !bytes.Equal(serBytes, deserBytes) {
		t.Fatal()
	}
}

func TestBackendKeyReading(t *testing.T) {
	buf := bytes.Buffer{}
	const Pid = 1234
	const Key = 5768
	femebe.WriteInt32(&buf, Pid)
	femebe.WriteInt32(&buf, Key)

	var m femebe.Message
	m.InitFromBytes(MsgBackendKeyDataK, buf.Bytes())

	kd, err := ReadBackendKeyData(&m)
	if err != nil {
		t.Fatal()
	}

	if kd.BackendPid != Pid {
		t.Fatal()
	}

	if kd.SecretKey != Key {
		t.Fatal()
	}
}
