package a

// TestSink collects the messages of a test. A test file declares it,
// so the test files of the package own it. An assignment to it patches
// no production code, in this file or in another test file.
var TestSink = func(message string) {}

// TestRegistry is a container that a test file declares. The test
// files of the package own it, entries included.
var TestRegistry = map[string]func(){}

// resetSink restores the sink from the file that declares it.
func resetSink() {
	TestSink = func(message string) {}
}
