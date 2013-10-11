package core

import (
	"bytes"
	"github.com/deafbybeheading/femebe/buf"
	"github.com/deafbybeheading/femebe/util"
	"io"
)

// A duplex stream of FEBE messages
type Stream interface {
	util.Flusher
	// Send the Message m on the stream, returning any error
	// encountered
	Send(m *Message) error
	// Check whether another message is available to read from the
	// stream without blocking
	HasNext() bool
	// Receive the next message, loading it into m. If this
	// returns an error, the contents of m are undefined.
	Next(m *Message) error
	Close() error
}

// The minimum number of bytes required to make a new hybridMsg when
// calling Next().  If buffering and less than MSG_HEADER_SIZE remain
// in the buffer, the remaining bytes must be saved for the next
// invocation of Next().
const MsgHeaderMinSize = 5

// State of the stream connection
type ConnState int32

const (
	ConnStartup ConnState = iota
	ConnNormal
	ConnErr
)

type MessageStream struct {
	rw    io.ReadWriteCloser
	state ConnState
	err   error

	// Incomplete message headers that should be chained into
	// message parsing with the subsequent .Next() invocation.
	msgRemainder bytes.Buffer

	// To avoid allocation in inner loops
	scratchBuf [8192]byte
}

func baseNewMessageStream(rw io.ReadWriteCloser, state ConnState) *MessageStream {
	buf := bytes.NewBuffer(make([]byte, 0, 8192))

	return &MessageStream{
		rw:           rw,
		msgRemainder: *buf,
		state:        state,
	}
}

// Create a new MessageStream for managing messages coming from a FEBE
// frontend (e.g., psql). The resulting message stream owns the
// ReadWriteCloser and the caller should not interact with the wrapped
// object directly.
func NewFrontendStream(rw io.ReadWriteCloser) *MessageStream {
	return baseNewMessageStream(rw, ConnStartup)
}

// Create a new MessageStream for managing messages comfing from a
// FEBE backend (e.g., Postgres). The resulting message stream owns
// the ReadWriteCloser and the caller should not interact with the
// wrapped object directly.
func NewBackendStream(rw io.ReadWriteCloser) *MessageStream {
	return baseNewMessageStream(rw, ConnNormal)
}

func (c *MessageStream) HasNext() bool {
	return c.msgRemainder.Len() >= MsgHeaderMinSize
}

func (c *MessageStream) Next(dst *Message) (err error) {
	switch c.state {
	case ConnStartup:
		msgSz, err := buf.ReadUint32(c.rw)
		if err != nil {
			c.err = err
			c.state = ConnErr
			return err
		}

		dst.InitPromise(MsgTypeFirst, msgSz, []byte{}, c.rw)
		c.state = ConnNormal
		return nil

	case ConnNormal:
	again:
		// Fast-path: if a message can be formed from the
		// buffer, do so immediately.
		if c.HasNext() {
			msgType := c.msgRemainder.Next(1)[0]
			msgSz := buf.ReadUint32FromBuffer(&c.msgRemainder)

			remainingSz := msgSz - 4

			if remainingSz > uint32(c.msgRemainder.Len()) {
				// Handle messages that are only
				// partially buffered by creating a
				// Promise-mesage that hybridizes the
				// already-buffered data and the
				// network.
				//
				// Copy bytes in the buffer into new
				// memory as it is about to be
				// recycled, which would cause corrupt
				// state.
				trailing := make([]byte, c.msgRemainder.Len())
				c.msgRemainder.Read(trailing)
				dst.InitPromise(msgType, msgSz,
					trailing, c.rw)
				return nil
			} else {
				// The whole message is in the buffer.
				// Address it by-reference rather than
				// copying it.
				dst.InitFromBytes(msgType,
					c.msgRemainder.Next(int(remainingSz)))
				return nil
			}
		}

		// No more deliverable messages are buffered and an
		// error has been set in a previous iteration:
		// transition to CONN_ERR.
		if !c.HasNext() && c.err != nil {
			c.state = ConnErr
			return c.err
		}

		// Slow-path: need to grab a chunk of bytes from the
		// kernel, so get as many as feasible, but do insist
		// on least enough to form another message header
		// unless the underlying Reader returns with an error.
		for !c.HasNext() {
			newBytes := c.scratchBuf[:]
			n, err := c.rw.Read(newBytes)

			// NB: errors from writing to the buffer is
			// ignored, because msgRemainder is a
			// bytes.Buffer and per specification it will
			// not ever fail to Write the full set of
			// bytes.  Beware if one is changing the type
			// of c.msgRemainder.
			c.msgRemainder.Write(newBytes[0:n])

			// Don't fail immediately, because a few valid
			// messages may have been received in addition
			// to an error.
			if err != nil {
				c.err = err
				goto again
			}
		}

		// The buffer should be full enough to at least
		// deliver a Promise style message, so just try again.
		goto again

	case ConnErr:
		return c.err

	default:
		panic("Oh snap")
	}

	panic("Oh snap")
}

func (c *MessageStream) Send(msg *Message) (err error) {
	_, err = msg.WriteTo(c.rw)
	return err
}

func (c *MessageStream) Flush() error {
	if flushable, ok := c.rw.(util.Flusher); ok {
		return flushable.Flush()
	}

	return nil
}

func (c *MessageStream) Close() error {
	return c.rw.Close()
}
