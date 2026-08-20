// Package a holds the fixtures for the nolaundering analyzer. The code
// is deliberately bad: it exists to trigger diagnostics.
package a

// User is the concrete type that most fixtures widen and take back.
type User struct{ Name string }

// Admin is a second concrete type. It has no method.
type Admin struct{ Name string }

// String makes User satisfy Stringer.
func (u User) String() string { return u.Name }

// Empty is a named empty interface.
type Empty interface{}

// Alias names the empty interface, so it is the same type.
type Alias = any

// Stringer is the narrow interface that User satisfies.
type Stringer interface{ String() string }

// Reader and ReadWriter drive the narrowing case. No widening step
// feeds an assertion from Reader to ReadWriter, so rule G01 owns it.
type Reader interface{ Read() }

// ReadWriter is broader than Reader.
type ReadWriter interface {
	Read()
	Write()
}

// Box holds a widened field. Fields are out of scope.
type Box struct{ V any }

// Produce returns a value that the caller cannot know.
func Produce() any { return User{} }

// pair returns two values, so an assignment from it takes a tuple.
func pair() (User, error) { return User{}, nil }

// --- The chained form: a widening step feeds the assertion. ---

// SameType hides the type and asks for it again in one expression.
func SameType(u User) User {
	return any(u).(User) // want `this assertion takes back a value that the operand widens from User to any; remove the widening`
}

// OtherType shows that the conversion manufactures the assertability:
// the code knows u is a User, and Admin is another type.
func OtherType(u User) Admin {
	return any(u).(Admin) // want `this assertion takes back a value that the operand widens from User to any; remove the widening`
}

// LiteralInterface writes the empty interface out.
func LiteralInterface(u User) User {
	return interface{}(u).(User) // want `this assertion takes back a value that the operand widens from User to interface\{\}; remove the widening`
}

// NamedEmpty widens to a named empty interface.
func NamedEmpty(u User) User {
	return Empty(u).(User) // want `this assertion takes back a value that the operand widens from User to Empty; remove the widening`
}

// AliasWiden widens through an alias of the empty interface.
func AliasWiden(u User) User {
	return Alias(u).(User) // want `this assertion takes back a value that the operand widens from User to Alias; remove the widening`
}

// Nested widens twice. The analyzer follows the whole chain.
func Nested(u User) User {
	return any(any(u)).(User) // want `this assertion takes back a value that the operand widens from User to any; remove the widening`
}

// Parenthesised puts the widening in parentheses.
func Parenthesised(u User) User {
	return (any(u)).(User) // want `this assertion takes back a value that the operand widens from User to any; remove the widening`
}

// ChainedAssert widens a narrow interface with an assertion that
// cannot fail, then takes the concrete type back.
func ChainedAssert(s Stringer) User {
	return s.(any).(User) // want `this assertion takes back a value that the operand widens from Stringer to any; remove the widening`
}

// ChainedImpossible needs the widening step to compile, because Admin
// has no String method and s.(Admin) is a compile error.
func ChainedImpossible(s Stringer) Admin {
	return s.(any).(Admin) // want `this assertion takes back a value that the operand widens from Stringer to any; remove the widening`
}

// SwitchOnWidening branches on a type the expression just hid.
func SwitchOnWidening(u User) string {
	switch t := any(u).(type) { // want `this type switch takes back a value that the operand widens from User to any; remove the widening`
	case User:
		return t.Name
	}
	return ""
}

// --- The chained form: accepted code. ---

// NarrowOnly asks a question the code cannot answer, and no widening
// step feeds it. Rule G01 owns the justification.
func NarrowOnly(r Reader) ReadWriter {
	return r.(ReadWriter)
}

// FromParam asserts a parameter. The caller widened, and this analyzer
// reads one function.
func FromParam(v any) User {
	return v.(User)
}

// NoOpConversion converts the empty interface to the empty interface,
// so the operand hides nothing.
func NoOpConversion(v any) User {
	return any(v).(User)
}

