package proto

import (
	"bytes"
	"github.com/uhoh-itsmaciek/femebe/buf"
	"github.com/uhoh-itsmaciek/femebe/core"
	e "github.com/uhoh-itsmaciek/femebe/error"
	"io"
	"testing"
)

// A helper that initializes a message, writes it into and then then
// reads it back out of core.
func firstMessageRoundTrip(t *testing.T,
	init func(m *core.Message)) (*core.Message, error) {
	// Pretend that a bad startup packet is being serialized
	// and sent to the server.
	rwc := newInMemRwc()
	sms := core.NewBackendStream(rwc)

	var m core.Message
	init(&m)
	sms.Send(&m)

	// Reuse the buffer that has been filled and pretend to be
	// serving a client connection isntead, which should result in
	// an error because the startup message is over-sized.
	cms := core.NewFrontendStream(rwc)
	if err := cms.Next(&m); err != nil {
		return nil, err
	}

	return &m, nil
}

func TestHugeStartup(t *testing.T) {
	init := func(m *core.Message) {
		m.InitPromise(core.MsgTypeFirst, 10005, []byte{},
			&bytes.Buffer{})
	}

	m, err := firstMessageRoundTrip(t, init)
	if err != nil {
		t.Fail()
	}

	_, err = ReadStartupMessage(m)
	if _, ok := err.(e.ErrTooBig); ok {
		// This is expected
	} else {
		t.Fatalf("Got error %#v, and it is not expected", err)
	}
}

func TestSmallStartup(t *testing.T) {
	init := func(m *core.Message) {
		m.InitPromise(core.MsgTypeFirst, 7, []byte{},
			&bytes.Buffer{})
	}

	m, err := firstMessageRoundTrip(t, init)
	if err != nil {
		t.Fail()
	}

	_, err = ReadStartupMessage(m)
	if _, ok := err.(e.ErrWrongSize); ok {
		// This is expected
	} else {
		t.Fatalf("Got error %#v, and it is not expected", err)
	}
}

func TestStartupSerDes(t *testing.T) {
	ms := core.NewFrontendStream(newInMemRwc())
	var m core.Message
	params := make(map[string]string)

	params["hello"] = "world"
	params["goodbye"] = "world"
	params["glory"] = "spite"

	InitStartupMessage(&m, params)

	ms.Send(&m)

	var deserM core.Message
	ms.Next(&deserM)

	serBytes, _ := m.Force()
	deserBytes, _ := deserM.Force()
	if !bytes.Equal(serBytes, deserBytes) {
		t.Fail()
	}
}

func TestBackendKeyReading(t *testing.T) {
	b := bytes.Buffer{}
	const Pid = 1234
	const Key = 5768
	buf.WriteInt32(&b, Pid)
	buf.WriteInt32(&b, Key)

	var m core.Message
	m.InitFromBytes(MsgBackendKeyDataK, b.Bytes())

	kd, err := ReadBackendKeyData(&m)
	if err != nil {
		t.Fail()
	}

	if kd.BackendPid != Pid {
		t.Fail()
	}

	if kd.SecretKey != Key {
		t.Fail()
	}
}

// utility types and functions for these tests
type inMemRwc struct {
	io.ReadWriter
	Contents *bytes.Buffer
}

func newInMemRwc() io.ReadWriteCloser {
	buf := bytes.NewBuffer(make([]byte, 0, 1024))
	return &inMemRwc{buf, buf}
}

func (rwc *inMemRwc) Close() error {
	return nil
}
