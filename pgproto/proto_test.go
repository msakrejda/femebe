package pgproto

import (
	"bytes"
	"femebe"
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
