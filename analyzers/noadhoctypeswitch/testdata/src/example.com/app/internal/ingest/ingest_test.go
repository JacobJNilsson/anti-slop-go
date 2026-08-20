package ingest_test

import (
	"testing"

	"example.com/app/internal/ingest"
)

// The external test package carries the path of the package with
// "_test" at the end. The rule drops that suffix, so one entry covers
// both packages and this switch stays clean.
func TestDecode(t *testing.T) {
	var v any = 1
	switch v.(type) {
	case int:
	}
	if ingest.Decode(1) != "int" {
		t.Error("Decode read another type")
	}
}
