package dogconf
/*
import (
	"fmt"
	"strings"
	"testing"
)

func TestParseRequest(t *testing.T) {
	for _, patchReq := range []string {
		`[route 'a' @ 42 [patch [addr='123.123']]]`,
		`[route 'my-very-long-server-identifier-maybe-a-uuid' @ 5
		 [patch [addr='123.123.123.123:5432', lock='t', user='foo', password='bar']]]`,
		`[route '''' @ 522423
		 [patch [lock='t', password='bar',user='foo']]]`,
		`[route 'foo' @ 5
		 [patch [lock='t']]]`,
		`[route '''' @ 5 [patch [lock='t']]]`,
		`[route all [get]]`,
		`[route all [delete]`,
		`[route 'foo' @ 42 [delete]]`,
		`[route 'bar' @ 42 [create [addr='123.123.123.125:5445']]]`,
		`[route '!' @ 5
		 [patch [password='x'',"',lock='true']]]`} {
		r, err := ParseRequest(strings.NewReader(patchReq))
		if err != nil {
			fmt.Printf("%v\n", err)
		} else {
			fmt.Printf("%v\n", r)
		}
	}
	
}*/