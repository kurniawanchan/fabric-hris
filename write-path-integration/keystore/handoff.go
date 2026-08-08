// INT-3: the salt / employeeKey_i hand-off channel FR-36 requires, kept
// structurally separate from gateway-client's Verify/VerificationResult per
// FR-14 (prd.md §7):
//
//	FR-14  "Endpoint verifikasi TIDAK BOLEH mengembalikan salt dalam bentuk
//	        apa pun, pada respons mana pun. Penyerahan salt kepada
//	        pemverifikasi berwenang berjalan lewat jalur terpisah (FR-36)."
//	FR-36  "Sistem HARUS menyediakan jalur terkendali dan ter-audit bagi
//	        pemverifikasi berwenang untuk memperoleh salt versi yang
//	        diperiksa — terpisah dari endpoint verifikasi (FR-14 tetap
//	        berlaku). Kewenangan mengikuti kepemilikan data: karyawan atas
//	        datanya sendiri, auditor sebatas lingkup auditnya."
//
// FR-14's own text is WHY this cannot be a field bolted onto
// gateway-client's VerificationResult — a shared response type is exactly
// the "endpoint verifikasi ... pada respons mana pun" FR-14 forbids
// carrying salt on. It has to be its own type, its own file, its own
// package boundary, reachable only through its own request path.
// gateway-client/verify.go's own doc-comment on Verify already points here
// by name ("the SEPARATE hand-off channel INT-3 defines") — this file is
// that channel.
//
// This backlog item's original text hedged its DoD against an unresolved
// question: whether S-4/PB-2 would land on an anchor-service architecture
// with an app-layer read-back surface (G-23), in which case a salt
// hand-off channel might have needed to reason about a second party
// holding copies of PII-adjacent material. That question is now moot:
// G-19, G-21, G-23, and PB-2 were all closed/dissolved by a 2026-08-06
// human sponsor decision (S-4), ratifying the in-band-recording
// architecture (ADR-0014) as final — there is no anchor-service, no Kafka,
// no cross-service read-back surface anywhere in this design. SaltHandoff
// below is written straightforwardly for that ratified shape; it does not
// hedge against an alternate architecture that no longer exists.
package keystore

import (
	"context"
	"errors"
	"fmt"
	"sync"
)

// ErrSaltAccessDenied is returned when a request is refused — either
// Authorize said no outright, or Authorize itself failed and RequestSalt
// treated that as a denial (fail closed, see below). It is deliberately
// distinct from whatever error SaltStore.GetSalt returns for a version that
// was never anchored: "denied" means a real thing existed and access to it
// was refused; "not found" means there was never a thing to access in the
// first place. Conflating the two would make the audit trail lie about how
// many actual unauthorized-access attempts occurred.
var ErrSaltAccessDenied = errors.New("keystore: salt access denied")

// AuthorizeSaltAccessFunc decides whether requesterID may obtain the salt
// for one specific (employeeID, profileSection, version). FR-36's
// authorization rule itself — "karyawan atas datanya sendiri, auditor
// sebatas lingkup auditnya" (the employee over their own record; an auditor
// within their audit's scope) — is a POLICY decision this package does not
// make, the same "interface accepted, policy owned elsewhere" boundary
// keystore.go already draws around custody/rotation for SaltStore,
// DocumentKeyStore, and EmployeeKeyStore. Injected by the caller who knows
// org structure/audit scope/session identity; never hardcoded here.
type AuthorizeSaltAccessFunc func(ctx context.Context, requesterID, employeeID, profileSection string, version int) (bool, error)

// AuditEntry is one recorded outcome of a salt-access request — the
// "ter-audit" half of FR-36's "jalur terkendali dan ter-audit" (a
// controlled AND audited path). Reason is human-readable context: empty on
// a grant, non-empty on a denial (what the denial cause was, or the
// underlying Authorize error's text if the policy check itself broke).
type AuditEntry struct {
	RequesterID    string
	EmployeeID     string
	ProfileSection string
	Version        int
	Granted        bool
	Reason         string
}

