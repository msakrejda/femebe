package femebe

import (
	"bytes"
	"encoding/binary"
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

func baseNewMessageStream(name string, r io.Reader, w io.Writer) *MessageStream {
	buf := bytes.NewBuffer(make([]byte, 0, 1024))

	return &MessageStream{
		Name:         name,
		r:            r,
		w:            w,
		msgRemainder: *buf,
		be:           &binEnc{},
	}
}

func NewMessageStreamIngress(
	name string, r io.Reader, w io.Writer) *MessageStream {
	c := baseNewMessageStream(name, r, w)
	c.state = CONN_STARTUP

	return c
}

func NewMessageStreamEgress(
	name string, r io.Reader, w io.Writer) *MessageStream {
	c := baseNewMessageStream(name, r, w)
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
	r     io.Reader
	w     io.Writer
	state ConnState
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

func (c *MessageStream) Next(dst *Message) (err error) {
	defer func() {
		recovered := recover()
		if e, ok := recovered.(error); ok {
			c.state = CONN_ERR
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
		msgSz, err := c.be.ReadUInt32(c.r)
		panicNonNil(err)
		remainingSz := msgSz - 4

		if remainingSz > MAX_STARTUP_PACKET_LENGTH {
			panic(errors.New("rejecting oversized startup packet"))
		}

		InitFullyBufferedMsg(dst, '\000', msgSz)
		_, err = io.CopyN(&dst.buffered, c.r, int64(remainingSz))
		panicNonNil(err)

		c.state = CONN_NORMAL
		return nil

	case CONN_NORMAL:
		hasFuture := true

	again:
		if c.HasNext() {
			msgType, err := c.be.ReadByte(&c.msgRemainder)
			panicNonNil(err)

			msgSz, err := c.be.ReadUInt32(&c.msgRemainder)
			panicNonNil(err)
			remainingSz := msgSz - 4

			if remainingSz > uint32(c.msgRemainder.Len()) {
				// Handle messages that are only
				// partially buffered by creating a
				// Promise-mesage that hybridizes the
				// already-buffered data and the
				// network.
				futureBytes := int64(remainingSz -
					uint32(c.msgRemainder.Len()))
				rest := io.LimitReader(c.r, futureBytes)
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
				panicNonNil(err)
				return nil
			}
		} else if !hasFuture {
			// Can't form even one more message and
			// underlying stream is exhausted, so tell the
			// caller.
			return io.EOF
		}

		// Slow-path: need to grab a chunk of bytes from the
		// kernel, so get as many as feasible, but do insist
		// on least enough to form another message header.
		for c.msgRemainder.Len() < MSG_HEADER_MIN_SIZE {
			newBytes := c.scratchBuf[:]
			n, err := c.r.Read(newBytes)

			// EOF is not known to be a cause for alarm
			// yet: that only can be determined if it's
			// found that this EOF truncates a message.
			if err == io.EOF {
				hasFuture = false
			} else {
				panicNonNil(err)
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
		return errors.New("MessageStream in error state")

	default:
		panic("Oh snap")
	}

	panic("Oh snap")
}

func (c *MessageStream) Send(msg *Message) (err error) {
	b := [4]byte{msg.MsgType()}

	if msg.MsgType() != '\000' {
		if _, err = c.w.Write(b[:1]); err != nil {
			return err
		}
	}

	bs := b[0:4]
	binary.BigEndian.PutUint32(bs, msg.Size())
	if _, err = c.w.Write(bs); err != nil {
		return err
	}

	if _, err := io.Copy(c.w, msg.Payload()); err != nil {
		return err
	}

	return err
}

func (c *MessageStream) Flush() error {
	if flushable, ok := c.w.(Flusher); ok {
		return flushable.Flush()
	}

	return nil
}
