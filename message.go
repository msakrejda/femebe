package femebe

import (
	"bytes"
	"fmt"
	"io"
)

type Message interface {
	MsgType() byte
	Size() uint32
	Payload() io.Reader
}

type hybridMsg struct {
	// Constant-width header
	msgType byte
	sz      uint32

	// Tracks the state of the Payload stream's progression
	payloadReader io.Reader

	// Message contents buffered in memory
	buffered *bytes.Buffer
}

func (m *hybridMsg) MsgType() byte {
	return m.msgType
}

func (m *hybridMsg) Payload() io.Reader {
	return m.payloadReader
}

func (m *hybridMsg) Size() uint32 {
	return m.sz
}

func NewFullyBufferedMsg(msgType byte, payload []byte) Message {
	hm := hybridMsg{
		msgType:  msgType,
		sz:       uint32(4 + len(payload)),
		buffered: bytes.NewBuffer(payload),
	}

	// In this degenerate case, the buffered data is *all* the
	// data, so the payloadReader and buffered content are the
	// same.
	hm.payloadReader = hm.buffered

	return &hm
}

func NewAuthenticationOk() Message {
	return NewFullyBufferedMsg('R', []byte{0, 0, 0, 0})
}

// TODO: all the other auth types

type ConnStatus byte

const (
	RFQ_IDLE     ConnStatus = 'I'
	RFQ_IN_TRANS            = 'T'
	RFQ_ERROR               = 'E'
)

func NewReadyForQuery(connState ConnStatus) (Message, error) {
	if connState != RFQ_IDLE && connState != RFQ_IN_TRANS && connState != RFQ_ERROR {
		return nil, fmt.Errorf("Invalid message type %v", connState)
	}

	return NewFullyBufferedMsg('Z', []byte{byte(connState)}), nil
}

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

func NewField(name string, dataType PGType) *FieldDescription {
	switch dataType {
	case INT16:
		return &FieldDescription{name, 0, 0, 21, 2, -1, ENC_FMT_TEXT}
	case INT32:
		return &FieldDescription{name, 0, 0, 23, 4, -1, ENC_FMT_TEXT}
	case INT64:
		return &FieldDescription{name, 0, 0, 20, 8, -1, ENC_FMT_TEXT}
	case FLOAT32:
		return &FieldDescription{name, 0, 0, 700, 4, -1, ENC_FMT_TEXT}
	case FLOAT64:
		return &FieldDescription{name, 0, 0, 701, 8, -1, ENC_FMT_TEXT}
	case STRING:
		return &FieldDescription{name, 0, 0, 25, -1, -1, ENC_FMT_TEXT}
	case BOOL:
		return &FieldDescription{name, 0, 0, 16, 1, -1, ENC_FMT_TEXT}
	}
	panic("Oh snap")
}

func NewRowDescription(fields []FieldDescription) Message {
	msgBytes := make([]byte, 0, len(fields)*(10+4+2+4+2+4+2))
	buff := bytes.NewBuffer(msgBytes)
	WriteInt16(buff, int16(len(fields)))
	for _, field := range fields {
		WriteCString(buff, field.name)
		WriteInt32(buff, field.tableOid)
		WriteInt16(buff, field.tableAttNo)
		WriteInt32(buff, field.typeOid)
		WriteInt16(buff, field.typLen)
		WriteInt32(buff, field.atttypmod)
		WriteInt16(buff, int16(field.format))
	}
	return NewFullyBufferedMsg('T', buff.Bytes())
}

func NewDataRow(cols []interface{}) Message {
	msgBytes := make([]byte, 0, 2+len(cols)*4)
	buff := bytes.NewBuffer(msgBytes)
	colCount := int16(len(cols))
	WriteInt16(buff, colCount)
	fmt.Printf("making data message with %v columns", colCount)
	for _, val := range cols {
		// TODO: allow format specification
		encodeValue(buff, val, ENC_FMT_TEXT)
	}
	return NewFullyBufferedMsg('D', buff.Bytes())
}

func NewCommandComplete(cmdTag string) Message {
	msgBytes := make([]byte, 0, len([]byte(cmdTag)))
	buff := bytes.NewBuffer(msgBytes)
	WriteCString(buff, cmdTag)
	return NewFullyBufferedMsg('C', buff.Bytes())
}

func NewQuery(query string) Message {
	msgBytes := make([]byte, 0, len([]byte(query)))
	buff := bytes.NewBuffer(msgBytes)
	WriteCString(buff, query)
	return NewFullyBufferedMsg('Q', buff.Bytes())
}

func encodeValue(buff *bytes.Buffer, val interface{}, format EncFmt) {
	switch val.(type) {
	case int16:
		EncodeInt16(buff, val.(int16), format)
	case int32:
		EncodeInt32(buff, val.(int32), format)
	case int64:
		EncodeInt64(buff, val.(int64), format)
	case float32:
		EncodeFloat32(buff, val.(float32), format)
	case float64:
		EncodeFloat64(buff, val.(float64), format)
	case string:
		EncodeString(buff, val.(string), format)
	case bool:
		EncodeBool(buff, val.(bool), format)
	default:
		panic("Can't encode value")
	}
}

type RowDescription struct {
	fields []FieldDescription
}

func ReadRowDescription(msg Message) (rd *RowDescription, err error) {
	if msg.MsgType() != 'T' {
		panic("Oh snap")
	}
	b := msg.Payload()
	fieldCount, err := ReadUInt16(b)
	if err != nil {
		return nil, err
	}

	fields := make([]FieldDescription, fieldCount)
	for i, _ := range fields {
		name, err := ReadCString(b)
		if err != nil {
			return nil, err
		}

		tableOid, err := ReadInt32(b)
		if err != nil {
			return nil, err
		}
		tableAttNo, err := ReadInt16(b)
		if err != nil {
			return nil, err
		}
		typeOid, err := ReadInt32(b)
		if err != nil {
			return nil, err
		}
		typLen, err := ReadInt16(b)
		if err != nil {
			return nil, err
		}
		atttypmod, err := ReadInt32(b)
		if err != nil {
			return nil, err
		}
		format, err := ReadInt16(b)
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

func ReadStartupMessage(msg Message) (sm *StartupMessage, err error) {
	if msg.MsgType() != '\000' {
		panic("Oh snap")
	}
	msgLen := msg.Size()
	b := msg.Payload()
	protoVer, err := ReadInt32(b)
	if err != nil {
		return nil, err
	}

	if protoVer != 0x00030000 {
		panic("Oh snap! Unrecognized protocol version number")
	}
	params := make(map[string]string)
	for remaining := msgLen - 4; remaining > 1; {
		key, err := ReadCString(b)
		if err != nil {
			return nil, err
		}

		val, err := ReadCString(b)
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
