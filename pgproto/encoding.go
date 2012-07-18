package pgproto

import (
	"bytes"
	"femebe"
	"fmt"
)

func encodeValText(buf *bytes.Buffer,
	val interface{}, format string) {
	result := fmt.Sprintf(format, val)
	femebe.WriteInt32(buf, int32(len([]byte(result))))
	buf.WriteString(result)
}

func BinEncodeInt16(buff *bytes.Buffer, val int16) {
	femebe.WriteInt32(buff, 2)
	femebe.WriteInt16(buff, val)
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
