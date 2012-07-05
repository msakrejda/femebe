package femebe

import (
	"bytes"
	"io"
)

type Message struct {
	// Constant-width header
	msgType byte
	sz      uint32

	// Tracks the state of the Payload stream's progression
	payloadReader io.Reader

	// Message contents buffered in memory
	buffered bytes.Buffer
}

func (m *Message) MsgType() byte {
	return m.msgType
}

func (m *Message) Payload() io.Reader {
	return m.payloadReader
}

func (m *Message) Size() uint32 {
	return m.sz
}

func InitFullyBufferedMsg(dst *Message, msgType byte, size uint32) {
	dst.msgType = msgType
	dst.sz = size
	dst.buffered.Reset()
	dst.payloadReader = &dst.buffered
}

func InitMsgFromBytes(dst *Message, msgType byte, payload []byte) {
	dst.msgType = msgType

	// Because message size includes the size word itself.
	dst.sz = uint32(len(payload)) + 4
	dst.buffered.Reset()
	dst.buffered.Write(payload)
}

func InitPromiseMsg(dst *Message, msgType byte, size uint32, r io.Reader) {
	dst.msgType = msgType
	dst.sz = size
	dst.payloadReader = r
	dst.buffered = bytes.Buffer{}
}

// TODO: all the other auth types

type ConnStatus byte

const (
	RFQ_IDLE     ConnStatus = 'I'
	RFQ_IN_TRANS            = 'T'
	RFQ_ERROR               = 'E'
)

func IsReadyForQuery(msg Message) bool {
	return msg.MsgType() == 'Z'
}

type FieldDescription struct {
	name       string
	tableOid   int32
	tableAttNo int16
	typeOid    int32
	typLen     int16
	atttypmod  int32
	format     EncFmt
}

type EncFmt int16

const (
	ENC_FMT_TEXT    EncFmt = 0
	ENC_FMT_BINARY         = 1
	ENC_FMT_UNKNOWN        = 0
)

type PGType int16

const (
	INT16 PGType = iota
	INT32
	INT64
	FLOAT32
	FLOAT64
	STRING
	BOOL
)

func (be *binEnc) encodeValue(buff *bytes.Buffer, val interface{},
	format EncFmt) {
	switch val.(type) {
	case int16:
		be.EncodeInt16(buff, val.(int16), format)
	case int32:
		be.EncodeInt32(buff, val.(int32), format)
	case int64:
		be.EncodeInt64(buff, val.(int64), format)
	case float32:
		be.EncodeFloat32(buff, val.(float32), format)
	case float64:
		be.EncodeFloat64(buff, val.(float64), format)
	case string:
		be.EncodeString(buff, val.(string), format)
	case bool:
		be.EncodeBool(buff, val.(bool), format)
	default:
		panic("Can't encode value")
	}
}

type RowDescription struct {
	fields []FieldDescription
}

func (be *binEnc) ReadRowDescription(msg Message) (
	rd *RowDescription, err error) {
	if msg.MsgType() != 'T' {
		panic("Oh snap")
	}
	b := msg.Payload()
	fieldCount, err := be.ReadUInt16(b)
	if err != nil {
		return nil, err
	}

	fields := make([]FieldDescription, fieldCount)
	for i, _ := range fields {
		name, err := be.ReadCString(b)
		if err != nil {
			return nil, err
		}

		tableOid, err := be.ReadInt32(b)
		if err != nil {
			return nil, err
		}
		tableAttNo, err := be.ReadInt16(b)
		if err != nil {
			return nil, err
		}
		typeOid, err := be.ReadInt32(b)
		if err != nil {
			return nil, err
		}
		typLen, err := be.ReadInt16(b)
		if err != nil {
			return nil, err
		}
		atttypmod, err := be.ReadInt32(b)
		if err != nil {
			return nil, err
		}
		format, err := be.ReadInt16(b)
		if err != nil {
			return nil, err
		}

		fields[i] = FieldDescription{name, tableOid, tableAttNo,
			typeOid, typLen, atttypmod, EncFmt(format)}
	}

	return &RowDescription{fields}, nil
}

type StartupMessage struct {
	params map[string]string
}

func (be *binEnc) ReadStartupMessage(msg Message) (
	sm *StartupMessage, err error) {
	if msg.MsgType() != '\000' {
		panic("Oh snap")
	}
	msgLen := msg.Size()
	b := msg.Payload()
	protoVer, err := be.ReadInt32(b)
	if err != nil {
		return nil, err
	}

	if protoVer != 0x00030000 {
		panic("Oh snap! Unrecognized protocol version number")
	}
	params := make(map[string]string)
	for remaining := msgLen - 4; remaining > 1; {
		key, err := be.ReadCString(b)
		if err != nil {
			return nil, err
		}

		val, err := be.ReadCString(b)
		if err != nil {
			return nil, err
		}

		remaining -= uint32(len(key) + len(val) + 2) /* null bytes */
		params[key] = val
	}

	// Fidelity check on the startup packet, whereby the last byte
	// must be a NUL.
	chrBuf := make([]byte, 1)
	_, err = io.ReadAtLeast(b, chrBuf, 1)

	if chrBuf[0] != '\000' {
		panic("Oh snap! WTF byte is this?")
	}

	return &StartupMessage{params}, nil
}
