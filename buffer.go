package femebe

import (
	"bytes"
	"encoding/binary"
	"io"
)

func WriteInt16(w io.Writer, val int16) {
	valBytes := make([]byte, 2)
	binary.BigEndian.PutUint16(valBytes, uint16(val))
	w.Write(valBytes)
}

func WriteInt32(w io.Writer, val int32) {
	valBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(valBytes, uint32(val))
	w.Write(valBytes)
}

func WriteCString(w io.Writer, val string) {
	io.WriteString(w, val)
	w.Write([]byte{'\000'})
}

func ReadInt16(r io.Reader) int16 {
	valBytes := make([]byte, 2)
	r.Read(valBytes)
	return int16(binary.BigEndian.Uint16(valBytes))
}

func ReadUInt16(r io.Reader) uint16 {
	valBytes := make([]byte, 2)
	r.Read(valBytes)
	return uint16(binary.BigEndian.Uint16(valBytes))
}

func ReadInt32(r io.Reader) int32 {
	valBytes := make([]byte, 4)
	r.Read(valBytes)
	return int32(binary.BigEndian.Uint32(valBytes))
}

func ReadUInt32(r io.Reader) uint32 {
	valBytes := make([]byte, 4)
	r.Read(valBytes)
	return uint32(binary.BigEndian.Uint32(valBytes))
}

func ReadCString(r io.Reader) (s string, err error) {
	var accum bytes.Buffer
	charBuf := make([]byte, 1)

	for {
		n, err := r.Read(charBuf)

		if err != nil {
			return "", err
		}

		// Handle the case of no error, yet no bytes were
		// retrieved.
		if n < 1 {
			continue
		}

		switch charBuf[0] {
		case '\000':
			return string(accum.Bytes()), nil
		default:
			accum.Write(charBuf)
		}
	}

	panic("Oh snap")
}
