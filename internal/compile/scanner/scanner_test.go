package scanner

import (
	"bytes"
	"errors"
	"io"
	"testing"
)

const src = `
const std = @import("std");

func main() void {
	return 1;
}
`

func TestScanner(t *testing.T) {
	rd := bytes.NewReader([]byte(src))
	scn := New(rd)

	for {
		tok, err := scn.Scan()
		if err != nil {
			if errors.Is(err, io.EOF) {
				return
			}
			t.Fatal(err)
		}

		t.Logf("%#v", tok)
	}
}
