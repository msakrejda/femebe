package pgproto

import (
	"bytes"
	. "femebe"
	"fmt"
	"io"
	"regexp"
	"strconv"
)

type ErrBadTypeCode struct {
	error
}

type ErrTooBig struct {
	error
}

type ErrWrongSize struct {
	error
}

func InitReadyForQuery(m *Message, connState ConnStatus) error {
	if connState != RfqIdle &&
		connState != RfqInTrans &&
		connState != RfqError {
		return ErrBadTypeCode{
			fmt.Errorf("Invalid message type %v", connState)}
	}

	m.InitFromBytes(MsgReadyForQueryZ, []byte{byte(connState)})
	return nil
}

func NewField(name string, typOid Oid) *FieldDescription {
	typSize := TypSize(typOid)
	return &FieldDescription{name, 0, 0, typOid, typSize, -1, EncFmtTxt}
}

func InitRowDescription(m *Message, fields []FieldDescription) {
	// use a heuristic estimate for length to avoid having to
	// resize the msgBytes array
	fieldLenEst := (10 + 4 + 2 + 4 + 2 + 4 + 2)
	msgBytes := make([]byte, 0, len(fields)*fieldLenEst)
	buf := bytes.NewBuffer(msgBytes)
	WriteInt16(buf, int16(len(fields)))
	for _, field := range fields {
		WriteCString(buf, field.Name)
		WriteUint32(buf, uint32(field.TableOid))
		WriteInt16(buf, field.TableAttNo)
		WriteUint32(buf, uint32(field.TypeOid))
		WriteInt16(buf, field.TypLen)
		WriteInt32(buf, field.Atttypmod)
		WriteInt16(buf, int16(field.Format))
	}

	m.InitFromBytes(MsgRowDescriptionT, buf.Bytes())
}

func InitDataRow(m *Message, encodedData [][]byte) {
	dataSize := 0
	for _, colVal := range encodedData {
		dataSize += len(colVal)
	}
	msgBytes := make([]byte, 0, 2+dataSize)
	buf := bytes.NewBuffer(msgBytes)
	colCount := int16(len(encodedData))
	WriteInt16(buf, colCount)
	for _, colVal := range encodedData {
		buf.Write(colVal)
	}

	m.InitFromBytes(MsgDataRowD, buf.Bytes())
}

func InitCommandComplete(m *Message, cmdTag string) {
	msgBytes := make([]byte, 0, len([]byte(cmdTag)))
	buf := bytes.NewBuffer(msgBytes)
	WriteCString(buf, cmdTag)

	m.InitFromBytes(MsgCommandCompleteC, buf.Bytes())
}

func InitQuery(m *Message, query string) {
	msgBytes := make([]byte, 0, len([]byte(query))+1)
	buf := bytes.NewBuffer(msgBytes)
	WriteCString(buf, query)
	m.InitFromBytes(MsgQueryQ, buf.Bytes())
}

type Query struct {
	Query string
}

func ReadQuery(msg *Message) (*Query, error) {
	qs, err := ReadCString(msg.Payload())
	if err != nil {
		return nil, err
	}

	return &Query{Query: qs}, err
}

type FieldDescription struct {
	Name       string
	TableOid   Oid
	TableAttNo int16
	TypeOid    Oid
	TypLen     int16
	Atttypmod  int32
	Format     EncFmt
}

type RowDescription struct {
	Fields []FieldDescription
}

func ReadRowDescription(msg *Message) (
	rd *RowDescription, err error) {
	if msg.MsgType() != MsgRowDescriptionT {
		return nil, ErrBadTypeCode{
			fmt.Errorf("Invalid message type %v", msg.MsgType())}
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

		fields[i] = FieldDescription{name, Oid(tableOid), tableAttNo,
			Oid(typeOid), typLen, atttypmod, EncFmt(format)}
	}

	return &RowDescription{fields}, nil
}

type DataRow struct {
	Values [][]byte
}

