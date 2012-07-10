package femebe

import (
	"bytes"
	"errors"
	"io"
)

type Flusher interface {
	Flush() error
}

// The minimum number of bytes required to make a new hybridMsg when
// calling Next().  If buffering and less than MSG_HEADER_SIZE remain
// in the buffer, the remaining bytes must be saved for the next
// invocation of Next().
const MSG_HEADER_MIN_SIZE = 5

// Startup packets longer than this are considered invalid.  Copied
// from the PostgreSQL source code.
const MAX_STARTUP_PACKET_LENGTH = 10000

func baseNewMessageStream(name string, rw io.ReadWriteCloser) *MessageStream {
	buf := bytes.NewBuffer(make([]byte, 0, 1024))

	return &MessageStream{
		Name:         name,
		rw:           rw,
		msgRemainder: *buf,
		be:           &binEnc{},
	}
}

func NewMessageStreamIngress(name string, rw io.ReadWriteCloser) *MessageStream {
	c := baseNewMessageStream(name, rw)
	c.state = CONN_STARTUP

	return c
}

func NewMessageStreamEgress(name string, rw io.ReadWriteCloser) *MessageStream {
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
	be    *binEnc

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
		msgSz, err := c.be.ReadUint32(c.rw)
		if err != nil {
			c.state = CONN_ERR
			return err
		}

		remainingSz := msgSz - 4

		if remainingSz > MAX_STARTUP_PACKET_LENGTH {
			panic(errors.New("rejecting oversized startup packet"))
		}

		InitFullyBufferedMsg(dst, '\000', msgSz)
		_, err = io.CopyN(&dst.buffered, c.rw, int64(remainingSz))
		if err != nil {
			c.state = CONN_ERR
			return err
		}

		c.state = CONN_NORMAL
		return nil

	case CONN_NORMAL:
	again:
		// Fast-path: if a message can be formed from the
		// buffer, do so immediately.
		if c.HasNext() {
			msgType, err := c.be.ReadByte(&c.msgRemainder)
			if err != nil {
				c.state = CONN_ERR
				return err
			}

			msgSz := c.be.ReadUint32FromBuffer(&c.msgRemainder)
			if err != nil {
				c.state = CONN_ERR
				return err
			}

			remainingSz := msgSz - 4

			if remainingSz > uint32(c.msgRemainder.Len()) {
				// Handle messages that are only
				// partially buffered by creating a
				// Promise-mesage that hybridizes the
				// already-buffered data and the
				// network.
				futureBytes := int64(remainingSz -
					uint32(c.msgRemainder.Len()))
				rest := io.LimitReader(c.rw, futureBytes)
				all := io.MultiReader(&c.msgRemainder, rest)

				InitPromiseMsg(dst, msgType, msgSz, all)
				return nil
			} else {
				// The whole message is in the buffer,
				// so optimize this down to some
				// memory copying, avoiding the need
				// for a more complex Promise-style
				// message.
				InitFullyBufferedMsg(dst, msgType, msgSz)
				_, err = dst.buffered.Write(
					c.msgRemainder.Next(int(remainingSz)))
				if err != nil {
					c.state = CONN_ERR
					return err
				}

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

			// Don't fail immediately, because a few valid
			// messages may have been received in addition
			// to an error.
			if err != nil {
				c.err = err
				goto again
			}

			// NB: errors from writing to the buffer is
			// ignored, because msgRemainder is a
			// bytes.Buffer and per specification it will
			// not ever fail to Write the full set of
			// bytes.  Beware if one is changing the type
			// of c.msgRemainder.
			c.msgRemainder.Write(newBytes[0:n])
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
