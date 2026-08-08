package chaincode

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// TestDeterministicRecordID_U8CoverageGap closes a narrow, genuine unit-test
// gap found by the QA-1 audit (qa-tests/unit-coverage-report.md): before
// this file, deterministicRecordID (record_profile_section.go) had NO
// direct unit test of its own. Every existing RecordProfileSection test
// (record_profile_section_test.go) only asserts res.RecordID is non-empty
// — never that it is stable for a repeated txID, never that two different
// txIDs cannot collide, never anything about its actual shape.
//
// This test does NOT close, and does not attempt to close, test-strategy.md
// U-8's literal text — "RecordID is a fresh UUID v4" — because the shipped
// implementation deliberately does not generate a random UUID v4.
// record_profile_section.go's own doc comment on deterministicRecordID
// explains why: Fabric endorsement requires every endorsing peer to
// simulate the same transaction to a byte-identical write set
// [docs: chaincode4ade.rst#technical-problem], which a crypto/rand-based
// UUID v4 cannot satisfy across independently-executing peers. The function
// instead derives a UUID-*shaped* (RFC 4122 §4.3, version-5, name-based)
// identifier from GetTxID(), which IS identical across every endorsing peer
// for one transaction. The source already flags this as "[FLAGGED
// DEVIATION — confirm before treating as final]"; this audit's
// unit-coverage-report.md reports it again as a stale/mismatched premise
// (U-8's spec text vs. the ratified determinism constraint) rather than
// silently reinterpreting U-8 to match the code.
//
// What this test DOES do: pin the ACTUAL behavior the determinism
// constraint requires, so a future refactor that reintroduces true
// randomness (breaking cross-peer endorsement matching) fails here, fast
// and locally, instead of only surfacing on a live multi-peer network.
func TestDeterministicRecordID_U8CoverageGap(t *testing.T) {
	t.Run("same txID always yields the same RecordID (required for cross-peer endorsement matching)", func(t *testing.T) {
		id1 := deterministicRecordID("tx-abc-123")
		id2 := deterministicRecordID("tx-abc-123")
		require.Equal(t, id1, id2, "deterministicRecordID must be a pure function of txID — non-determinism here would break endorsement matching (chaincode4ade.rst#technical-problem)")
	})

	t.Run("different txIDs yield different RecordIDs", func(t *testing.T) {
		id1 := deterministicRecordID("tx-abc-123")
		id2 := deterministicRecordID("tx-def-456")
		require.NotEqual(t, id1, id2, "distinct transactions must not collide on RecordID")
	})

	t.Run("output is UUID-shaped RFC 4122, version 5 -- NOT version 4, confirming the documented deviation from U-8's literal text rather than silently matching it", func(t *testing.T) {
		id := deterministicRecordID("tx-abc-123")
		parsed, err := uuid.Parse(id)
		require.NoError(t, err, "deterministicRecordID must produce a syntactically valid UUID string")
		require.Equal(t, uuid.Version(5), parsed.Version(), "deterministicRecordID is deliberately version-5 (name-based/deterministic), not version-4 (random) -- see record_profile_section.go's FLAGGED DEVIATION comment; this is the regression guard for that documented, deliberate departure from U-8's literal spec text")
	})
}
