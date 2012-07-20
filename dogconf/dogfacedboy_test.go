package dogconf

import (
	"fmt"
	"strings"
	"testing"
)

func TestDogFacedBoy(t *testing.T) {
	sm := NewSessionMap()
	rm := NewRouteMap()
	for _, patchReq := range []string{
		`[route all [delete]]`,
		`[route 'foo' @ 42 [delete]]`,
		`[route 'bar' @ 42 [create [addr='123.123.123.125:5445']]]`,
		`[route 'bar' [create [addr='123.124.123.125:5445']]]`,
		`[route 'bar' @ 1 [patch [addr='123.123.123.125:5445']]]`,
		`[route 'bar' [get]]`,
		`[route all [get]]`,

		`[route '!xp' @ 5 [patch [password='x'',"',lock='true']]]`} {

		result, err := HandleRequest(rm, sm, strings.NewReader(patchReq))

		fmt.Printf("%v\n", patchReq)
		fmt.Printf("%v\n", result)

		if err != nil {
			fmt.Printf("\t%v\n", err)
		}
	}
}
