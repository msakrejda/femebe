package codec

import (
	"github.com/uhoh-itsmaciek/femebe/proto"
	"testing"
)

func TestGuessOids(t *testing.T) {
	verifyOids := func(rows [][]interface{}, expected []proto.Oid) {
		result := GuessOids(rows)
		if len(result) != len(expected) {
			t.Errorf("Expected %v; got %v", expected, result)
			return
		}
		for i := range result {
			if result[i] != expected[i] {
				t.Errorf("Expected %v; got %v", expected[i], result[i])
				break
			}
		}
	}
	verifyOids([][]interface{}{}, []proto.Oid{})
}
