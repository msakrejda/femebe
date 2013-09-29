// Operators to handle first-packet interactions with a client.  That
// includes:
//
// * SSL Negotiation Requests
// * Startup Packet
// * Cancellation Requests
//
// Startup can be re-done after an SSL Negotiation request, and this
// can be modelled by making a TLS connection and then creating a new
// MessageStream from femebe on the plaintext output of that.
//
// Copyright (c) 2012, Heroku.  All rights reserved.
package pgproto

import (
	"github.com/deafbybeheading/femebe"
	e "github.com/deafbybeheading/femebe/error"
)

type Startup struct {
	Params map[string]string
}

func ReadStartupMessage(m *femebe.Message) (*Startup, error) {
	var err error

	if remainingSz := m.Size() - 4; remainingSz > 10000 {
		// Startup packets longer than this are considered
		// invalid.  Copied from the PostgreSQL source code.
		err = e.TooBig(
			"Rejecting oversized startup packet: got %v",
			m.Size())
		return nil, err
	} else if remainingSz < 4 {
		// We expect all initialization messages to
		// have at least a 4-byte header
		err = e.WrongSize(
			"Expected message of at least 4 bytes; got %v",
			remainingSz)
		return nil, err
	}

	body, err := m.Force()
	if err != nil {
		return nil, err
	}

	var b femebe.Reader
	b.InitReader(body)
	protoVer, _ := femebe.ReadInt32(&b)

	const SupportedProtover = 0x00030000
	if protoVer != SupportedProtover {
		err = e.StartupVersion("bad version: got %x expected %x",
				protoVer, SupportedProtover)
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
		return nil, e.StartupFmt("malformed startup packet")
	}

	return &Startup{params}, nil
}
