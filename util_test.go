package femebe

import (
	"bytes"
	"errors"
	"io"
	"testing"
)

type closableBuffer struct {
	io.ReadWriter
}

func (c *closableBuffer) Close() error {
	// noop, to satisfy interface
	return nil
}

func newClosableBuffer(buf *bytes.Buffer) *closableBuffer {
	return &closableBuffer{buf}
}

func TestErrToChannel(t *testing.T) {
	expectErr := func(errCh <- chan error) {
		select {
		case <- errCh:
			// do nothing
		default:
			t.Errorf("no error available on channel; want error")
		}
	}
	errAfter := func(n int) (func() error) {
		return func() error {
			n--
			if n == 0 {
				return errors.New("err")
			} else {
				return nil
			}
		}
	}
	errCh := make(chan error, 1)
	errToChannel(errAfter(1), errCh)
	expectErr(errCh)
	errToChannel(errAfter(2), errCh)
	expectErr(errCh)
	errToChannel(errAfter(5), errCh)
	expectErr(errCh)
	select {
	case err := <- errCh:
		t.Errorf("got error %v; want nil", err)
	default:
		// do nothing
	}
}
