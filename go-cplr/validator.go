package sduicompiler

import "strings"

// Validator groups the structural checks the composer applies while
// expanding component references.
type Validator struct{}

// validatorAssertArgsDeclared rejects arguments absent from a component
// variable declaration.
func validatorAssertArgsDeclared(vars []string, args *Object) {
	var unknown []string
	for _, name := range args.Keys() {
		if !containsString(vars, name) {
			unknown = append(unknown, name)
		}
	}
	if len(unknown) > 0 {
		fail("Component reference has unknown args: %s", strings.Join(unknown, ", "))
	}
}

// validatorAssertHoleDeclared rejects a variable slot absent from its
// component declaration.
func validatorAssertHoleDeclared(vars []string, name string) {
	if !containsString(vars, name) {
		fail("Component body has undeclared slot .%s", name)
	}
}

// validatorAssertNoCircularPath rejects a resolved file path re-entered on
// the active expansion chain.
func validatorAssertNoCircularPath(filePath string, visiting []string) {
	if containsString(visiting, filePath) {
		fail("Circular reference: %s", strings.Join(appendPath(visiting, filePath), " -> "))
	}
}

func containsString(items []string, target string) bool {
	for _, item := range items {
		if item == target {
			return true
		}
	}
	return false
}

// appendPath copies-then-appends so shared context slices are never mutated.
func appendPath(items []string, extra string) []string {
	out := make([]string, len(items)+1)
	copy(out, items)
	out[len(items)] = extra
	return out
}
