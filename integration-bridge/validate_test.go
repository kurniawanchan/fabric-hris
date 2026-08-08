package main

import (
	"encoding/base64"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func newValidateRequest(t *testing.T, body string) *http.Request {
	t.Helper()
	return httptest.NewRequest(http.MethodPost, "/v1/profile-sections/PERSONAL", strings.NewReader(body))
}

// callValidateRequest wraps validateRequest with a fresh httptest.NewRecorder
// as the ResponseWriter every call site needs for http.MaxBytesReader.
func callValidateRequest(t *testing.T, req *http.Request) (employeeInternalID, userID string, newValue, document []byte, err error) {
	t.Helper()
	return validateRequest(httptest.NewRecorder(), req)
}

func TestValidateRequest_ValidEnvelope_Succeeds(t *testing.T) {
	req := newValidateRequest(t, `{"employeeInternalID":"emp-1","userID":"user-1","newValue":{"fullName":"Test Employee"}}`)

	employeeInternalID, userID, newValue, document, err := callValidateRequest(t, req)
	if err != nil {
		t.Fatalf("validateRequest() unexpected error: %v", err)
	}
	if employeeInternalID != "emp-1" {
		t.Errorf("employeeInternalID = %q, want %q", employeeInternalID, "emp-1")
	}
	if userID != "user-1" {
		t.Errorf("userID = %q, want %q", userID, "user-1")
	}
	if string(newValue) != `{"fullName":"Test Employee"}` {
		t.Errorf("newValue = %s, want exact byte-for-byte match of the input JSON", newValue)
	}
	if document != nil {
		t.Errorf("document = %v, want nil (none provided)", document)
	}
}

func TestValidateRequest_LargeInteger_PreservedByteForByte(t *testing.T) {
	// A 19-digit bank-account-shaped number, well past float64's 2^53 exact-
	// integer boundary. If validateRequest ever unmarshals newValue into
	// map[string]interface{}/interface{} before re-marshaling, this digit
	// string would silently change. json.RawMessage must never let that
	// happen -- this is the exact mechanism protecting DataHash (FR-2/AC#4).
	const largeInt = "9223372036854775807"
	body := `{"employeeInternalID":"emp-1","userID":"user-1","newValue":{"bankAccountNumber":` + largeInt + `}}`
	req := newValidateRequest(t, body)

	_, _, newValue, _, err := callValidateRequest(t, req)
	if err != nil {
		t.Fatalf("validateRequest() unexpected error: %v", err)
	}
	if !strings.Contains(string(newValue), largeInt) {
		t.Errorf("newValue = %s, want it to contain the exact digit string %q untouched", newValue, largeInt)
	}
}

func TestValidateRequest_MissingEmployeeInternalID_Rejected(t *testing.T) {
	req := newValidateRequest(t, `{"userID":"user-1","newValue":{}}`)

	_, _, _, _, err := callValidateRequest(t, req)
	assertValidationError(t, err)
}

func TestValidateRequest_MissingUserID_Rejected(t *testing.T) {
	req := newValidateRequest(t, `{"employeeInternalID":"emp-1","newValue":{}}`)

	_, _, _, _, err := callValidateRequest(t, req)
	assertValidationError(t, err)
}

func TestValidateRequest_MissingNewValue_Rejected(t *testing.T) {
	req := newValidateRequest(t, `{"employeeInternalID":"emp-1","userID":"user-1"}`)

	_, _, _, _, err := callValidateRequest(t, req)
	assertValidationError(t, err)
}

func TestValidateRequest_InvalidJSON_Rejected(t *testing.T) {
	req := newValidateRequest(t, `{not valid json`)

	_, _, _, _, err := callValidateRequest(t, req)
	assertValidationError(t, err)
}

func TestValidateRequest_ValidDocument_DecodedCorrectly(t *testing.T) {
	want := []byte("supporting document bytes")
	encoded := base64.StdEncoding.EncodeToString(want)
	req := newValidateRequest(t, `{"employeeInternalID":"emp-1","userID":"user-1","newValue":{},"document":"`+encoded+`"}`)

	_, _, _, document, err := callValidateRequest(t, req)
	if err != nil {
		t.Fatalf("validateRequest() unexpected error: %v", err)
	}
	if string(document) != string(want) {
		t.Errorf("document = %q, want %q", document, want)
	}
}

func TestValidateRequest_InvalidDocumentBase64_Rejected(t *testing.T) {
	req := newValidateRequest(t, `{"employeeInternalID":"emp-1","userID":"user-1","newValue":{},"document":"not-valid-base64!!!"}`)

	_, _, _, _, err := callValidateRequest(t, req)
	assertValidationError(t, err)
}

// TestValidateRequest_PaddedIdentifiers_TrimmedBeforeReturn is the fix for a
// code review finding: the emptiness check used TrimSpace, but the raw,
// untrimmed values were returned -- a padded identifier would silently
// produce a different downstream pseudonym/hash than its trimmed form.
func TestValidateRequest_PaddedIdentifiers_TrimmedBeforeReturn(t *testing.T) {
	req := newValidateRequest(t, `{"employeeInternalID":"  emp-1  ","userID":"  user-1  ","newValue":{}}`)

	employeeInternalID, userID, _, _, err := callValidateRequest(t, req)
	if err != nil {
		t.Fatalf("validateRequest() unexpected error: %v", err)
	}
	if employeeInternalID != "emp-1" {
		t.Errorf("employeeInternalID = %q, want the trimmed %q", employeeInternalID, "emp-1")
	}
	if userID != "user-1" {
		t.Errorf("userID = %q, want the trimmed %q", userID, "user-1")
	}
}

// TestValidateRequest_NullNewValue_Rejected is the fix for a code review
// finding: len(env.NewValue) == 0 only catches full field omission, not the
// literal JSON null (a non-empty 4-byte json.RawMessage), which would
// otherwise anchor a meaningless section value.
func TestValidateRequest_NullNewValue_Rejected(t *testing.T) {
	req := newValidateRequest(t, `{"employeeInternalID":"emp-1","userID":"user-1","newValue":null}`)

	_, _, _, _, err := callValidateRequest(t, req)
	assertValidationError(t, err)
}

// TestValidateRequest_NonObjectNewValue_Rejected covers the same fix's other
// half: every profile section is conceptually a JSON object, so a bare
// string/number/array/bool is rejected too, not just null.
func TestValidateRequest_NonObjectNewValue_Rejected(t *testing.T) {
	for _, tc := range []string{`"a string"`, `42`, `[1,2,3]`, `true`} {
		req := newValidateRequest(t, `{"employeeInternalID":"emp-1","userID":"user-1","newValue":`+tc+`}`)

		_, _, _, _, err := callValidateRequest(t, req)
		if err == nil {
			t.Errorf("newValue=%s: got nil error, want rejection -- newValue must be a JSON object", tc)
			continue
		}
		assertValidationError(t, err)
	}
}

// TestValidateRequest_BodyExceedsLimit_Rejected is the fix for a code review
// finding: no size limit was applied anywhere in the request path, making
// the document-carrying endpoint an unmitigated memory-exhaustion vector.
func TestValidateRequest_BodyExceedsLimit_Rejected(t *testing.T) {
	oversizedDocument := strings.Repeat("a", maxRequestBodyBytes+1)
	body := `{"employeeInternalID":"emp-1","userID":"user-1","newValue":{},"document":"` + oversizedDocument + `"}`
	req := newValidateRequest(t, body)

	_, _, _, _, err := callValidateRequest(t, req)
	if err == nil {
		t.Fatal("validateRequest() got nil error for an oversized body, want rejection")
	}
	assertValidationError(t, err)
}

func assertValidationError(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Fatal("validateRequest() got nil error, want a *validationError")
	}
	var ve *validationError
	if !errors.As(err, &ve) {
		t.Errorf("validateRequest() error = %v, want it to be a *validationError", err)
	}
}
