package main

import (
	"context"
	"errors"

	writepaths "writepaths"
)

// authError marks a failure originating in the [auth] stage. Wrapping the
// underlying error in this type is the ONLY signal classify uses to route
// it to "error" -- never a string match, never a stage-order assumption.
type authError struct{ error }

func (e *authError) Unwrap() error { return e.error }

// validationError marks a failure originating in the [validate] stage.
type validationError struct{ error }

func (e *validationError) Unwrap() error { return e.error }

// classify implements AD-3's decision table -- the ONLY place in this
// module that decides a response's status. It classifies by error TYPE
// only; it never sources employeeInternalID/profileSection from err --
// this function answers "what happened", not "to whom", and doesn't even
// accept identifiers as a parameter. Callers (registerProfileSectionRoute)
// keep identifiers available separately (today via a closure variable, see
// pipeline.go's bridgectx) precisely because errors.As-unwrapping an error
// wouldn't work on the [dispatch] timeout path, which has no typed error to
// unwrap them from.
//
//	[auth] rejection                                        -> error
//	[validate] rejection                                    -> rejected
//	*writepaths.PartialFailureError (from anchor())          -> partial_failure
//	a [dispatch] timeout (context.DeadlineExceeded)          -> partial_failure
//	any other error (e.g. a plain Hooks.SaveSection failure) -> error
//	nil                                                      -> committed
func classify(err error) string {
	if err == nil {
		return "committed"
	}

	var ae *authError
	if errors.As(err, &ae) {
		return "error"
	}

	var ve *validationError
	if errors.As(err, &ve) {
		return "rejected"
	}

	if errors.Is(err, context.DeadlineExceeded) {
		return "partial_failure"
	}

	var pfErr *writepaths.PartialFailureError
	if errors.As(err, &pfErr) {
		return "partial_failure"
	}

	return "error"
}
