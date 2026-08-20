// Package a holds the fixtures for the nountypedmap analyzer. The code
// is deliberately bad: it exists to trigger diagnostics.
package a

// Alias is the same type as the map it names, so its use sites are
// reported.
type Alias = map[string]any

// Named is a defined type, so its use sites are accepted.
type Named map[string]any

// Iface has a method, so it is not the empty interface.
type Iface interface{ Do() }

// Empty is a named empty interface. The rule does not take the
// underlying type, so a map of Empty is accepted.
type Empty interface{}

func Param(m map[string]any) {} // want `parameter uses map\[string\]any; describe the data with a named struct`

func Result() map[string]interface{} { // want `result uses map\[string\]interface\{\}; describe the data with a named struct`
	return nil
}

func Both(m map[string]any) map[string]any { // want `parameter uses map\[string\]any; describe the data with a named struct` `result uses map\[string\]any; describe the data with a named struct`
	return m
}

func AliasParam(m Alias) {} // want `parameter uses Alias; describe the data with a named struct`

func NamedParam(m Named) {}

func (n Named) Get(k string) Named { return n }

func ErrorMap(m map[string]error) {}

func IfaceMap(m map[string]Iface) {}

func EmptyIfaceMap(m map[string]Empty) {}

func AnyKey(m map[any]string) {}

func SliceOfMaps(m []map[string]any) {}

func PointerToMap(m *map[string]any) {}

func Generic[T any](m map[string]T) {}

// Box is a generic type. An instantiation is a named type, so the rule
// accepts it. G03 and G04 cover the "any" argument.
type Box[T any] struct{ V T }

func TakeBox(b Box[any]) {}

// Variadic hides the map in a slice, so the rule accepts it.
func Variadic(ms ...map[string]any) {}

// Config carries a reported field and an accepted one.
type Config struct {
	Extra map[string]any // want `struct field uses map\[string\]any; describe the data with a named struct`
	Name  string
}

// InnerStruct declares a struct in a function body. The spec names
// struct fields with no location, so the field is reported.
func InnerStruct() {
	type payload struct {
		Data map[string]any // want `struct field uses map\[string\]any; describe the data with a named struct`
		ID   string
	}
	_ = payload{}
}

// Typed is a package-level variable with a written type.
var Typed map[string]any // want `package-level variable uses map\[string\]any; describe the data with a named struct`

// Inferred is a package-level variable with an inferred type.
var Inferred = map[string]any{} // want `package-level variable uses map\[string\]any; describe the data with a named struct`

// Count and Bag share one spec. Only the map is reported, at the name
// of the map.
var Count, Bag = 1, map[string]any{} // want `package-level variable uses map\[string\]any; describe the data with a named struct`

var Clean = map[string]string{}

// Nested holds a map of maps. The rule tests the value type only and
// does not walk into it.
var Nested map[string]map[string]any

// A blank name still declares a package-level variable, so it is
// reported.
var _ = map[string]any{} // want `package-level variable uses map\[string\]any; describe the data with a named struct`

// Anon holds an anonymous struct. The field is reported once.
var Anon = struct {
	Data map[string]any // want `struct field uses map\[string\]any; describe the data with a named struct`
}{}

// Callback holds a function type in a field. The parameter is reported
// once, from the function type.
type Callback struct {
	Run func(m map[string]any) // want `parameter uses map\[string\]any; describe the data with a named struct`
}

var NamedVar Named

const Limit = 10

// Locals are never reported. A function literal in the body carries a
// signature, so its parameter is reported.
func Locals() {
	var written map[string]any
	inferred := map[string]any{}
	inner := func(m map[string]any) {} // want `parameter uses map\[string\]any; describe the data with a named struct`
	_, _, _ = written, inferred, inner
}

// Handler is a function type declaration. The parameter is reported
// once, from the function type.
type Handler func(m map[string]any) // want `parameter uses map\[string\]any; describe the data with a named struct`

// Literal holds a function literal.
var Literal = func(m map[string]any) {} // want `parameter uses map\[string\]any; describe the data with a named struct`

// Store is an interface. Its method signatures are reported.
type Store interface {
	Save(m map[string]any) error  // want `parameter uses map\[string\]any; describe the data with a named struct`
	Load(k string) map[string]any // want `result uses map\[string\]any; describe the data with a named struct`
	Keys(k string) map[string]bool
}

// AliasInferred takes its type from an alias. The message prints the
// type through go/types, and GODEBUG=gotypesalias controls whether the
// alias survives, so the pattern accepts both spellings.
var AliasInferred = Alias{} // want `package-level variable uses (Alias|map\[string\]any); describe the data with a named struct`
