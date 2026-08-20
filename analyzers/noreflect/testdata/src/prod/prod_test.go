package prod

import "testing"

// The test file of the package imports no reflect, so it reports
// nothing.
func TestDescribe(t *testing.T) {
	if Describe(1) != "int" {
		t.Fatal("the type of an integer is not int")
	}
}
