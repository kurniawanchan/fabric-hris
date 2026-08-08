package chaincode

import "errors"

// Sentinel errors, wrapped with context via fmt.Errorf("%w: ...", ErrX, ...)
// at each call site so callers can distinguish outcomes with errors.Is
// without parsing message strings.
//
// [design note] The Fabric docs' own repo style guide recommends
// github.com/pkg/errors for stack-trace-carrying errors
// [docs: error-handling.rst#general-overview]; this module instead uses the
// standard library's error-wrapping (`%w`) to keep this chaincode module
// dependency-light. This is a deliberate, flagged deviation from that
// convention, not an oversight — either approach satisfies "the returned
// message surfaces to the client, keep it actionable and free of secrets."
var (
	// ErrRecordNotFound is the genuinely-distinct "not found" outcome
	// (api-contracts.md "NotFound", FR-15): no record exists yet for the
	// requested (employeeID, profileSection) key. This must never be
	// collapsed into, or confused with, any other error — a caller
	// distinguishes "never anchored" from "anchored, but my local recompute
	// doesn't match" (the latter this contract cannot ever compute, since it
	// never receives a value to compare — that judgment is entirely the
	// caller's, per the client-side verification protocol).
	ErrRecordNotFound = errors.New("employeeprofilerecord: profile section record not found")

	// ErrStaleChainReference is returned when the submitted prevHash does not
	// match the current head's dataHash for (employeeID, profileSection) —
	// the caller's view of the chain is out of date (data-model.md §5.2,
	// api-contracts.md "Errors"). This is an application-level rejection the
	// caller resolves by re-reading the head and resubmitting — it is
	// deliberately NOT the same Go error value Fabric's own MVCC_READ_CONFLICT
	// produces at commit time (that is a ledger/gateway-level outcome, out of
	// this contract's control) — both resolve the same way, but this sentinel
	// covers only the chaincode-level check done against already-read state.
	ErrStaleChainReference = errors.New("employeeprofilerecord: stale chain reference (prevHash does not match current head)")

	// ErrUnauthorized is returned by the CID/MSP ABAC hook (CC-4, ADR-0005's
	// shape, FR-6, security-architecture.md T13) on every one of the four
	// functions — deny-by-default: a caller whose identity cannot be read via
	// CID, or whose MSP is not one this contract recognizes for the
	// operation attempted, is rejected before any state is read or written.
	ErrUnauthorized = errors.New("employeeprofilerecord: caller identity failed MSP/ABAC verification")

	// ErrInvalidArgument covers argument-shape rejections (an out-of-enum
	// profileSection, FR-8; a missing required field) — distinct from the
	// two outcomes above.
	ErrInvalidArgument = errors.New("employeeprofilerecord: invalid argument")
)
