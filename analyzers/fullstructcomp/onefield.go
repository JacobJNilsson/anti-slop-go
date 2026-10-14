package fullstructcomp

import (
	"go/ast"
	"go/constant"
	"go/types"
	"strings"

	"golang.org/x/tools/go/analysis"
)

// cmpPath and cmpoptsPath are the packages of the cmp module that the
// second report resolves. The analyzer reads the import path and never
// the name of an import.
const (
	cmpPath     = "github.com/google/go-cmp/cmp"
	cmpoptsPath = "github.com/google/go-cmp/cmp/cmpopts"
)

// The two messages of the second report. The first one names the
// single field that the diff still compares. The second one covers an
// ignore list that drops every exported field.
const (
	oneFieldMessage = "cmp.Diff compares only the field %s after cmpopts.IgnoreFields; " +
		"assert that field with a plain assertion"
	noFieldMessage = "cmp.Diff compares no field after cmpopts.IgnoreFields; " +
		"remove the diff or assert the field the test needs with a plain assertion"
)

// reportOneField reports the cmp.Diff calls of one function declaration
// whose ignore list leaves one exported field or none. Such a diff
// states the claim of a plain assertion in the long form.
func reportOneField(pass *analysis.Pass, fn *ast.FuncDecl) {
	info := pass.TypesInfo

	ast.Inspect(fn, func(n ast.Node) bool {
		call, isCall := n.(*ast.CallExpr)
		if !isCall || !calls(info, call, cmpPath, "Diff") || len(call.Args) < 3 || call.Ellipsis.IsValid() {
			return true
		}
		operand := comparedStruct(info, call.Args[0], call.Args[1])
		if operand == nil {
			return true
		}
		left, removed, counted := remainingFields(info, operand, call.Args[2:])
		switch {
		case !counted || removed == 0 || len(left) > 1:
		case len(left) == 0:
			pass.ReportRangef(call, "%s", noFieldMessage)
		default:
			pass.ReportRangef(call, oneFieldMessage, left[0].Name())
		}

		return true
	})
}

// comparedStruct returns the type that two operands hold after the
// removal of one pointer. It returns nil if the operands hold different
// types or a type that is no struct.
func comparedStruct(info *types.Info, want, got ast.Expr) types.Type {
	wantType, gotType := unpointer(info.TypeOf(want)), unpointer(info.TypeOf(got))
	if !types.Identical(wantType, gotType) {
		return nil
	}
	if _, isStruct := wantType.Underlying().(*types.Struct); !isStruct {
		return nil
	}

	return wantType
}

// unpointer removes one pointer from a type.
func unpointer(t types.Type) types.Type {
	if pointer, isPointer := types.Unalias(t).(*types.Pointer); isPointer {
		return pointer.Elem()
	}

	return t
}

// remainingFields returns the exported fields that the IgnoreFields
// options on the operand type leave, and the number of exported fields
// they drop. The last result is false if a name is no constant or no
// field of the struct.
func remainingFields(info *types.Info, operand types.Type, options []ast.Expr) ([]*types.Var, int, bool) {
	structure, _ := operand.Underlying().(*types.Struct)
	dropped := make(map[string]bool)
	for _, option := range options {
		call, isCall := ast.Unparen(option).(*ast.CallExpr)
		if !isCall || !calls(info, call, cmpoptsPath, "IgnoreFields") {
			continue
		}
		if !types.Identical(info.TypeOf(call.Args[0]), operand) {
			continue
		}
		for _, arg := range call.Args[1:] {
			value := info.Types[arg].Value
			if value == nil {
				return nil, 0, false
			}
			name := constant.StringVal(value)
			if !strings.Contains(name, ".") && !hasField(structure, name) {
				return nil, 0, false
			}
			dropped[name] = true
		}
	}

	var left []*types.Var
	removed := 0
	for field := range structure.Fields() {
		switch {
		case !field.Exported():
		case dropped[field.Name()]:
			removed++
		default:
			left = append(left, field)
		}
	}

	return left, removed, true
}

// hasField reports whether a struct declares a field itself. A name
// that only a promoted field carries drops a field of the embedded
// value, which this count does not read.
func hasField(structure *types.Struct, name string) bool {
	for field := range structure.Fields() {
		if field.Name() == name {
			return true
		}
	}

	return false
}

// calls reports whether a call names a function of a package. The
// lookup resolves the object, so a renamed import still matches.
func calls(info *types.Info, call *ast.CallExpr, path, name string) bool {
	sel, isSelector := ast.Unparen(call.Fun).(*ast.SelectorExpr)
	if !isSelector {
		return false
	}
	fn, isFunc := info.Uses[sel.Sel].(*types.Func)

	return isFunc && fn.Name() == name && fn.Pkg() != nil && fn.Pkg().Path() == path
}
