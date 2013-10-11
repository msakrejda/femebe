package buf

import (
	"bytes"
	"testing"
)

func verifyWriteInt16(val int16, expected []byte, t *testing.T) {
	w := new(bytes.Buffer)
	// We don't care about bytes written here because the
	// comparison below is guaranteed to fail if it is not
	// what we expect
	_, err := WriteInt16(w, val)
	if err != nil {
		t.Errorf("Unexpected error in encoding %v", err)
	}
	written := w.Bytes()
	result := bytes.Compare(written, expected)
	if result != 0 {
		t.Errorf("Expected %v; got %v", expected, written)
	}
}

func TestWriteInt16(t *testing.T) {
	verifyWriteInt16(0, []byte{0, 0}, t)
	verifyWriteInt16(1, []byte{0, 1}, t)
	verifyWriteInt16(0xFF, []byte{0, 0xFF}, t)
	verifyWriteInt16(0x7FFF, []byte{0x7F, 0xFF}, t)
	verifyWriteInt16(-1, []byte{0xFF, 0xFF}, t)
}

func verifyWriteInt32(val int32, expected []byte, t *testing.T) {
	w := new(bytes.Buffer)
	_, err := WriteInt32(w, val)
	if err != nil {
		t.Errorf("Unexpected error in encoding %v", err)
	}
	written := w.Bytes()
	result := bytes.Compare(written, expected)
	if result != 0 {
		t.Errorf("Expected %v; got %v", expected, written)
	}
}

func TestWriteInt32(t *testing.T) {
	verifyWriteInt32(0, []byte{0, 0, 0, 0}, t)
	verifyWriteInt32(1, []byte{0, 0, 0, 1}, t)
	verifyWriteInt32(0xFF, []byte{0, 0, 0, 0xFF}, t)
	verifyWriteInt32(0xFFFF, []byte{0, 0, 0xFF, 0xFF}, t)
	verifyWriteInt32(0xFFFFFF, []byte{0, 0xFF, 0xFF, 0xFF}, t)
	verifyWriteInt32(0x7FFFFFFF, []byte{0x7F, 0xFF, 0xFF, 0xFF}, t)
	verifyWriteInt32(-1, []byte{0xFF, 0xFF, 0xFF, 0xFF}, t)
}

func verifyWriteUint32(val uint32, expected []byte, t *testing.T) {
	w := new(bytes.Buffer)
	_, err := WriteUint32(w, val)
	if err != nil {
		t.Errorf("Unexpected error in encoding %v", err)
	}
	written := w.Bytes()
	result := bytes.Compare(written, expected)
	if result != 0 {
		t.Errorf("Expected %v; got %v", expected, written)
	}
}

func TestWriteUint32(t *testing.T) {
	verifyWriteUint32(0, []byte{0, 0, 0, 0}, t)
	verifyWriteUint32(1, []byte{0, 0, 0, 1}, t)
	verifyWriteUint32(0xFF, []byte{0, 0, 0, 0xFF}, t)
	verifyWriteUint32(0xFFFF, []byte{0, 0, 0xFF, 0xFF}, t)
	verifyWriteUint32(0xFFFFFF, []byte{0, 0xFF, 0xFF, 0xFF}, t)
	verifyWriteUint32(0x7FFFFFFF, []byte{0x7F, 0xFF, 0xFF, 0xFF}, t)
	verifyWriteUint32(0xFFFFFFFF, []byte{0xFF, 0xFF, 0xFF, 0xFF}, t)
}

func verifyWriteCString(val string, expected []byte, t *testing.T) {
	w := new(bytes.Buffer)
	_, err := WriteCString(w, val)
	if err != nil {
		t.Errorf("Unexpected error in encoding %v", err)
	}
	written := w.Bytes()
	result := bytes.Compare(written, expected)
	if result != 0 {
		t.Errorf("Expected %v; got %v", expected, written)
	}
}

func TestWriteCString(t *testing.T) {
	verifyWriteCString("", []byte{0}, t)
	verifyWriteCString("a", []byte{'a', 0}, t)
	verifyWriteCString("pgsql", []byte{'p', 'g', 's', 'q', 'l', 0}, t)
	// TODO: more unicode tests
	verifyWriteCString("éè", []byte{0xC3, 0xA9, 0xC3, 0xA8, 0}, t)
}

func verifyReadInt16(val []byte, expected int16, t *testing.T) {
	r := bytes.NewReader(val)

	result, err := ReadInt16(r)
	if err != nil {
		t.Errorf("Unexpected error in decoding %v", err)
	}
	if result != expected {
		t.Errorf("Expected %v; got %v", expected, result)
	}
}

func TestReadInt16(t *testing.T) {
	verifyReadInt16([]byte{0x00, 0x00}, 0, t)
	verifyReadInt16([]byte{0x00, 0x01}, 1, t)
	verifyReadInt16([]byte{0x00, 0xFF}, 0xFF, t)
	verifyReadInt16([]byte{0x7F, 0xFF}, 0x7FFF, t)
	verifyReadInt16([]byte{0xFF, 0xFF}, -1, t)
}

func verifyReadUint16(val []byte, expected uint16, t *testing.T) {
	r := bytes.NewReader(val)

	result, err := ReadUint16(r)
	if err != nil {
		t.Errorf("Unexpected error in decoding %v", err)
	}
	if result != expected {
		t.Errorf("Expected %v; got %v", expected, result)
	}
}