func ReadDataRow(m *Message) (*DataRow, error) {
	if m.MsgType() != MsgDataRowD {
		return nil, ErrBadTypeCode{
			fmt.Errorf("Invalid message type %v", m.MsgType())}
	}
	b := m.Payload()
	fieldCount, err := ReadUint16(b)
	if err != nil {
		return nil, err
	}

	values := make([][]byte, fieldCount)

	for i := range values {
		fieldLen, err := ReadInt32(b)
		if err != nil {
			return nil, err
		}
		if fieldLen >= 0 {
			fieldData := make([]byte, fieldLen)
			io.ReadFull(b, fieldData)
			values[i] = fieldData
		} else if fieldLen == -1 {
			values[i] = nil
		} else {
			return nil, ErrWrongSize{
				fmt.Errorf("Invalid length %v for field %v",
					fieldLen, i)}
		}
	}
	return &DataRow{values}, nil
}

type CommandComplete struct {
	Tag           string
	AffectedCount uint64
	Oid           uint32
}

func ReadCommandComplete(m *Message) (*CommandComplete, error) {
	if m.MsgType() != MsgCommandCompleteC {
		return nil, ErrBadTypeCode{
			fmt.Errorf("Invalid message type %v", m.MsgType())}
	}

	p := m.Payload()
	fullTag, err := ReadCString(p)
	if err != nil {
		return nil, err
	}

	cmdRe := regexp.MustCompile("(INSERT|DELETE|UPDATE|SELECT|MOVE|FETCH|COPY) (\\d+)(?: (\\d+))?")
	if match := cmdRe.FindStringSubmatch(fullTag); match != nil {
		var rowcountIdx int
		var rowcount uint64
		var oid uint32

		hasOid := len(match) == 4 && match[3] != ""
		tag := match[1]

		if hasOid {
			val, err := strconv.ParseUint(match[2], 10, 32)
			if err != nil {
				panic("Oh snap")
			}
			oid = uint32(val)
			rowcountIdx = 3
		} else {
			rowcountIdx = 2
			oid = 0
		}

		rowcount, err := strconv.ParseUint(match[rowcountIdx], 10, 64)
		if err != nil {
			panic("Oh snap")
		}

		return &CommandComplete{tag, rowcount, oid}, nil
	} else {
		return &CommandComplete{fullTag, 0, 0}, nil
	}

	panic("Oh snap")
}

type ErrorResponse struct {
	Details map[byte]string
}

func ReadErrorResponse(msg *Message) (*ErrorResponse, error) {
	if msg.MsgType() != MsgErrorResponseE {
		return nil, ErrBadTypeCode{
			fmt.Errorf("Invalid message type %v", msg.MsgType())}
	}

	p := msg.Payload()
	details := make(map[byte]string)
	for {
		fieldCode, err := ReadByte(p)
		if err != nil {
			return nil, err
		}
		if fieldCode == 0 {
			break
		}
		fieldValue, err := ReadCString(p)
		if err != nil {
			return nil, err
		}
		details[fieldCode] = fieldValue
	}
	return &ErrorResponse{details}, nil
}

func InitAuthenticationOk(m *Message) {
	m.InitFromBytes(MsgAuthenticationOkR, []byte{0, 0, 0, 0})
}

type BackendKeyData struct {
	Pid int32
	Key int32
}

func ReadBackendKeyData(msg *Message) (*BackendKeyData, error) {
	if msg.MsgType() != MsgBackendKeyDataK {
		return nil, ErrBadTypeCode{
			fmt.Errorf("Invalid message type %v", msg.MsgType())}
	}

	const RIGHT_SZ = 12
	if msg.Size() != RIGHT_SZ {
		return nil, ErrWrongSize{
			fmt.Errorf("BackendKeyData is wrong size: "+
				"expected %v, got %v", RIGHT_SZ, msg.Size())}
	}

	r := msg.Payload()
	pid, err := ReadInt32(r)
	if err != nil {
		return nil, err
	}

	key, err := ReadInt32(r)
	if err != nil {
		return nil, err
	}

	return &BackendKeyData{Pid: pid, Key: key}, err
}

