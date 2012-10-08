package dogconf

import (
	"./stable"
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"
)

// Used for fast and lucid handling of regression file generation.
// The main function is to record, in-memory, the bytes written to a
// result file (as to avoid re-reading them from disk) for future
// comparison against expected-output files on disk.
type resultFile struct {
	io.Writer
	io.Closer
	Byteser
	buf     bytes.Buffer
	diskOut io.WriteCloser
}

type Byteser interface {
	Bytes() []byte
}

// Open a new resultFile, failing the test should that not be
// possible.
func newResultFile(t *testing.T, path string) *resultFile {
	destFile, err := os.OpenFile(path,
		os.O_CREATE|os.O_TRUNC|os.O_RDWR, 0666)
	if err != nil {
		t.Fatalf("Could not open results file at %v: %v",
			path, err)
	}

	rf := resultFile{}
	rf.diskOut = destFile
	rf.Writer = io.MultiWriter(destFile, &rf.buf)
	rf.Closer = rf.diskOut
	rf.Byteser = &rf.buf

	return &rf
}

func astRegress(t *testing.T, name string, input string) {
	// Set up destination file to dump test results
	destFileName := filepath.Join("ast_regress", "results", name) + ".out"
	resultOut := newResultFile(t, destFileName)
	defer resultOut.Close()

	// Write the input to the top of the output file because that
	// makes it easier to skim the corresponding results
	// immediately after it.
	_, err := io.WriteString(resultOut, "INPUT<\n"+input+"\n\n")
	if err != nil {
		t.Fatalf("Could echo test input to results file: %v", err)
	}

	result, err := ParseRequest(bytes.NewBuffer([]byte(input)))

	// Run the parser
	formatted := stable.Sprintf("%#v\n", result)
	_, err = io.WriteString(resultOut, "OUTPUT>\n"+formatted)
	if err != nil {
		t.Fatalf("Could write test output to results file: %v", err)
	}

	// Open the expected-output file
	expectedFileName := filepath.Join(
		"ast_regress", "expected", name) + ".out"
	expectedFile, err := os.OpenFile(expectedFileName, os.O_RDONLY, 0666)
	if err != nil {
		t.Fatalf("Could not open expected output file at %v: %v",
			expectedFileName, err)
	}
	defer expectedFile.Close()

	// Perform a quick comparison between the bytes in memory and
	// the bytes on disk.  It is the intention that at a later
	// date a diff can be emitted in the slow-path when there is a
	// failure, even though technically 'diff' could also be
	// expensively used to determine if the test failed or not.
	resultBytes := []byte(resultOut.Bytes())

	// Read one more byte than required to see if expected output
	// is longer than result output.
	expectedBytes := make([]byte, len(resultBytes)+1)

	n, err := io.ReadAtLeast(expectedFile, expectedBytes, len(expectedBytes))
	switch err {
	case io.EOF:
		t.Fatalf("Expected output file is empty: %v", expectedFile)

	case io.ErrUnexpectedEOF:
		// Check if the read input has the same size and
		// contents.  The test must succeed if it does.
		if n != len(resultBytes) ||
			!bytes.Equal(resultBytes, expectedBytes[0:n]) {
			t.Fatal("Difference between results and expected")
		}

		// Test success
	case nil:
		t.Fatal("Difference between results and expected")
	default:
		t.Fatalf("ast_regress bug: unexpected error %v", err)
	}
}

func TestDeleteAll(t *testing.T) {
	astRegress(t, "delete_all", `[route all [delete]]`)
}

func TestDeleteAt(t *testing.T) {
	astRegress(t, "delete_at", `[route 'foo' @ 42 [delete]]`)
}

func TestCreateRouteAtTime(t *testing.T) {
	astRegress(t, "create_route_at",
		`[route 'bar' @ 42 [create [addr='123.123.123.125:5445']]]`)
}

func TestCreateRoute(t *testing.T) {
	astRegress(t, "create_route",
		`[route 'bar' [create [addr='123.124.123.125:5445']]]`)
}

func TestPatchRoute(t *testing.T) {
	astRegress(t, "patch_at_address",
		`[route 'bar' @ 1 [patch [addr='123.123.123.125:5445']]]`)
}

func TestGetRoute(t *testing.T) {
	astRegress(t, "get_one_route", `[route 'bar' [get]]`)
}

func TestGetAllRoutes(t *testing.T) {
	astRegress(t, "get_all_routes", `[route all [get]]`)
}

func TestQuoting(t *testing.T) {
	astRegress(t, "quoting",
		`[route '!xp' @ 5 [patch [password='x'',"',lock='true']]]`)
}
