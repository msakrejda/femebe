package codec

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"github.com/uhoh-itsmaciek/femebe/buf"
	"github.com/uhoh-itsmaciek/femebe/proto"
	"log"
	"strconv"
	"time"
)

func EncodeValue(buff *bytes.Buffer, val interface{}, format proto.EncFmt) (err error) {
	if format == proto.EncFmtTxt {
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
			return fmt.Errorf("Can't encode value %#v of type %T", val, val)
		}
	} else {
		return fmt.Errorf("Can't encode in format %v", format)
	}
	return nil
}

func BinEncodeInt16(buff *bytes.Buffer, val int16) {
	buf.WriteInt32(buff, 2)
	buf.WriteInt16(buff, val)
}

func TextEncodeInt16(buff *bytes.Buffer, val int16) {
	encodeValText(buff, val, "%d")
}

func TextEncodeInt32(buff *bytes.Buffer, val int32) {
	encodeValText(buff, val, "%d")
}

func TextEncodeInt64(buff *bytes.Buffer, val int64) {
	encodeValText(buff, val, "%d")
}

func TextEncodeFloat32(buff *bytes.Buffer, val float32) {
	encodeValText(buff, val, "%e")
}

func TextEncodeFloat64(buff *bytes.Buffer, val float64) {
	encodeValText(buff, val, "%e")
}

func TextEncodeString(buff *bytes.Buffer, val string) {
	encodeValText(buff, val, "%s")
}

func TextEncodeBool(buff *bytes.Buffer, val bool) {
	encodeValText(buff, val, "%t")
}

func encodeValText(b *bytes.Buffer,
	val interface{}, format string) {
	result := fmt.Sprintf(format, val)
	buf.WriteInt32(b, int32(len([]byte(result))))
	b.WriteString(result)
}

// Decode Postgres (text) encoding into a reasonably corresponding Go
// type (lifted from pq)
func Decode(s []byte, typ proto.Oid) interface{} {
	switch typ {
	case proto.OidText, proto.OidVarchar:
		return string(s)
	case proto.OidBytea:
		// N.B.: assumes hex bytea output
		s = s[2:] // trim off "\\x"
		d := make([]byte, hex.DecodedLen(len(s)))
		_, err := hex.Decode(d, s)
		if err != nil {
			log.Fatalf("femebe: %s", err)
		}
		return d
	case proto.OidTimestamp:
		return mustParse("2006-01-02 15:04:05", typ, s)
	case proto.OidTimestamptz:
		return mustParse("2006-01-02 15:04:05-07", typ, s)
	case proto.OidTime:
		return mustParse("15:04:05", typ, s)
	case proto.OidTimetz:
		return mustParse("15:04:05-07", typ, s)
	case proto.OidDate:
		return mustParse("2006-01-02", typ, s)
	case proto.OidBool:
		return s[0] == 't'
	case proto.OidInt8, proto.OidInt4, proto.OidInt2:
		i, err := strconv.ParseInt(string(s), 10, 64)
		if err != nil {
			log.Fatalf("femebe: %s", err)
		}
		return i
	case proto.OidFloat4, proto.OidFloat8:
		var bits int
		if typ == proto.OidFloat4 {
			bits = 32
		} else {
			bits = 64
		}
		f, err := strconv.ParseFloat(string(s), bits)
		if err != nil {
			log.Fatalf("femebe: %s", err)
		}
		return f
	default:
		return s
	}
}

func mustParse(f string, typ proto.Oid, s []byte) time.Time {
	str := string(s)

	// Special case until time.Parse bug is fixed:
	// http://code.google.com/p/go/issues/detail?id=3487
	if str[len(str)-2] == '.' {
		str += "0"
	}

	// check for a 30-minute-offset timezone
	if (typ == proto.OidTimestamptz || typ == proto.OidTimetz) &&
		str[len(str)-3] == ':' {
		f += ":00"
	}
	t, err := time.Parse(f, str)
	if err != nil {
		log.Fatalf("femebe: decode: %s", err)
	}
	return t
}

// Describe which Go type this Postgres OID will map to in the scheme
// above
func DescribeType(typ proto.Oid) string {
	switch typ {
	case proto.OidText, proto.OidVarchar:
		return "string"
	case proto.OidBytea:
		return "[]byte"
	case proto.OidTimestamp, proto.OidTimestamptz, proto.OidTime, proto.OidTimetz, proto.OidDate:
		return "time.Time"
	case proto.OidBool:
		return "boolean"
	case proto.OidInt8, proto.OidInt4, proto.OidInt2:
		return "int64"
	case proto.OidFloat4, proto.OidFloat8:
		return "float64"
	default:
		return "unknown"
	}
}
