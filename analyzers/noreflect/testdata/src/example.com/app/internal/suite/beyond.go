package suite

import "reflect"

// Kind reads the type of a value, which is a use of reflect beyond
// DeepEqual. The allowance covers DeepEqual alone, so this file
// reports like a test file that uses more of reflect.
func Kind(value any) string {
	return reflect.TypeOf(value).Kind().String() // want `this test file uses reflect beyond DeepEqual`
}