// Logical names of various ErrorResponse and NoticeResponse keys,
// taken from
// http://www.postgresql.org/docs/current/static/protocol-error-fields.html
func DescribeStatusCode(code byte) string {
	switch code {
	// The field contents are ERROR, FATAL, or PANIC (in an error
	// message), or WARNING, NOTICE, DEBUG, INFO, or LOG (in a
	// notice message), or a localized translation of one of
	// these. Always present.
	case 'S':
		return "Severity"
		// The SQLSTATE code for the error (see Appendix A at
		// http://www.postgresql.org/docs/current/static/errcodes-appendix.html
		// ). Not localizable. Always present.
	case 'C':
		return "Code"
		// The primary human-readable error message. This should be
		// accurate but terse (typically one line). Always present.
	case 'M':
		return "Message"
		// An optional secondary error message carrying more detail
		// about the problem. Might run to multiple lines.
	case 'D':
		return "Detail"
		// An optional suggestion what to do about the problem. This
		// is intended to differ from Detail in that it offers advice
		// (potentially inappropriate) rather than hard facts. Might
		// run to multiple lines.
	case 'H':
		return "Hint"
		// The field value is a decimal ASCII integer, indicating an
		// error cursor position as an index into the original query
		// string. The first character has index 1, and positions are
		// measured in characters not bytes.
	case 'P':
		return "Position"
		// This is defined the same as the P field, but it is used
		// when the cursor position refers to an internally generated
		// command rather than the one submitted by the client. The q
		// field will always appear when this field appears.
	case 'p':
		return "Internal position"
		// The text of a failed internally-generated command. This
		// could be, for example, a SQL query issued by a PL/pgSQL
		// function.
	case 'q':
		return "Internal query"
		// An indication of the context in which the error
		// occurred. Presently this includes a call stack traceback of
		// active procedural language functions and
		// internally-generated queries. The trace is one entry per
		// line, most recent first.
	case 'W':
		return "Where"
		// The file name of the source-code location where the error was reported.
	case 'F':
		return "File"
		// the line number of the source-code location where the error was reported.
	case 'L':
		return "Line"
		// the name of the source-code routine reporting the error.
	case 'R':
		return "Routine"
	default:
		return fmt.Sprintf("[unknown: %v]", code)
	}
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
	EncFmtTxt     EncFmt = 0
	EncFmtBinary         = 1
	EncFmtUnknown        = 0
)

// Special sub-message coding for Close and Describe
const (
	IsPortal = 'P'
	IsStmt   = 'S'
)

// Sub-message character coding that is part of ReadyForQuery
type ConnStatus byte

const (
	RfqIdle    ConnStatus = 'I'
	RfqInTrans            = 'T'
	RfqError              = 'E'
)

// Message tags
const (
	MsgAuthenticationOkR                byte = 'R'
	MsgAuthenticationCleartextPasswordR      = 'R'
	MsgAuthenticationMD5PasswordR            = 'R'
	MsgAuthenticationSCMCredentialR          = 'R'
	MsgAuthenticationGSSR                    = 'R'
	MsgAuthenticationSSPIR                   = 'R'
	MsgAuthenticationGSSContinueR            = 'R'
	MsgBackendKeyDataK                       = 'K'
	MsgBindB                                 = 'B'
	MsgBindComplete2                         = '2'
	MsgCloseC                                = 'C'
	MsgCloseComplete3                        = '3'
	MsgCommandCompleteC                      = 'C'
	MsgCopyDataD                             = 'd'
	MsgCopyDoneC                             = 'c'
	MsgCopyFailF                             = 'f'
	MsgCopyInResponseG                       = 'G'
	MsgCopyOutResponseH                      = 'H'
	MsgCopyBothResponseW                     = 'W'
	MsgDataRowD                              = 'D'
	MsgDescribeD                             = 'D'
	MsgEmptyQueryResponseI                   = 'I'
	MsgErrorResponseE                        = 'E'
	MsgExecuteE                              = 'E'
	MsgFlushH                                = 'H'
	MsgFunctionCallF                         = 'F'
	MsgFunctionCallResponseV                 = 'V'
	MsgNoDataN                               = 'n'
	MsgNoticeResponseN                       = 'N'
	MsgNotificationResponseA                 = 'A'
	MsgParameterDescriptionT                 = 't'
	MsgParameterStatusS                      = 'S'
	MsgParseP                                = 'P'
	MsgParseComplete1                        = '1'
	MsgPasswordMessageP                      = 'p'
	MsgPortalSuspendedS                      = 's'
	MsgQueryQ                                = 'Q'
	MsgReadyForQueryZ                        = 'Z'
	MsgRowDescriptionT                       = 'T'

	// SSLRequest is not seen here because we treat SSLRequest as
	// a protocol negotiation mechanic rather than a first-class
	// message, so it does not appear here

	MsgSyncS      = 'S'
	MsgTerminateX = 'X'
)
