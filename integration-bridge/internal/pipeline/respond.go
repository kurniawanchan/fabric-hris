package pipeline

import (
	"encoding/json"
	"net/http"
)

// responseBody is the wire shape of every route's response (AD-3's
// Consistency Convention): status is the ONLY business-outcome channel;
// recordID is present only when status is "committed"; detail carries an
// error's message on any non-committed outcome. detail must never carry
// newValue/document bytes (AC#7) -- every error type that can reach here
// (authError, validationError, writepaths.PartialFailureError, a plain
// Hooks/SaveSection error, context.DeadlineExceeded) is constructed from
// fixed strings and pseudonymous identifiers, never from echoing request
// body content back.
type responseBody struct {
	Status   string `json:"status"`
	RecordID string `json:"recordID,omitempty"`
	Detail   string `json:"detail,omitempty"`
}

// chaincodeResult mirrors just enough of
// chaincode.RecordProfileSectionResult's JSON shape to extract recordID --
// result is that struct's raw JSON bytes, passed through unmodified from
// gatewayclient.SubmitRecordProfileSection. This bridge has no use for
// Timestamp/Version itself.
type chaincodeResult struct {
	RecordID string `json:"recordID"`
}

// httpStatusFor maps AD-3's body status field to a transport-only HTTP
// status code. "error" covers two origins classify() deliberately collapses
// (an [auth] credential fault and a bridge/dependency fault from
// Store.SaveSection) -- see grounding-gaps.md G-33 for why [respond] cannot
// tell them apart here, and why 500 (the bridge/dependency-fault half of
// AD-3's own wording) is the chosen default rather than 401.
func httpStatusFor(status string) int {
	switch status {
	case "committed", "partial_failure":
		return http.StatusOK
	case "rejected":
		return http.StatusBadRequest
	default: // "error"
		return http.StatusInternalServerError
	}
}

// respond is [respond] (AD-3/AD-6): it writes the response envelope and
// nothing else. status must already be [map-error]'s output -- respond
// itself never classifies anything except this one invariant check: a
// "committed" status with no usable recordID cannot be a real chaincode
// success (RecordProfileSectionResult always carries one) -- it can only
// come from a bug in a Hooks/DispatchFunc implementation. Reporting it as
// "committed" anyway would silently hide that bug behind an
// indistinguishable-from-success response (code review finding); reporting
// it as "error" surfaces it instead.
func respond(w http.ResponseWriter, status string, result []byte, err error) {
	body := responseBody{Status: status}

	if status == "committed" {
		var cr chaincodeResult
		if unmarshalErr := json.Unmarshal(result, &cr); unmarshalErr == nil && cr.RecordID != "" {
			body.RecordID = cr.RecordID
		} else {
			body.Status = "error"
			body.Detail = "integrationbridge: internal error -- dispatch reported success with no usable recordID"
		}
	}
	if err != nil {
		body.Detail = err.Error()
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(httpStatusFor(body.Status))
	_ = json.NewEncoder(w).Encode(body)
}
