// Package pipeline implements ARCHITECTURE-SPINE.md's AD-1 5-stage pipeline
// -- [auth] -> [validate] -> [dispatch] -> [map-error] -> [respond] -- as a
// package boundary distinct from cmd/integrationbridge's composition root.
//
// This package must never import gatewayclient, ipfsclient, or keystore --
// those are infrastructure, and AD-1's pipeline stages are this system's
// application/orchestration layer, not a place that should know how a
// dependency is dialed or stored. Verify with:
//
//	go list -f '{{join .Imports "\n"}}' ./internal/pipeline | grep -E '^(gatewayclient|ipfsclient|keystore)$'
//
// (must find nothing).
//
// writepaths is the one permitted sibling import, used only by classify.go
// for its *writepaths.PartialFailureError check -- that is depending on this
// system's domain core (the write-path error vocabulary), not
// infrastructure, which is allowed. See classify.go's own doc comment for
// why no further inversion is available there.
package pipeline
