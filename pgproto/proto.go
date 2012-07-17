package pgproto

import (
	"bytes"
	. "femebe"
	"fmt"
	"reflect"
)

func IsReadyForQuery(msg *Message) bool {
	return msg.MsgType() == MSG_READY_FOR_QUERY_Z
}

func InitReadyForQuery(m *Message, connState ConnStatus) {
	if connState != RFQ_IDLE &&
		connState != RFQ_INTRANS &&
		connState != RFQ_ERROR {
		panic(fmt.Errorf("Invalid message type %v", connState))
	}

	m.InitFromBytes(MSG_READY_FOR_QUERY_Z, []byte{byte(connState)})
}

func NewField(name string, typOid uint32) *FieldDescription {
	typSize := TypSize(typOid)
	return &FieldDescription{name, 0, 0, typOid, typSize, -1, ENC_FMT_TEXT}
}

func InitRowDescription(m *Message, fields []FieldDescription) {
	// use a heuristic estimate for length to avoid having to
	// resize the msgBytes array
	fieldLenEst := (10+4+2+4+2+4+2)
	msgBytes := make([]byte, 0, len(fields) * fieldLenEst)
	buf := bytes.NewBuffer(msgBytes)
	WriteInt16(buf, int16(len(fields)))
	for _, field := range fields {
		WriteCString(buf, field.name)
		WriteInt32(buf, field.tableOid)
		WriteInt16(buf, field.tableAttNo)
		WriteUint32(buf, field.typeOid)
		WriteInt16(buf, field.typLen)
		WriteInt32(buf, field.atttypmod)
		WriteInt16(buf, int16(field.format))
	}

	m.InitFromBytes(MSG_ROW_DESCRIPTION_T, buf.Bytes())
}

// InitDataRow initializes the Message m as a DataRow message with
// data from the value array cols.
func InitDataRow(m *Message, cols []interface{}) {
	msgBytes := make([]byte, 0, 2+len(cols)*4)
	buf := bytes.NewBuffer(msgBytes)
	colCount := int16(len(cols))
	WriteInt16(buf, colCount)
	for _, val := range cols {
		// TODO: allow format specification
		encodeValue(buf, val, ENC_FMT_TEXT)
	}

	m.InitFromBytes(MSG_DATA_ROW_D, buf.Bytes())
}

func InitCommandComplete(m *Message, cmdTag string) {
	msgBytes := make([]byte, 0, len([]byte(cmdTag)))
	buf := bytes.NewBuffer(msgBytes)
	WriteCString(buf, cmdTag)

	m.InitFromBytes(MSG_COMMAND_COMPLETE_C, buf.Bytes())
}

func InitQuery(m *Message, query string) {
	msgBytes := make([]byte, 0, len([]byte(query))+1)
	buf := bytes.NewBuffer(msgBytes)
	WriteCString(buf, query)
	m.InitFromBytes(MSG_QUERY_Q, buf.Bytes())
}

type Query struct {
	Query string
}

func IsQuery(msg *Message) bool {
	return msg.MsgType() == 'Q'
}

func ReadQuery(msg *Message) (*Query, error) {
	qs, err := ReadCString(msg.Payload())
	if err != nil {
		return nil, err
	}

	return &Query{Query: qs}, err
}

type FieldDescription struct {
	name       string
	tableOid   int32
	tableAttNo int16
	typeOid    uint32
	typLen     int16
	atttypmod  int32
	format     EncFmt
}

func encodeValue(buff *bytes.Buffer, val interface{},
	format EncFmt) (err error) {
	if format == ENC_FMT_TEXT {
		switch val.(type) {
		case int16:
			TextEncodeInt16(buff, val.(int16))
		case int32:
			TextEncodeInt32(buff, val.(int32))
		case int64:
			TextEncodeInt64(buff, val.(int64))
		case float32:
			TextEncodeFloat32(buff, val.(float32))
		case float64:
			TextEncodeFloat64(buff, val.(float64))
		case string:
			TextEncodeString(buff, val.(string))
		case bool:
			TextEncodeBool(buff, val.(bool))
		default:
			return fmt.Errorf("Can't encode value: %#q:%#q\n",
				reflect.TypeOf(val), val)
		}
	} else {
		return fmt.Errorf("Can't encode in format %v")
	}
	return nil
}

type RowDescription struct {
	fields []FieldDescription
}

