package femebe

import (
	"encoding/binary"
	"io"
)

// ReadBytes() is defined in a couple of go packages (bufio, bytes)
// but not reified into an interface, so do that here.
type ReadByteser interface {
	ReadBytes(delim byte) (line []byte, err error)
}

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

func ReadCString(r ReadByteser) string {
	line, _ := r.ReadBytes('\000')
	lineLen := len(line)
	return string(line[0 : lineLen-1])
}
