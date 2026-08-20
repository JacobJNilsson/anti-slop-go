// This file holds the type-parameter walk that two rules share. Go
// reads and builds a value of a parameterized type through an interface
// only. A widening of such a value is therefore no choice of the
// author, and G05 and G06 both leave it alone. One implementation of
// that test cannot drift from itself.

package signature

import "go/types"

// MentionsTypeParam reports whether a type parameter appears in a type.
//
// The walk covers the type constructors that an assertion and a
// conversion can name: a named type carries its type arguments, a map
// carries two types, a struct, a tuple, and a signature carry their
// members, and every other constructor carries one element type.
func MentionsTypeParam(t types.Type) bool {
	switch x := types.Unalias(t).(type) {
	case *types.TypeParam:
		return true
	case *types.Named:
		args := x.TypeArgs()
		return mentionsAny(args.Len(), args.At)
	case *types.Map:
		return MentionsTypeParam(x.Key()) || MentionsTypeParam(x.Elem())
	case *types.Struct:
		return mentionsAny(x.NumFields(), func(i int) types.Type { return x.Field(i).Type() })
	case *types.Tuple:
		return mentionsAny(x.Len(), func(i int) types.Type { return x.At(i).Type() })
	case *types.Signature:
		return MentionsTypeParam(x.Params()) || MentionsTypeParam(x.Results())
	case interface{ Elem() types.Type }:
		// A pointer, a slice, an array, and a channel.
		return MentionsTypeParam(x.Elem())
	}

	return false
}

// mentionsAny reports whether a type parameter appears in one of the n
// types that at returns.
func mentionsAny(n int, at func(int) types.Type) bool {
	for i := range n {
		if MentionsTypeParam(at(i)) {
			return true
		}
	}

	return false
}
