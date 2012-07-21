package pgproto

import (
	"bytes"
	"femebe"
	"testing"
)

func TestBackendKeyReading(t *testing.T) {
	buf := bytes.Buffer{}
	const PID = 1234
	const KEY = 5768
	femebe.WriteInt32(&buf, PID)
	femebe.WriteInt32(&buf, KEY)

	var m femebe.Message
	m.InitFromBytes(MSG_BACKEND_KEY_DATA_K, buf.Bytes())

	kd, err := ReadBackendKeyData(&m)
	if err != nil {
		t.Fatal()
	}

	if kd.pid != PID {
		t.Fatal()
	}

	if kd.key != KEY {
		t.Fatal()
	}
}
