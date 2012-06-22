package femebe

import (
	"fmt"
	"bytes"
)

type Message struct {
	msgType byte
	payload []byte
}

func NewAuthenticationOk() *Message {
	return &Message{'R', []byte{0, 0, 0, 0}}
}

func IsAuthenticationOk(msg *Message) bool {
	return msg.msgType == 'R' &&
		msg.payload[0] == 0 &&
		msg.payload[1] == 0 &&
		msg.payload[2] == 0 &&
		msg.payload[3] == 0
}

// TODO: all the other auth types

const (
	RFQ_IDLE = 'I'
	RFQ_IN_TRANS = 'T'
	RFQ_ERROR = 'E'
)

type ConnStatus byte

func NewReadyForQuery(connState ConnStatus) (*Message, error) {
	if connState != RFQ_IDLE && connState != RFQ_IN_TRANS && connState != RFQ_ERROR {
		return nil, fmt.Errorf("Invalid message type %v", connState)
	}
	return &Message{'Z', []byte{byte(connState)}}, nil
}

func IsReadyForQuery(msg *Message) bool {
	return msg.msgType == 'Z'
}

type FieldDescription struct {
	name string
	tableOid int32
	tableAttNo int16
	typeOid int32
	typLen int16
	atttypmod int32
	format int16
}

const (
	ENC_FMT_TEXT = 0
	ENC_FMT_BINARY = 1
	ENC_FMT_UNKNOWN = 0
)

type EncFmt int16

const (
	INT16 = iota
	INT32
	INT64
	FLOAT32
	FLOAT64
	STRING
	BOOL
)

type PGType int16

func NewField(name string, dataType PGType) (*FieldDescription) {
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

func NewRowDescription(fields []FieldDescription) *Message {
	msgBytes := make([]byte, 0, len(fields) * (10 + 4 + 2 + 4 + 2 + 4 + 2))
	buff := bytes.NewBuffer(msgBytes)
	WriteInt16(buff, int16(len(fields)))
	for _, field := range fields {
		WriteCString(buff, field.name)
		WriteInt32(buff, field.tableOid)
		WriteInt16(buff, field.tableAttNo)
		WriteInt32(buff, field.typeOid)
		WriteInt16(buff, field.typLen)
		WriteInt32(buff, field.atttypmod)
		WriteInt16(buff, field.format)
	}
	return &Message{'T', buff.Bytes()}
}

func NewDataRow(cols []interface{}) *Message {
	msgBytes := make([]byte, 0, 2 + len(cols) * 4)
	buff := bytes.NewBuffer(msgBytes)
	colCount := int16(len(cols))
	WriteInt16(buff, colCount)
	fmt.Printf("making data message with %v columns", colCount)
	for _, val := range cols {
		// TODO: allow format specification
		encodeValue(buff, val, ENC_FMT_TEXT)
	}
	return &Message{'D', buff.Bytes()}
}

func NewCommandComplete(cmdTag string) *Message {
	msgBytes := make([]byte, 0, len([]byte(cmdTag)))
	buff := bytes.NewBuffer(msgBytes)
	WriteCString(buff, cmdTag)
	return &Message{'C', buff.Bytes()}
}

func NewQuery(query string) *Message {
	msgBytes := make([]byte, 0, len([]byte(query)))
	buff := bytes.NewBuffer(msgBytes)
	WriteCString(buff, query)
	return &Message{'Q', buff.Bytes()}
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
	fields[] FieldDescription
}

func ReadRowDescription(msg *Message) *RowDescription {
	if msg.msgType != 'T' {
		panic("Oh snap")
	}
	b := bytes.NewBuffer(msg.payload)
	fieldCount := ReadUInt16(b)
	fields := make([]FieldDescription, fieldCount)
	for i, _ := range fields {
		name := ReadCString(b)
		tableOid := ReadInt32(b)
		tableAttNo := ReadInt16(b)
		typeOid := ReadInt32(b)
		typLen := ReadInt16(b)
		atttypmod := ReadInt32(b)
		format := ReadInt16(b)
		fields[i] = FieldDescription{name, tableOid, tableAttNo,
			typeOid, typLen, atttypmod, format}
	}
	return &RowDescription{fields}
}

type StartupMessage struct {
	params map[string]string	
}

func ReadStartupMessage(msg *Message) *StartupMessage {
	if msg.msgType != '\000' {
		panic("Oh snap")
	}
	msgLen := len(msg.payload)
	b := bytes.NewBuffer(msg.payload)
	protoVer := ReadInt32(b)
	if protoVer != 0x00030000 {
		panic("Oh snap! Unrecognized protocol version number")
	}
	params := make(map[string]string)
	for remaining := msgLen - 4; remaining > 1; {
		key := ReadCString(b)
		val := ReadCString(b)
		remaining -= len(key) + len(val) + 2 /* null bytes */
		params[key] = val
	}
	terminator, _ := b.ReadByte()
	if terminator != '\000' {
		panic("Oh snap! WTF byte is this?")
	}
	return &StartupMessage{params}
}
