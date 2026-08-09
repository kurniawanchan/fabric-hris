package pipeline

import "context"

// bridgectx is the Stage Contract (ARCHITECTURE-SPINE.md AD-1): a single
// request-scoped struct carrying identifiers and results across the
// pipeline's stages, so that identifiers stay available even on paths with
// no typed error to unwrap them from (e.g. the [dispatch] timeout path,
// AD-4, has no *writepaths.PartialFailureError to source them from).
//
// Today, RegisterProfileSectionRoute (handlers.go) achieves this by holding
// bc as a closure variable and mutating/reading it directly -- classify()
// and respond() receive what they need as ordinary parameters, not via
// context retrieval. withBridgeCtx/bridgeCtxFrom below exist so a future
// logging stage (none exists yet in this codebase) can retrieve identifiers
// from ctx without needing its own parameter threaded through every call
// site; they are not currently called from production code, only from this
// file's own test (Story 1.2 code review, Decision 2).
//
// The cmd/integrationbridge + internal/pipeline package split was evaluated
// as a possible reason to finally wire bridgeCtxFrom in, and the answer was
// no: the split puts all 5 pipeline stages in this same package, so there is
// no new boundary for it to cross that would justify retrieving bc from ctx
// instead of the closure variable already in hand. It remains intentionally
// unused until a future stage (e.g. logging/observability) actually needs to
// read the context back out.
type bridgectx struct {
	EmployeeInternalID string
	ProfileSection     string
	TenantID           string
	Result             []byte
	Err                error
}

type bridgectxKey struct{}

// withBridgeCtx attaches bc to ctx, retrievable via bridgeCtxFrom.
func withBridgeCtx(ctx context.Context, bc *bridgectx) context.Context {
	return context.WithValue(ctx, bridgectxKey{}, bc)
}

// bridgeCtxFrom retrieves the *bridgectx attached by withBridgeCtx. Panics
// if none was attached, so a future caller of this function fails loudly at
// the seam where the mistake was made rather than silently running without
// the identifiers it expected.
func bridgeCtxFrom(ctx context.Context) *bridgectx {
	bc, ok := ctx.Value(bridgectxKey{}).(*bridgectx)
	if !ok {
		panic("integrationbridge: bridgectx not attached to context -- every stage after [auth] must run inside withBridgeCtx")
	}
	return bc
}
