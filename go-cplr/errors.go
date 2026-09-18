package sduicompiler

import "fmt"

// compileError carries a compiler failure through internal panics so the
// public entry points can surface it as an ordinary error with the exact
// message strings the conformance suite matches.
type compileError struct{ msg string }

func (e *compileError) Error() string { return e.msg }

// fail aborts the current compilation with a formatted message.
func fail(format string, args ...any) {
	panic(&compileError{msg: fmt.Sprintf(format, args...)})
}

// recoverCompile converts a compileError panic into *err; other panics
// propagate untouched.
func recoverCompile(err *error) {
	if r := recover(); r != nil {
		if ce, ok := r.(*compileError); ok {
			*err = ce
			return
		}
		panic(r)
	}
}
