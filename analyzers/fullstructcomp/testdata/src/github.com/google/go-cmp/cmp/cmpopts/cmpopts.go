// Package cmpopts fakes github.com/google/go-cmp/cmp/cmpopts for the
// fixtures of rule G12.
package cmpopts

import "github.com/google/go-cmp/cmp"

// IgnoreFields skips the named fields of one struct type.
func IgnoreFields(typ any, names ...string) cmp.Option { return nil }

// IgnoreUnexported skips the unexported fields of the given types.
func IgnoreUnexported(typs ...any) cmp.Option { return nil }
