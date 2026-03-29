package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestCmdVersion(t *testing.T) {
	var buf bytes.Buffer
	if err := cmdVersion(&buf); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := buf.String()
	for _, want := range []string{"dotmem version", "commit:", "built:", "go:"} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q:\n%s", want, out)
		}
	}
}
