package util

import (
	"errors"
	"testing"
)

func TestErrToChannel(t *testing.T) {
	expectErr := func(errCh <-chan error) {
		select {
		case <-errCh:
			// do nothing
		default:
			t.Errorf("no error available on channel; want error")
		}
	}
	errAfter := func(n int) func() error {
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
	ErrToChannel(errAfter(1), errCh)
	expectErr(errCh)
	ErrToChannel(errAfter(2), errCh)
	expectErr(errCh)
	ErrToChannel(errAfter(5), errCh)
	expectErr(errCh)
	select {
	case err := <-errCh:
		t.Errorf("got error %v; want nil", err)
	default:
		// do nothing
	}
}
