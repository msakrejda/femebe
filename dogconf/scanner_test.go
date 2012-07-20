package dogconf
/*
import (
	"fmt"
	"strings"
	"testing"
)

func TestScan(t *testing.T) {
	for _, r := range []string {
		`'"''"'`,
		`  [\n]@=[@  \nfoo   'bar'  1233323    x ` } {
		fmt.Printf("Scanning %v\n", r)
		var s = new(Scanner)
		s.Init(strings.NewReader(r))
		for token := s.Scan(); token.Type != EOF; token = s.Scan() {
			fmt.Printf("Scanned '%v'\n", token)
			fmt.Printf("Peeking at '%v'\n", s.Peek())
			fmt.Printf("Peeking again, just fort the lulz: '%v'\n", s.Peek())
		}		
	}

}
*/