func ReadRowDescription(msg *Message) (
	rd *RowDescription, err error) {
	if msg.MsgType() != MSG_ROW_DESCRIPTION_T {
		panic("Oh snap")
	}
	b := msg.Payload()
	fieldCount, err := ReadUint16(b)
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
		typeOid, err := ReadUint32(b)
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

func InitAuthenticationOk(m *Message) {
	m.InitFromBytes(MSG_AUTHENTICATION_OK_R, []byte{0, 0, 0, 0})
}

// FEBE Message type constants shamelessly stolen from the pq library.
//
// All the constants in this file have a special naming convention:
// "msg(NameInManual)(characterCode)".  This results in long and
// awkward constant names, but also makes it easy to determine what
// the author's intent is quickly in code (consider that both
// msgDescribeD and msgDataRowD appear on the wire as 'D') as well as
// debugging against captured wire protocol traffic (where one will
// only see 'D', but has a sense what state the protocol is in).

type EncFmt int16

const (
	ENC_FMT_TEXT    EncFmt = 0
	ENC_FMT_BINARY         = 1
	ENC_FMT_UNKNOWN        = 0
)

// Special sub-message coding for Close and Describe
const (
	IS_PORTAL = 'P'
	IS_STMT   = 'S'
)

// Sub-message character coding that is part of ReadyForQuery
type ConnStatus byte

const (
	RFQ_IDLE    ConnStatus = 'I'
	RFQ_INTRANS            = 'T'
	RFQ_ERROR              = 'E'
)

// Message tags
const (
	MSG_AUTHENTICATION_OK_R                 byte = 'R'
	MSG_AUTHENTICATION_CLEARTEXT_PASSWORD_R      = 'R'
	MSG_AUTHENTICATION_M_D5_PASSWORD_R           = 'R'
	MSG_AUTHENTICATION_S_C_M_CREDENTIAL_R        = 'R'
	MSG_AUTHENTICATION_G_S_S_R                   = 'R'
	MSG_AUTHENTICATION_S_S_P_I_R                 = 'R'
	MSG_AUTHENTICATION_G_S_S_CONTINUE_R          = 'R'
	MSG_BACKEND_KEY_DATA_K                       = 'K'
	MSG_BIND_B                                   = 'B'
	MSG_BIND_COMPLETE2                           = '2'
	MSG_CANCEL_REQUEST                           = 129 // see below
	MSG_CLOSE_C                                  = 'C'
	MSG_CLOSE_COMPLETE3                          = '3'
	MSG_COMMAND_COMPLETE_C                       = 'C'
	MSG_COPY_DATAD                               = 'd'
	MSG_COPY_DONEC                               = 'c'
	MSG_COPY_FAILF                               = 'f'
	MSG_COPY_IN_RESPONSE_G                       = 'G'
	MSG_COPY_OUT_RESPONSE_H                      = 'H'
	MSG_COPY_BOTH_RESPONSE_W                     = 'W'
	MSG_DATA_ROW_D                               = 'D'
	MSG_DESCRIBE_D                               = 'D'
	MSG_EMPTY_QUERY_RESPONSE_I                   = 'I'
	MSG_ERROR_RESPONSE_E                         = 'E'
	MSG_EXECUTE_E                                = 'E'
	MSG_FLUSH_H                                  = 'H'
	MSG_FUNCTION_CALL_F                          = 'F'
	MSG_FUNCTION_CALL_RESPONSE_V                 = 'V'
	MSG_NO_DATAN                                 = 'n'
	MSG_NOTICE_RESPONSE_N                        = 'N'
	MSG_NOTIFICATION_RESPONSE_A                  = 'A'
	MSG_PARAMETER_DESCRIPTIONT                   = 't'
	MSG_PARAMETER_STATUS_S                       = 'S'
	MSG_PARSE_P                                  = 'P'
	MSG_PARSE_COMPLETE1                          = '1'
	MSG_PASSWORD_MESSAGEP                        = 'p'
	MSG_PORTAL_SUSPENDEDS                        = 's'
	MSG_QUERY_Q                                  = 'Q'
	MSG_READY_FOR_QUERY_Z                        = 'Z'
	MSG_ROW_DESCRIPTION_T                        = 'T'
	// We treat SSLRequest as a protocol negotiation mechanic
	// rather than a first-class message, so it does not appear
	// here

	// StartupMessage and CancelRequest formatted differently:
	// on the wire, they do not have a formal message type, so
	// we use the top bit of these 8-bit bytes to flag these
	// with distinct message types. This is a pretty ugly hack,
	// but allows us to treat the messages uniformly throughout
	// most of the system
	MSG_STARTUP_MESSAGE = 128
	MSG_SYNC_S          = 'S'
	MSG_TERMINATE_X     = 'X'
)