// FromCallResult asserts the result of a call. The result type is the
// empty interface already.
func FromCallResult() User {
	return Produce().(User)
}

// FieldWiden widens into a struct field. Fields are out of scope.
func FieldWiden(u User) User {
	var b Box
	b.V = u
	return b.V.(User)
}

// --- The binding form: one function widens and takes back. ---

// VarWiden declares the binding with the empty interface type.
func VarWiden(u User) User {
	var v any = u
	return v.(User) // want `this assertion takes back a value that v widens from User to any at line 138; remove the widening`
}

// ShortWiden declares the binding with a short variable declaration.
func ShortWiden(u User) User {
	v := any(u)
	return v.(User) // want `this assertion takes back a value that v widens from User to any at line 144; remove the widening`
}

// NamedInterfaceBinding widens to a narrower interface than the empty
// one. The rule covers every interface, not the empty one alone.
func NamedInterfaceBinding(u User) User {
	var s Stringer = u
	return s.(User) // want `this assertion takes back a value that s widens from User to Stringer at line 151; remove the widening`
}

// BranchWiden assigns in one branch. The zero value of the binding is
// nil, and an assertion on nil fails, so the branch does not make the
// assertion honest.
func BranchWiden(u User, on bool) User {
	var v any
	if on {
		v = u
	}
	return v.(User) // want `this assertion takes back a value that v widens from User to any at line 161; remove the widening`
}

// TwoWidenings puts two types in one binding, which makes a union.
// The assertion separates the union, so it asks a real question and
// the rule accepts it.
func TwoWidenings(u User, ad Admin, on bool) bool {
	var v any = u
	if on {
		v = ad
	}
	_, ok := v.(User)
	return ok
}

// PairedSpec declares two bindings in one statement.
func PairedSpec(u User) User {
	var v, w any = u, u
	_ = w
	return v.(User) // want `this assertion takes back a value that v widens from User to any at line 180; remove the widening`
}

// CommaOk asks a question that the widening already answered, so the
// ok result is dead weight.
func CommaOk(u User) bool {
	v := any(u)
	_, ok := v.(User) // want `this assertion takes back a value that v widens from User to any at line 188; remove the widening`
	return ok
}

// SwitchOnBinding branches on a type the function knows.
func SwitchOnBinding(u User) string {
	v := any(u)
	switch t := v.(type) { // want `this type switch takes back a value that v widens from User to any at line 195; remove the widening`
	case User:
		return t.Name
	}
	return ""
}

// Justified carries a SAFETY comment. Rule G01 accepts the comment.
// This rule still reports, because the fix is to delete the widening
// and not to justify the assertion.
func Justified(u User) User {
	v := any(u)
	// SAFETY: the line above puts a User in v.
	return v.(User) // want `this assertion takes back a value that v widens from User to any at line 207; remove the widening`
}

// TwiceAsserted gets one report for each assertion.
func TwiceAsserted(u User) (User, User) {
	v := any(u)
	return v.(User), v.(User) // want `v widens from User to any at line 214` `v widens from User to any at line 214`
}

// Shadowed declares an inner binding that hides the parameter. The
// inner binding launders; the parameter does not.
func Shadowed(v any, u User) (User, User) {
	inner := User{}
	{
		v := any(u)
		inner = v.(User) // want `this assertion takes back a value that v widens from User to any at line 223; remove the widening`
	}
	return inner, v.(User)
}

// InLiteral shows that a function literal is its own function.
func InLiteral(u User) func() User {
	return func() User {
		v := any(u)
		return v.(User) // want `this assertion takes back a value that v widens from User to any at line 232; remove the widening`
	}
}

// BothForms has a laundered binding and a widening operand. The
// operand is the closer evidence, so the analyzer reports it once.
func BothForms(u User) User {
	s := Stringer(u)
	return any(s).(User) // want `this assertion takes back a value that the operand widens from Stringer to any; remove the widening`
}