func TestReadUint16(t *testing.T) {
	verifyReadUint16([]byte{0x00, 0x00}, 0, t)
	verifyReadUint16([]byte{0x00, 0x01}, 1, t)
	verifyReadUint16([]byte{0x00, 0xFF}, 0xFF, t)
	verifyReadUint16([]byte{0x7F, 0xFF}, 0x7FFF, t)
	verifyReadUint16([]byte{0xFF, 0xFF}, 0xFFFF, t)
}

func verifyReadInt32(val []byte, expected int32, t *testing.T) {
	r := bytes.NewReader(val)

	result, err := ReadInt32(r)
	if err != nil {
		t.Errorf("Unexpected error in decoding %v", err)
	}
	if result != expected {
		t.Errorf("Expected %v; got %v", expected, result)
	}
}

func TestReadInt32(t *testing.T) {
	verifyReadInt32([]byte{0x00, 0x00, 0x00, 0x00}, 0, t)
	verifyReadInt32([]byte{0x00, 0x00, 0x00, 0x01}, 1, t)
	verifyReadInt32([]byte{0x00, 0x00, 0x00, 0xFF}, 0xFF, t)
	verifyReadInt32([]byte{0x00, 0x00, 0xFF, 0xFF}, 0xFFFF, t)
	verifyReadInt32([]byte{0x00, 0xFF, 0xFF, 0xFF}, 0xFFFFFF, t)
	verifyReadInt32([]byte{0x00, 0xFF, 0xFF, 0xFF}, 0xFFFFFF, t)
	verifyReadInt32([]byte{0x00, 0xFF, 0xFF, 0xFF}, 0xFFFFFF, t)
	verifyReadInt32([]byte{0x7F, 0xFF, 0xFF, 0xFF}, 0x7FFFFFFF, t)
	verifyReadInt32([]byte{0xFF, 0xFF, 0xFF, 0xFF}, -1, t)
}

func verifyReadUint32(val []byte, expected uint32, t *testing.T) {
	r := bytes.NewReader(val)

	result, err := ReadUint32(r)
	if err != nil {
		t.Errorf("Unexpected error in decoding %v", err)
	}
	if result != expected {
		t.Errorf("Expected %v; got %v", expected, result)
	}
}

func TestReadUint32(t *testing.T) {
	verifyReadUint32([]byte{0x00, 0x00, 0x00, 0x00}, 0, t)
	verifyReadUint32([]byte{0x00, 0x00, 0x00, 0x01}, 1, t)
	verifyReadUint32([]byte{0x00, 0x00, 0x00, 0xFF}, 0xFF, t)
	verifyReadUint32([]byte{0x00, 0x00, 0xFF, 0xFF}, 0xFFFF, t)
	verifyReadUint32([]byte{0x00, 0xFF, 0xFF, 0xFF}, 0xFFFFFF, t)
	verifyReadUint32([]byte{0x7F, 0xFF, 0xFF, 0xFF}, 0x7FFFFFFF, t)
	verifyReadUint32([]byte{0xFF, 0xFF, 0xFF, 0xFF}, 0xFFFFFFFF, t)
}

func verifyReadUint32FromBuffer(val []byte, expected uint32, t *testing.T) {
	buf := bytes.NewBuffer(val)

	result := ReadUint32FromBuffer(buf)
	if result != expected {
		t.Errorf("Expected %v; got %v", expected, result)
	}
}

func TestReadUint32FromBuffer(t *testing.T) {
	verifyReadUint32FromBuffer([]byte{0x00, 0x00, 0x00, 0x00}, 0, t)
	verifyReadUint32FromBuffer([]byte{0x00, 0x00, 0x00, 0x01}, 1, t)
	verifyReadUint32FromBuffer([]byte{0x00, 0x00, 0x00, 0xFF}, 0xFF, t)
	verifyReadUint32FromBuffer([]byte{0x00, 0x00, 0xFF, 0xFF}, 0xFFFF, t)
	verifyReadUint32FromBuffer([]byte{0x00, 0xFF, 0xFF, 0xFF}, 0xFFFFFF, t)
	verifyReadUint32FromBuffer([]byte{0x7F, 0xFF, 0xFF, 0xFF}, 0x7FFFFFFF, t)
	verifyReadUint32FromBuffer([]byte{0xFF, 0xFF, 0xFF, 0xFF}, 0xFFFFFFFF, t)
}

func verifyReadCString(val []byte, expected string, t *testing.T) {
	r := bytes.NewReader(val)

	result, err := ReadCString(r)
	if err != nil {
		t.Errorf("Unexpected error in decoding %v", err)
	}
	if result != expected {
		t.Errorf("Expected %v; got %v", expected, result)
	}
}

func TestReadCString(t *testing.T) {
	verifyReadCString([]byte{0}, "", t)
	verifyReadCString([]byte{'a', 0}, "a", t)
	verifyReadCString([]byte{'p', 'g', 's', 'q', 'l', 0}, "pgsql", t)
	// TODO: more unicode tests
	verifyReadCString([]byte{0xC3, 0xA9, 0xC3, 0xA8, 0}, "éè", t)
}

func verifyReadByte(val byte, t *testing.T) {
	r := bytes.NewReader([]byte{val})

	result, err := ReadByte(r)
	if err != nil {
		t.Errorf("Unexpected error in decoding %v", err)
	}
	if result != val {
		t.Errorf("Expected %v; got %v", val, result)
	}
}

func TestReadByte(t *testing.T) {
	verifyReadByte(0, t)
	verifyReadByte(1, t)
	verifyReadByte(0xFF, t)
}
