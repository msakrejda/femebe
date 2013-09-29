package message

import (
	"bytes"
	"github.com/deafbybeheading/femebe"
	"testing"
)

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

	if kd.Pid != Pid {
		t.Fatal()
	}

	if kd.Key != Key {
		t.Fatal()
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
