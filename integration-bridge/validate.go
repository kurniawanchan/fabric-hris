package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
)

// requestEnvelope is the wire shape every route accepts. NewValue is
// json.RawMessage deliberately -- see validateRequest's own doc comment for
// why this is the one field that must never be decoded any other way.
type requestEnvelope struct {
	EmployeeInternalID string          `json:"employeeInternalID"`
	UserID             string          `json:"userID"`
	NewValue           json.RawMessage `json:"newValue"`
	Document           string          `json:"document"` // base64, optional
}

// maxRequestBodyBytes caps every request body this bridge accepts. This is
// a reasonable placeholder, not a ratified number (no AC/architecture doc
// specifies a real supporting-document size limit yet) -- without SOME cap,
// document (base64 file bytes) makes this endpoint an unmitigated
// memory-exhaustion vector, since the whole body is read into memory before
// any validation runs (code review finding).
const maxRequestBodyBytes = 10 << 20 // 10 MiB

// validateRequest parses and validates the request body. newValue is
// returned exactly as received -- json.RawMessage captures the raw,
// unparsed bytes of that JSON value, so a large integer (e.g. a PAYROLL
// bank-account number past float64's 2^53 exact-integer boundary) can never
// be silently altered here. Do not change this to decode newValue into
// map[string]interface{}/interface{} for field-level checks: that would
// force a parse-and-remarshal round trip through encoding/json, which
// converts every JSON number to float64 -- corrupting exactly the class of
// value ComputeDataHash later hashes byte-for-byte (see the story's own Dev
// Notes for the verified call chain).
//
// Any failure here is wrapped in *validationError, never a bare error --
// classify() relies on that type to route it to status "rejected".
func validateRequest(w http.ResponseWriter, r *http.Request) (employeeInternalID, userID string, newValue, document []byte, err error) {
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodyBytes)

	var env requestEnvelope
	if decErr := json.NewDecoder(r.Body).Decode(&env); decErr != nil {
		return "", "", nil, nil, &validationError{fmt.Errorf("integrationbridge: invalid request body: %w", decErr)}
	}

	employeeInternalID = strings.TrimSpace(env.EmployeeInternalID)
	if employeeInternalID == "" {
		return "", "", nil, nil, &validationError{errors.New("integrationbridge: employeeInternalID is required")}
	}
	userID = strings.TrimSpace(env.UserID)
	if userID == "" {
		return "", "", nil, nil, &validationError{errors.New("integrationbridge: userID is required")}
	}
	if !isJSONObject(env.NewValue) {
		return "", "", nil, nil, &validationError{errors.New("integrationbridge: newValue is required and must be a JSON object")}
	}

	var doc []byte
	if env.Document != "" {
		doc, err = base64.StdEncoding.DecodeString(env.Document)
		if err != nil {
			return "", "", nil, nil, &validationError{fmt.Errorf("integrationbridge: document is not valid base64: %w", err)}
		}
	}

	return employeeInternalID, userID, []byte(env.NewValue), doc, nil
}

// isJSONObject reports whether raw's first non-whitespace byte is '{' --
// enough to reject an omitted field (len 0), a literal null, and any other
// non-object JSON value (string/number/array/bool) without ever parsing the
// value itself, preserving the byte-for-byte guarantee validateRequest's own
// doc comment describes. Every profile section is conceptually a JSON
// object; a non-object newValue would anchor a meaningless section value.
func isJSONObject(raw json.RawMessage) bool {
	trimmed := bytes.TrimSpace(raw)
	return len(trimmed) > 0 && trimmed[0] == '{'
}
