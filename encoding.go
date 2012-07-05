package femebe

import (
	"bytes"
	"fmt"
)

func (be *binEnc) encodeValText(buf *bytes.Buffer,
	val interface{}, format string) {
	result := fmt.Sprintf(format, val)
	be.WriteInt32(buf, int32(len([]byte(result))))
	buf.WriteString(result)
}

func (be *binEnc) EncodeInt16(buff *bytes.Buffer, val int16, format EncFmt) {
	if format == ENC_FMT_TEXT {
		be.encodeValText(buff, val, "%d")
	} else if format == ENC_FMT_BINARY {
		be.WriteInt32(buff, 2)
		be.WriteInt16(buff, val)
	} else {
		panic("Unknown format")
	}
}

func (be *binEnc) EncodeInt32(buff *bytes.Buffer, val int32, format EncFmt) {
	if format == ENC_FMT_TEXT {
		be.encodeValText(buff, val, "%d")
	} else {
		panic("Unknown format")
	}
}

func (be *binEnc) EncodeInt64(buff *bytes.Buffer, val int64, format EncFmt) {
	if format == ENC_FMT_TEXT {
		be.encodeValText(buff, val, "%d")
	} else {
		panic("Unknown format")
	}
}

func (be *binEnc) EncodeFloat32(buff *bytes.Buffer, val float32, format EncFmt) {
	if format == ENC_FMT_TEXT {
		be.encodeValText(buff, val, "%e")
	} else {
		panic("Unknown format")
	}
}

func (be *binEnc) EncodeFloat64(buff *bytes.Buffer, val float64, format EncFmt) {
	if format == ENC_FMT_TEXT {
		be.encodeValText(buff, val, "%e")
	} else {
		panic("Unknown format")
	}
}

func (be *binEnc) EncodeString(buff *bytes.Buffer, val string, format EncFmt) {
	if format == ENC_FMT_TEXT {
		be.encodeValText(buff, val, "%s")
	} else {
		panic("Unknown format")
	}
}

func (be *binEnc) EncodeBool(buff *bytes.Buffer, val bool, format EncFmt) {
	if format == ENC_FMT_TEXT {
		be.encodeValText(buff, val, "%t")
	} else {
		panic("Unknown format")
	}
}
