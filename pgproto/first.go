package pgproto

import (
	"bytes"
	"errors"
	"femebe"
	"fmt"
)

type ErrStartupSmall struct {
	error
}

type ErrStartupBig struct {
	error
}

type ErrStartupVersion struct {
	error
}

type ErrStartupFmt struct {
	error
}

type Startup struct {
	Params map[string]string
}

func IsStartup(m *femebe.Message) (bool, error) {
	body, err := m.Force()
	if err != nil {
		return false, err
	}

	return bytes.HasPrefix(body, []byte{0x00, 0x03, 0x00, 0x00}), nil
}

func ReadStartupMessage(m *femebe.Message) (*Startup, error) {
	var err error

	if remainingSz := m.Size() - 4; remainingSz > 10000 {
		// Startup packets longer than this are considered
		// invalid.  Copied from the PostgreSQL source code.
		err = ErrStartupBig{fmt.Errorf(
			"Rejecting oversized startup packet: got %v",
			m.Size())}
		return nil, err
	} else if remainingSz < 4 {
		// We expect all initialization messages to
		// have at least a 4-byte header
		err = ErrStartupSmall{
			fmt.Errorf(
				"Expected message of at least 4 bytes; got %v",
				remainingSz)}
		return nil, err
	}

	body, err := m.Force()
	if err != nil {
		return nil, err
	}

	var b femebe.Reader
	b.InitReader(body)
	protoVer, _ := femebe.ReadInt32(&b)

	const SUPPORTED_PROTOVER = 0x00030000
	if protoVer != SUPPORTED_PROTOVER {
		err = ErrStartupVersion{
			fmt.Errorf("bad version: got %x expected %x",
				protoVer, SUPPORTED_PROTOVER)}
		return nil, err
	}

	params := make(map[string]string)
	for remaining := b.Len(); remaining > 1; {
		key, err := femebe.ReadCString(&b)
		if err != nil {
			return nil, err
		}

		val, err := femebe.ReadCString(&b)
		if err != nil {
			return nil, err
		}

		remaining -= len(key) + len(val) + 2 /* null bytes */
		params[key] = val
	}

	// Fidelity check on the startup packet, whereby the last byte
	// must be a NUL.
	if d, _ := femebe.ReadByte(&b); d != '\000' {
		return nil, ErrStartupFmt{
			errors.New("malformed startup packet")}
	}

	return &Startup{params}, nil
}

func (s *Startup) FillMessage(m *femebe.Message) {
	buf := bytes.NewBuffer(make([]byte, 0, 1024))

	// Startup-message type word
	buf.Write([]byte{0x00, 0x03, 0x00, 0x00})

	for name, value := range s.Params {
		femebe.WriteCString(buf, name)
		femebe.WriteCString(buf, value)
	}

	buf.Write([]byte{'\000'})

	m.InitFromBytes(femebe.MSG_TYPE_FIRST, buf.Bytes())
}
