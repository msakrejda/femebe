package femebe

import (
	"bytes"
	"crypto/tls"
	"errors"
	"io"
	"net"
)

type Flusher interface {
	Flush() error
}

// The minimum number of bytes required to make a new hybridMsg when
// calling Next().  If buffering and less than MSG_HEADER_SIZE remain
// in the buffer, the remaining bytes must be saved for the next
// invocation of Next().
const MSG_HEADER_MIN_SIZE = 5

func baseNewMessageStream(name string, rw io.ReadWriteCloser) *MessageStream {
	buf := bytes.NewBuffer(make([]byte, 0, 8192))

	return &MessageStream{
		Name:         name,
		rw:           rw,
		msgRemainder: *buf,
	}
}

type Config struct {
	Name    string
	Sslmode string
}

func NegotiateTLS(c net.Conn, sslmode string, config *tls.Config) (
	net.Conn, error) {
	if sslmode != "disable" {
		// send an SSLRequest message
		// length: int32(8)
		// code:   int32(80877103)
		c.Write([]byte{0x00, 0x00, 0x00, 0x08,
			0x04, 0xd2, 0x16, 0x2f})

		sslResponse := make([]byte, 1)
		bytesRead, err := io.ReadFull(c, sslResponse)
		if bytesRead != 1 || err != nil {
			return nil, errors.New("Could not read response to SSL Request")
		}

		if sslResponse[0] == 'S' {
			return tls.Client(c, config), nil
		} else if sslResponse[0] == 'N' && sslmode != "allow" &&
			sslmode != "prefer" {
			// reject; we require ssl
			return nil, errors.New("SSL required but declined by server.")
		} else {
			return c, nil
		}

		panic("Oh snap!")
	}

	return c, nil
}

func NewClientMessageStream(name string, rw io.ReadWriteCloser) *MessageStream {
	c := baseNewMessageStream(name, rw)
	c.state = CONN_STARTUP

	return c
}

func NewServerMessageStream(name string, rw io.ReadWriteCloser) *MessageStream {
	c := baseNewMessageStream(name, rw)
	c.state = CONN_NORMAL

	return c
}

type ConnState int32

const (
	CONN_STARTUP ConnState = iota
	CONN_NORMAL
	CONN_ERR
)

type MessageStream struct {
	Name  string
	rw    io.ReadWriteCloser
	state ConnState
	err   error

	// Incomplete message headers that should be chained into
	// message parsing with the subsequent .Next() invocation.
	msgRemainder bytes.Buffer

	// To avoid allocation in inner loops
	scratchBuf [8192]byte
}

func (c *MessageStream) HasNext() bool {
	return c.msgRemainder.Len() >= MSG_HEADER_MIN_SIZE
}

func (c *MessageStream) Next(dst *Message) error {
	switch c.state {
	case CONN_STARTUP:
		msgSz, err := ReadUint32(c.rw)
		if err != nil {
			c.err = err
			c.state = CONN_ERR
			return err
		}

		dst.InitPromise(MSG_TYPE_FIRST, msgSz, []byte{}, c.rw)
		c.state = CONN_NORMAL
		return nil

	case CONN_NORMAL:
	again:
		// Fast-path: if a message can be formed from the
		// buffer, do so immediately.
		if c.HasNext() {
			msgType := c.msgRemainder.Next(1)[0]
			msgSz := ReadUint32FromBuffer(&c.msgRemainder)

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
			c.state = CONN_ERR
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

	case CONN_ERR:
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
	if flushable, ok := c.rw.(Flusher); ok {
		return flushable.Flush()
	}

	return nil
}