// AuditLog records every salt-access decision. It MUST be given both grants
// and denials, never wired up as a successes-only log — a denial here is
// itself a security-relevant event (an unauthorized access attempt against
// employee PII-adjacent material), arguably the more important half of
// what this interface exists to capture.
type AuditLog interface {
	Record(ctx context.Context, entry AuditEntry) error
}

// SaltHandoff is FR-36's controlled, audited path. Salts is the same
// SaltStore REC-2 already defines (no second copy of salt custody is
// created here); Authorize and Audit are the two policy/observability seams
// this package exposes without deciding their contents.
type SaltHandoff struct {
	Salts     SaltStore
	Authorize AuthorizeSaltAccessFunc
	Audit     AuditLog
}

// RequestSalt is the single entry point FR-36 describes.
//
// Authorization failures fail CLOSED: if Authorize itself returns an error
// (a downed policy backend, a broken dependency, anything), that is treated
// as a denial — never as an implicit grant. A policy check that fails open
// would turn an ordinary infrastructure outage into a confidentiality
// breach against PII-adjacent material, which is a strictly worse failure
// mode than an authorized verifier having to retry.
//
// A SaltStore.GetSalt failure for a genuinely-nonexistent salt (the
// requested version was never anchored) is NOT recorded as a denial in the
// audit log — see ErrSaltAccessDenied's doc-comment for why that distinction
// matters. It is propagated to the caller undecorated by this package's own
// denial semantics; RequestSalt still returns a non-nil error either way, so
// callers that only check "did I get a salt back" behave correctly without
// having to know the difference.
func (h *SaltHandoff) RequestSalt(ctx context.Context, requesterID, employeeID, profileSection string, version int) ([]byte, error) {
	ok, err := h.Authorize(ctx, requesterID, employeeID, profileSection, version)
	if err != nil {
		h.audit(ctx, requesterID, employeeID, profileSection, version, false, err.Error())
		return nil, fmt.Errorf("keystore: salt access authorization check failed, failing closed: %w", err)
	}
	if !ok {
		h.audit(ctx, requesterID, employeeID, profileSection, version, false,
			"requester is not the record's employee and not an in-scope auditor")
		return nil, ErrSaltAccessDenied
	}

	salt, err := h.Salts.GetSalt(ctx, employeeID, profileSection, version)
	if err != nil {
		// Deliberately un-audited as a denial: this is a "never anchored"
		// lookup miss, not a refusal of access to something that exists.
		return nil, err
	}

	h.audit(ctx, requesterID, employeeID, profileSection, version, true, "")
	return salt, nil
}

// audit is best-effort: a failure to WRITE the audit record is not folded
// into RequestSalt's own return value. Doing otherwise would make a
// logging-sink outage able to block legitimate salt access entirely — its
// own fail-open/fail-closed trade-off, and the wrong one to make here,
// since RequestSalt's actual security decision (Authorize) has already run
// and been recorded-or-attempted by the time this is called.
func (h *SaltHandoff) audit(ctx context.Context, requesterID, employeeID, profileSection string, version int, granted bool, reason string) {
	_ = h.Audit.Record(ctx, AuditEntry{
		RequesterID:    requesterID,
		EmployeeID:     employeeID,
		ProfileSection: profileSection,
		Version:        version,
		Granted:        granted,
		Reason:         reason,
	})
}

// InMemoryAuditLog is a reference AuditLog — demo-grade, like every other
// InMemory* type in this package: no persistence across process restarts,
// no tamper-evidence, no retention/export discipline (all out of this
// item's scope). Safe for concurrent use.
type InMemoryAuditLog struct {
	mu      sync.Mutex
	entries []AuditEntry
}

func NewInMemoryAuditLog() *InMemoryAuditLog {
	return &InMemoryAuditLog{}
}

func (l *InMemoryAuditLog) Record(_ context.Context, entry AuditEntry) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.entries = append(l.entries, entry)
	return nil
}

// Entries returns a snapshot copy, not the live slice — so a test reading
// the result can't race with a concurrent Record appending to it.
func (l *InMemoryAuditLog) Entries() []AuditEntry {
	l.mu.Lock()
	defer l.mu.Unlock()
	out := make([]AuditEntry, len(l.entries))
	copy(out, l.entries)
	return out
}
