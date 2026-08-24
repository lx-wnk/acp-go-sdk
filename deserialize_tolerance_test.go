package acp

import (
	"encoding/json"
	"testing"
)

// The schema marks 249 properties x-deserialize-default-on-error: a peer may send one in
// a shape this build cannot decode, and the message should lose that property rather than
// be rejected. Without it a peer one protocol revision away drops whole messages.
func TestDeserializeTolerance_BadShapeLosesOnlyThatProperty(t *testing.T) {
	var loc ToolCallLocation
	if err := json.Unmarshal([]byte(`{"path":"/x","line":"not-a-number"}`), &loc); err != nil {
		t.Fatalf("a tolerable property rejected the whole message: %v", err)
	}
	if loc.Path != "/x" {
		t.Fatalf("the surrounding message was not preserved: %+v", loc)
	}
	if loc.Line != nil {
		t.Fatalf("the undecodable property should be absent, got %v", *loc.Line)
	}
}

// Tolerance must not weaken a property the schema does not mark.
func TestDeserializeTolerance_UnmarkedPropertyStillFails(t *testing.T) {
	var req CreateTerminalRequest
	if err := json.Unmarshal([]byte(`{"sessionId":123,"command":"ls"}`), &req); err == nil {
		t.Fatal("a bad value for an unmarked property was accepted")
	}
}

// A valid payload must decode exactly as before: the tolerant path runs only after the
// ordinary decode has already failed.
func TestDeserializeTolerance_ValidPayloadUnaffected(t *testing.T) {
	var loc ToolCallLocation
	if err := json.Unmarshal([]byte(`{"path":"/x","line":42}`), &loc); err != nil {
		t.Fatalf("valid payload rejected: %v", err)
	}
	if loc.Line == nil || *loc.Line != 42 {
		t.Fatalf("valid value not decoded: %+v", loc)
	}
}

// The motivating case: a typed map whose value arrives in the wrong type used to reject
// the enclosing message outright.
func TestDeserializeTolerance_TypedMapValueOfWrongType(t *testing.T) {
	var req CreateTerminalRequest
	if err := json.Unmarshal([]byte(`{"sessionId":"s","command":"ls","env":{"FOO":123}}`), &req); err != nil {
		t.Fatalf("a bad env value rejected the whole request: %v", err)
	}
	if req.Command != "ls" {
		t.Fatalf("the surrounding request was not preserved: %+v", req)
	}
}
