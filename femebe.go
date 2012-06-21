package x

import (
	"encoding/binary"
	"io"
)

type MessageStream interface {
	Send(m *Message) (err error)
	Next() (m *Message, err error)
}

func NewMessageStream(name string, r io.Reader, w io.Writer) MessageStream {
	return &ctxt{Name: name, r: r,  w: w, state: CONN_BEGIN}
}

type ConnState int32

const (
	CONN_BEGIN ConnState = iota
	CONN_ERR
	CONN_CONNECTED
)

type ctxt struct {
	Name string
	r io.Reader
	w io.Writer
	state ConnState
}

func (c *ctxt) Next() (msg *Message, err error) {
	// N.B.: We intend this to block before we check the state for
	// very specific reasons: if Next() is called before Send() in
	// "client" mode, and we want to acknowledge that transition
	// before processing the message
	msgHeader := make([]byte, 5)
	var msgType byte
	var payload []byte
	var size uint32
	_, err = io.ReadFull(c.r, msgHeader)

	if err != nil {
		c.state = CONN_ERR
		return nil, err
	}

	if c.state == CONN_BEGIN {
		msgType = '\000'
		size = uint32(binary.BigEndian.Uint32(msgHeader[0:4])) - 4
		payload = make([]byte, size)
		payload[0] = msgHeader[4]
		io.ReadFull(c.r, payload[1:])

		c.state = CONN_CONNECTED
	} else {
		msgType = msgHeader[0]
		size = uint32(binary.BigEndian.Uint32(msgHeader[1:])) - 4
		payload = make([]byte, size)
		io.ReadFull(c.r, payload)
	}
	if err != nil {
		c.state = CONN_ERR
		return nil, err
	}

	return &Message{msgType: msgType, payload: payload}, err
}

func (c *ctxt) Send(m *Message) (err error) {
	if c.state == CONN_BEGIN {
		c.state = CONN_CONNECTED
	}
	msgLen := int32(4 + len(m.payload))
	if m.msgType != '\000' {
		err = binary.Write(c.w, binary.BigEndian, m.msgType)
		if err != nil {
			return err
		}
	}

	err = binary.Write(c.w, binary.BigEndian, msgLen)
	if err != nil {
		return err
	}
	// TODO: think about big writes
	_, err = c.w.Write(m.payload)
	if err != nil {
		return err
	}
	return err
}

