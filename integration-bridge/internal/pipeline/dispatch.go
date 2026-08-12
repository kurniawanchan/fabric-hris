package pipeline

import (
	"context"
	"time"
)

// DispatchFunc matches the shape of every Hooks write-path method (e.g.
// writepaths.Hooks.UpdatePersonalData) -- dependency-inverted so [dispatch]
// is unit-testable against a deliberately slow/erroring stub, the same
// technique Story 1.1 used for newGatewayClientFunc.
type DispatchFunc func(ctx context.Context, employeeInternalID, userID, recordIdentity string, newValue, document []byte) ([]byte, error)

// dispatch is [dispatch] (AD-4): context.WithTimeout is the literal first
// statement, scoped to only this call -- it starts once [auth]/[validate]
// have already succeeded. fn runs synchronously; it is never raced against
// the timeout in a goroutine (AD-4's Rule: a detached still-running submit
// racing an already-sent HTTP response is a double-submission risk against
// the one shared, retained Gateway connection). Because fn (ultimately
// SubmitTransaction) accepts no ctx of its own, a fired deadline cannot
// forcibly abort it -- dispatch can only detect a timeout retrospectively,
// by checking dispatchCtx.Err() AFTER fn returns.
//
// A fired deadline only overrides fn's OWN error -- it never overrides a
// genuine success. Since dispatch already blocked synchronously for the
// call's entire real duration, a nil error means fn produced a definitive,
// correct result; discarding it would misreport a committed write as
// uncertain for no reason beyond "it took longer than budget" (Story 1.2
// code review, Decision 1). When fn itself also errors after the deadline
// fired, dispatch reports context.DeadlineExceeded regardless of fn's exact
// error text: classify() maps that to "partial_failure" (AC#6 -- never a
// distinct "timeout" status).
func dispatch(ctx context.Context, budget time.Duration, fn DispatchFunc, employeeInternalID, userID, recordIdentity string, newValue, document []byte) ([]byte, error) {
	dispatchCtx, cancel := context.WithTimeout(ctx, budget)
	defer cancel()

	result, err := fn(dispatchCtx, employeeInternalID, userID, recordIdentity, newValue, document)
	if err != nil && dispatchCtx.Err() == context.DeadlineExceeded {
		return result, context.DeadlineExceeded
	}
	return result, err
}
