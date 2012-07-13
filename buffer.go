package femebe

import (
	"bytes"
	"encoding/binary"
	"io"
)

func WriteInt16(w io.Writer, val int16) (n int, err error) {
	var be [2]byte
	valBytes := be[0:2]
	binary.BigEndian.PutUint16(valBytes, uint16(val))
	return w.Write(valBytes)
}

func WriteInt32(w io.Writer, val int32) (n int, err error) {
	var be [4]byte
	valBytes := be[0:4]
	binary.BigEndian.PutUint32(valBytes, uint32(val))

	return w.Write(valBytes)
}

func WriteCString(w io.Writer, val string) (n int, err error) {
	n, err = w.Write([]byte(val))
	if err != nil {
		return n, err
	}
	_, err = w.Write([]byte{'\000'})
	return n + 1, err
}

func ReadInt16(r io.Reader) (int16, error) {
	var be [2]byte
	valBytes := be[0:2]
	if _, err := io.ReadFull(r, valBytes); err != nil {
		return 0, err
	}

	return int16(binary.BigEndian.Uint16(valBytes)), nil
}

func ReadUint16(r io.Reader) (uint16, error) {
	var be [2]byte
	valBytes := be[0:2]
	if _, err := io.ReadFull(r, valBytes); err != nil {
		return 0, err
	}

	return uint16(binary.BigEndian.Uint16(valBytes)), nil
}

func ReadInt32(r io.Reader) (int32, error) {
	var be [4]byte
	valBytes := be[0:4]
	if _, err := io.ReadFull(r, valBytes); err != nil {
		return 0, err
	}

	return int32(binary.BigEndian.Uint32(valBytes)), nil
}

func ReadUint32(r io.Reader) (ret uint32, err error) {
	var be [4]byte
	valBytes := be[0:4]
	if _, err = io.ReadFull(r, valBytes); err != nil {
		return 0, err
	}

	return binary.BigEndian.Uint32(valBytes), nil
}

func ReadUint32FromBuffer(r *bytes.Buffer) uint32 {
	var be [4]byte
	valBytes := be[0:4]
	r.Read(valBytes)

	return binary.BigEndian.Uint32(valBytes)
}

func ReadCString(r io.Reader) (s string, err error) {
	var be [1]byte
	charBuf := be[0:1]

	var accum bytes.Buffer

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

func ReadByte(r io.Reader) (ret byte, err error) {
	var be [1]byte
	valBytes := be[0:1]

	if _, err = io.ReadFull(r, valBytes); err != nil {
		return 0, err
	}

	return valBytes[0], nil
}
