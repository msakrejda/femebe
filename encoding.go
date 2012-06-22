package femebe

import (
	"fmt"
	"bytes"
)

func encodeValText(buf *bytes.Buffer, val interface{}, format string) {
	result := fmt.Sprintf(format, val)
	WriteInt32(buf, int32(len([]byte(result))))
	buf.WriteString(result)	
}

func EncodeInt16(buff *bytes.Buffer, val int16, format EncFmt) {
	if (format == ENC_FMT_TEXT) {
		encodeValText(buff, val, "%d")
	} else if (format == ENC_FMT_BINARY) {
		WriteInt32(buff, 2)
		WriteInt16(buff, val)
	} else {
		panic("Unknown format")
	}
}
	
func EncodeInt32(buff *bytes.Buffer, val int32, format EncFmt) {
	if (format == ENC_FMT_TEXT) {
		encodeValText(buff, val, "%d")
	} else {
		panic("Unknown format")
	}
}

func EncodeInt64(buff *bytes.Buffer, val int64, format EncFmt) {
	if (format == ENC_FMT_TEXT) {
		encodeValText(buff, val, "%d")
	} else {
		panic("Unknown format")
	}
}

func EncodeFloat32(buff *bytes.Buffer, val float32, format EncFmt) {
	if (format == ENC_FMT_TEXT) {
		encodeValText(buff, val, "%e")
	} else {
		panic("Unknown format")
	}
}

func EncodeFloat64(buff *bytes.Buffer, val float64, format EncFmt) {
	if (format == ENC_FMT_TEXT) {
		encodeValText(buff, val, "%e")
	} else {
		panic("Unknown format")
	}
}

func EncodeString(buff *bytes.Buffer, val string, format EncFmt) {
	if (format == ENC_FMT_TEXT) {
		encodeValText(buff, val, "%s")
	} else {
		panic("Unknown format")
	}
}

func EncodeBool(buff *bytes.Buffer, val bool, format EncFmt) {
	if (format == ENC_FMT_TEXT) {
		encodeValText(buff, val, "%t")
	} else {
		panic("Unknown format")
	}
}
