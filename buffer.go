package femebe

import (
	"bytes"
	"encoding/binary"
)



func WriteInt16(b *bytes.Buffer, val int16) {
	valBytes := make([]byte, 2)
	binary.BigEndian.PutUint16(valBytes, uint16(val))
	b.Write(valBytes)
}

func WriteInt32(b *bytes.Buffer, val int32) {
	valBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(valBytes, uint32(val))
	b.Write(valBytes)
}

func WriteCString(b *bytes.Buffer, val string) { 
	b.WriteString(val)
	b.WriteByte('\000')
}

func ReadInt16(b *bytes.Buffer) int16 {
	valBytes := make([]byte, 2)
	b.Read(valBytes)
	return int16(binary.BigEndian.Uint16(valBytes))
}

func ReadUInt16(b *bytes.Buffer) uint16 {
	valBytes := make([]byte, 2)
	b.Read(valBytes)
	return uint16(binary.BigEndian.Uint16(valBytes))
}

func ReadInt32(b *bytes.Buffer) int32 {
	valBytes := make([]byte, 4)
	b.Read(valBytes)
	return int32(binary.BigEndian.Uint32(valBytes))
}

func ReadUInt32(b *bytes.Buffer) uint32 {
	valBytes := make([]byte, 4)
	b.Read(valBytes)
	return uint32(binary.BigEndian.Uint32(valBytes))
}

func ReadCString(b *bytes.Buffer) string {
	line, _ := b.ReadBytes('\000')
	lineLen := len(line)
	return string(line[0:lineLen-1])
}
