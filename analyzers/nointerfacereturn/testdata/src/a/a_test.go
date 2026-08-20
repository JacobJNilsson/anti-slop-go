package a

import "testing"

// A test file gets no exemption. A helper that always builds one
// concrete type hides that type from the test that reads the value.
func newTestStore() Storage { return &memStore{} } // want `result uses Storage, and every return builds \*memStore; return the concrete type`

func TestLoad(t *testing.T) {
	if got := newTestStore().Load("k"); got != "k" {
		t.Errorf("Load(k) = %q; want %q", got, "k")
	}
}
