// Command jcsverify closes backlog pre-build gate PB-3 / gap G-24: pins a
// concrete RFC 8785 (JSON Canonicalization Scheme) Go library and verifies
// it byte-exact-matches the RFC's own published test vectors (test U-4).
//
// Library pinned: github.com/cyberphone/json-canonicalization
// Version pinned: v0.0.0-20241213102144-19d51d7fe467 (see go.mod/go.sum —
// this repo has no semver tags; the pseudo-version above is the exact pin).
// Maintained by Anders Rundgren, RFC 8785's co-author — the reference
// implementation, not a third-party reimplementation; ships parallel
// implementations in Go/Java/.NET/Python/JS all verified against the SAME
// 6 test vectors (arrays/french/structures/unicode/values/weird.json),
// giving REC-1's writer and any non-Go verifier (if one is ever built) a
// shared, independently-maintained reference rather than two divergent
// from-scratch implementations.
package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	"github.com/cyberphone/json-canonicalization/go/src/webpki.org/jsoncanonicalizer"
)

// testdataDir must point at the pinned library's own testdata/ directory —
// resolved via `go env GOMODCACHE` at build time is fragile across
// machines, so this tool expects it passed as argv[1] (the orchestrator
// resolves it once via a shell one-liner; see the verification run this
// tool's own report cites).
func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: jcsverify <path-to-json-canonicalization/testdata>")
		os.Exit(1)
	}
	testdata := os.Args[1]

	vectors := []string{"arrays.json", "french.json", "structures.json", "unicode.json", "values.json", "weird.json"}
	allPass := true
	for _, name := range vectors {
		input, err := os.ReadFile(filepath.Join(testdata, "input", name))
		if err != nil {
			fmt.Fprintln(os.Stderr, "FATAL reading input", name, err)
			os.Exit(1)
		}
		expected, err := os.ReadFile(filepath.Join(testdata, "output", name))
		if err != nil {
			fmt.Fprintln(os.Stderr, "FATAL reading expected output", name, err)
			os.Exit(1)
		}

		actual, err := jsoncanonicalizer.Transform(input)
		if err != nil {
			fmt.Printf("%-16s FAIL (transform error: %v)\n", name, err)
			allPass = false
			continue
		}

		// Byte-exact comparison is the actual claim PB-3/U-4 needs — not
		// "looks similar", not "same after re-parsing". If the library
		// trims/adds a trailing newline etc. relative to the fixture, this
		// will legitimately fail and that's the point of running it for
		// real rather than trusting the library's own README.
		match := bytes.Equal(actual, expected)
		status := "PASS"
		if !match {
			status = "FAIL"
			allPass = false
		}
		fmt.Printf("%-16s %s (%d bytes)\n", name, status, len(actual))
		if !match {
			fmt.Printf("  expected: %q\n  actual:   %q\n", truncate(expected, 200), truncate(actual, 200))
		}

		// Idempotency check (also part of the library's own test suite,
		// re-verified here rather than trusted): canonicalizing already-
		// canonical output must be a no-op.
		recycled, err := jsoncanonicalizer.Transform(actual)
		if err != nil || !bytes.Equal(recycled, actual) {
			fmt.Printf("  IDEMPOTENCY FAIL for %s\n", name)
			allPass = false
		}
	}

	if !allPass {
		fmt.Fprintln(os.Stderr, "\nFAIL: at least one RFC 8785 test vector did not byte-match — PB-3 CANNOT close on this library/version")
		os.Exit(1)
	}
	fmt.Println("\nSUCCESS: all 6 RFC 8785 test vectors byte-exact-match, idempotent on re-canonicalization — PB-3/G-24 closes on github.com/cyberphone/json-canonicalization@v0.0.0-20241213102144-19d51d7fe467")
}

func truncate(b []byte, n int) []byte {
	if len(b) <= n {
		return b
	}
	return b[:n]
}
