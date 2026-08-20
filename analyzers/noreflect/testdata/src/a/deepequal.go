package a

import "reflect" // want `this package imports reflect`

// The DeepEqual exemption covers test files only. A production file
// that compares two values with DeepEqual gets a report, because the
// comparison belongs to the types themselves.
func EqualLists(left, right []string) bool {
	return reflect.DeepEqual(left, right)
}
