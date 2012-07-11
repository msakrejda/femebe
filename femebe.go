package femebe

import (
	"bytes"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"reflect"
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

func baseNewMessageStream(config *Config, rw io.ReadWriteCloser) *MessageStream {
	buf := bytes.NewBuffer(make([]byte, 0, 1024))

	return &MessageStream{
		Name:         config.Name,
		rw:           rw,
		msgRemainder: *buf,
		be:           &binEnc{},
	}
}

type Config struct {
	Name      string
	Sslmode   string
	TlsConfig *tls.Config
}

func NewClientMessageStream(config *Config, rw io.ReadWriteCloser) *MessageStream {
	c := baseNewMessageStream(config, rw)
	c.state = CONN_STARTUP

	return c
}

func NewServerMessageStream(config *Config, rw io.ReadWriteCloser) (*MessageStream, error) {
	if config.Sslmode != "disable" {
		// TODO: this is a little ugly; TLS negotiation is (rather sensibly)
		// only available on Conns rather than arbitrary ReadWriteClosers
		conn, ok := rw.(net.Conn)

		if !ok && config.Sslmode != "allow" &&
			config.Sslmode != "prefer" {
			return nil, fmt.Errorf("Cannot establish required SSL connection on %v",
				reflect.TypeOf(rw))
		}
		// send an SSLRequest message
		// length: int32(8)
		// code:   int32(80877103)
		rw.Write([]byte{0x00, 0x00, 0x00, 0x08,
			0x04, 0xd2, 0x16, 0x2f})

		sslResponse := make([]byte, 1)
		bytesRead, err := io.ReadFull(rw, sslResponse)
		if bytesRead != 1 || err != nil {
			return nil, errors.New("Could not read response to SSL Request")
		}

		if sslResponse[0] == 'S' {
			rw = tls.Client(conn, config.TlsConfig)
		} else if sslResponse[0] == 'N' && config.Sslmode != "allow" &&
			config.Sslmode != "prefer" {
			// reject; we require ssl
			return nil, errors.New("SSL required but declined by server.")
		} else {
			// do nothing; proceed with normal startup
		}
	}

	c := baseNewMessageStream(config, rw)
	c.state = CONN_NORMAL

	return c, nil
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
			panic(errors.New("Rejecting oversized startup packet"))
		}
		if remainingSz < 4 {
			// We expect all initialization messages to
			// have at least a 4-byte header
			panic(fmt.Errorf("Expected message of at least 4 bytes; got %v",
				remainingSz))
		}
		headerBytes := make([]byte, 4)
		headerRead, err := io.ReadFull(c.rw, headerBytes)
		var headerType byte
		if headerRead != 4 {
			panic(fmt.Errorf("Could not read startup message header;"+
				" expected 4 bytes, only got %v", headerRead))
		}
		if bytes.HasPrefix(headerBytes, []byte{0x00, 0x03, 0x00, 0x00}) {
			headerType = MSG_STARTUP_MESSAGE
		} else if bytes.HasPrefix(headerBytes, []byte{0x04, 0xd2, 0x16, 0x2e}) {
			headerType = MSG_CANCEL_REQUEST
		} else {
			panic(fmt.Errorf("Unexpected startup message header '%v'",
				headerBytes))
		}

		hr := bytes.NewReader(headerBytes)
		startupReader := io.MultiReader(hr, c.rw)
		InitFullyBufferedMsg(dst, headerType, msgSz)
		_, err = io.CopyN(&dst.buffered, startupReader, int64(remainingSz))
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