// ConstantBinding widens a constant. The default type of the constant
// is the evidence that the widening hides.
func ConstantBinding() bool {
	var v any = 5
	_, ok := v.(int) // want `this assertion takes back a value that v widens from int to any at line 247; remove the widening`
	return ok
}

// GenericBound widens a type parameter. The constraint of T is not
// the type that the variable holds, and the analyzer reads no generic
// function.
func GenericBound[T Stringer](t T) bool {
	v := any(t)
	_, ok := v.(User)
	return ok
}

// --- The binding form: accepted code. ---

// MixedSources gives the binding a value that the function cannot
// know, so the assertion answers a real question.
func MixedSources(u User, in any) User {
	v := any(u)
	v = in
	return v.(User)
}

// FromCallBinding takes the value from a call that returns any.
func FromCallBinding() User {
	v := Produce()
	return v.(User)
}

// FromMapBinding reads a map of dynamic values.
func FromMapBinding(m map[string]any) User {
	v := m["k"]
	return v.(User)
}

// FromChannelBinding receives a dynamic value.
func FromChannelBinding(ch chan any) User {
	v := <-ch
	return v.(User)
}

// FromParameterBinding copies a parameter, which holds no evidence.
func FromParameterBinding(p any) User {
	v := p
	return v.(User)
}

// OtherVariable asserts a variable that no widening feeds.
func OtherVariable(u User, p any) User {
	v := any(u)
	_ = v
	return p.(User)
}

// Captured lets a function literal read the binding. The analyzer
// reads one function at a time, so it drops the binding.
func Captured(u User) func() User {
	v := any(u)
	return func() User {
		return v.(User)
	}
}

// AddressTaken gives the address away, so a statement the analyzer
// cannot see can put another value in the binding.
func AddressTaken(u User, set func(*any)) User {
	v := any(u)
	set(&v)
	return v.(User)
}

// TupleAssign takes both values from one call. The analyzer does not
// read the parts of a tuple, so it drops the binding.
func TupleAssign(u User) User {
	var v any = u
	v, _ = pair()
	return v.(User)
}

// SpecFromTuple declares two bindings from one call, so the
// declaration takes a tuple and the analyzer drops both bindings.
func SpecFromTuple() User {
	var v, w any = pair()
	_ = w
	return v.(User)
}

// AddressOfLiteral takes the address of a literal, and a literal is no
// binding.
func AddressOfLiteral(u User) *Box {
	return &Box{V: u}
}

// RangeAssigned takes values from a slice of dynamic values.
func RangeAssigned(u User, all []any) User {
	var v any = u
	for _, v = range all {
	}
	return v.(User)
}

// NilBinding holds no evidence, because nil has no type to hide.
func NilBinding() bool {
	var v any = nil
	_, ok := v.(User)
	return ok
}

// GenericAny has no evidence to lose, because the constraint of T is
// the empty interface.
func GenericAny[T any](t T) bool {
	v := any(t)
	_, ok := v.(User)
	return ok
}

// LocalConst declares a constant, which is not a binding.
func LocalConst(p any) User {
	const name = "root"
	_ = name
	return p.(User)
}

// InterfaceSource widens a narrower interface into the empty one. The
// interface does not name the type that the variable holds, so the
// assertion asks a real question.
func InterfaceSource(s Stringer) User {
	var v any = s
	return v.(User)
}

// GenericChained shows the shape that the Go language demands: a value
// of a parameterized type takes an assertion only through an
// interface, so the conversion is no choice of the author.
func GenericChained[T Stringer](t T) User {
	return any(t).(User)
}

// Holder is a generic type. Its methods take the type parameters of
// the receiver.
type Holder[T any] struct{ V T }

// Get launders a value that carries no type parameter. A generic
// method gives no shelter to that shape.
func (h Holder[T]) Get(u User) User {
	return any(u).(User) // want `this assertion takes back a value that the operand widens from User to any; remove the widening`
}

// The blank name declares a function that nothing calls. The analyzer
// reads it like every other function.
func _(u User) User {
	return any(u).(User) // want `this assertion takes back a value that the operand widens from User to any; remove the widening`
}

