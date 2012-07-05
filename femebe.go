package femebe

import (
	"encoding/binary"
	"errors"
	"io"
)

type MessageStream interface {
	Send(m Message) (err error)
	Next() (m Message, err error)
}

// The minimum number of bytes required to make a new hybridMsg when
// calling Next().  If buffering and less than MSG_HEADER_SIZE remain
// in the buffer, the remaining bytes must be saved for the next
// invocation of Next().
const MSG_HEADER_MIN_SIZE = 5

// Startup packets longer than this are considered invalid.  Copied
// from the PostgreSQL source code.
const MAX_STARTUP_PACKET_LENGTH = 10000

func NewMessageStreamIngress(name string, r io.Reader, w io.Writer) MessageStream {
	return &ctxt{Name: name, r: r, w: w, state: CONN_STARTUP}
}

func NewMessageStreamEgress(name string, r io.Reader, w io.Writer) MessageStream {
	return &ctxt{Name: name, r: r, w: w, state: CONN_NORMAL}
}

type ConnState int32

const (
	CONN_STARTUP ConnState = iota
	CONN_NORMAL
	CONN_ERR
)

type ctxt struct {
	Name  string
	r     io.Reader
	w     io.Writer
	state ConnState
}

func (c *ctxt) Next() (msg Message, err error) {
	defer func() {
		recovered := recover()
		if e, ok := recovered.(error); ok {
			msg = nil
			err = e
		} else if recovered != nil {
			// This wasn't an error, so it may be a string
			// suggesting something is *really* wrong
			// (e.g. an assertion failure).  It probably
			// says "Oh snap".
			panic(recovered)
		}
	}()

	panicNonNil := func(err error) {
		if err != nil {
			panic(err)
		}
	}

	switch c.state {
	case CONN_STARTUP:
		msgSz, err := ReadUInt32(c.r)
		panicNonNil(err)
		msgSz -= 4

		if msgSz > MAX_STARTUP_PACKET_LENGTH {
			panic(errors.New("rejecting oversized startup packet"))
		}

		payload := make([]byte, msgSz)
		_, err = io.ReadFull(c.r, payload)
		panicNonNil(err)

		c.state = CONN_NORMAL

		return NewFullyBufferedMsg('\000', payload), nil

	case CONN_NORMAL:
		msgType, err := ReadByte(c.r)
		panicNonNil(err)

		msgSz, err := ReadUInt32(c.r)
		panicNonNil(err)
		msgSz -= 4

		payload := make([]byte, msgSz)
		_, err = io.ReadFull(c.r, payload)
		panicNonNil(err)

		return NewFullyBufferedMsg(msgType, payload), nil

	case CONN_ERR:
		return nil, errors.New("MessageStream in error state")

	default:
		panic("Oh snap")
	}

	panic("Oh snap")
}

func (c *ctxt) Send(msg Message) (err error) {
	if msg.MsgType() != '\000' {
		err = binary.Write(c.w, binary.BigEndian, msg.MsgType())
		if err != nil {
			return err
		}
	}

	err = binary.Write(c.w, binary.BigEndian, msg.Size())
	if err != nil {
		return err
	}

	if _, err := io.Copy(c.w, msg.Payload()); err != nil {
		return err
	}

	return err
}