// InterfaceTargetChained asks whether the value satisfies an
// interface. The widening is still there, but the question stands on
// its own, so the rule leaves it.
func InterfaceTargetChained(u User) bool {
	_, ok := any(u).(Stringer)
	return ok
}

// InterfaceTargetBinding asks the same question about a binding.
func InterfaceTargetBinding(u User) bool {
	var v any = u
	_, ok := v.(Stringer)
	return ok
}

// --- Package level. ---

// defaultUser feeds the package-level fixtures.
var defaultUser = User{Name: "root"}

// Global shows that the chained form needs no function around it.
var Global = any(defaultUser).(User) // want `this assertion takes back a value that the operand widens from User to any; remove the widening`

// pkgWidened is a package-level widening. The analyzer tracks a
// binding inside one function only, so the assertion below stays
// clean.
var pkgWidened any = defaultUser

// PackageBinding asserts a package-level binding.
func PackageBinding() User {
	return pkgWidened.(User)
}

// --- Generic code: the type parameter decides. ---

// GenericPlain launders a value that carries no type parameter. The
// function is generic, and the widening is still a choice.
func GenericPlain[T any](t T, u User) User {
	_ = t
	return any(u).(User) // want `this assertion takes back a value that the operand widens from User to any; remove the widening`
}

// GenericSliceTarget gives a concrete value back in the parameterized
// type. Go accepts no other spelling, so the rule leaves it. Every
// other element type, such as a pointer, an array, and a channel,
// takes the same path.
func GenericSliceTarget[T any](v []int32) []T {
	return any(v).([]T)
}

// GenericMapTarget names the type parameter in the value type of a
// map.
func GenericMapTarget[T any](v map[string]int32) map[string]T {
	return any(v).(map[string]T)
}

// GenericNamedTarget names the type parameter in the type argument of
// a named type.
func GenericNamedTarget[T any](v Holder[int32]) Holder[T] {
	return any(v).(Holder[T])
}

// GenericStructTarget names the type parameter in a field of a struct
// type.
func GenericStructTarget[T any](v struct{ V int32 }) struct{ V T } {
	return any(v).(struct{ V T })
}

// GenericFuncTarget names the type parameter in a parameter of a
// function type.
func GenericFuncTarget[T any](v func(int32)) func(T) {
	return any(v).(func(T))
}

// --- More edges. ---

// NeverAssigned holds the zero value only. No widening feeds the
// assertion, so the rule accepts it.
func NeverAssigned() bool {
	var v any
	_, ok := v.(User)
	return ok
}

// DoubleWiden puts the binding in the empty interface again. The
// second step hides nothing, so the analyzer looks through it and
// reports the binding under it.
func DoubleWiden(u User) User {
	v := any(u)
	return any(v).(User) // want `this assertion takes back a value that v widens from User to any at line 490; remove the widening`
}

// TwoSameWidenings puts the same type in the binding twice. The
// binding stays laundered, and the message names the first widening.
func TwoSameWidenings(u User, other User, on bool) User {
	var v any = u
	if on {
		v = other
	}
	return v.(User) // want `this assertion takes back a value that v widens from User to any at line 497; remove the widening`
}

// BranchCommaOk asks with the comma-ok form after one branch widens.
// The rule reports it. The fix here is not to delete the widening:
// hold the value in a typed variable with a separate bool, or in a
// typed pointer.
func BranchCommaOk(u User, on bool) (User, bool) {
	var v any
	if on {
		v = u
	}
	got, ok := v.(User) // want `this assertion takes back a value that v widens from User to any at line 511; remove the widening`
	return got, ok
}

// pkgOnly is a package-level widening that only package-level code
// reads. The analyzer binds a variable inside one function, so it
// tracks neither this variable nor the assertion below.
var pkgOnly any = defaultUser

// PkgOnlyAsserted takes the type back at package level.
var PkgOnlyAsserted = pkgOnly.(User)
