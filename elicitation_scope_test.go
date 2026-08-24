package acp

import (
	"encoding/json"
	"testing"
)

// Elicitation scope arrives through a sibling anyOf on the mode schema, not through
// its properties. The arm structs were built from properties alone, so sessionId,
// toolCallId and requestId were dropped on decode and absent on re-encode — leaving
// a client unable to tell which session was asking, and any SDK-built proxy
// silently stripping the scope while forwarding.
func TestCreateElicitationRequest_SessionScopeRoundTrips(t *testing.T) {
	const in = `{"mode":"url","elicitationId":"e-1","message":"Sign in",` +
		`"url":"https://example.com/auth","sessionId":"sess-A","toolCallId":"tc-9"}`

	var req CreateElicitationRequest
	if err := json.Unmarshal([]byte(in), &req); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if req.Url == nil {
		t.Fatalf("expected the url arm, got %+v", req)
	}
	if req.Url.SessionId == nil || *req.Url.SessionId != "sess-A" {
		t.Fatalf("sessionId lost: %+v", req.Url.SessionId)
	}
	if req.Url.ToolCallId == nil || *req.Url.ToolCallId != "tc-9" {
		t.Fatalf("toolCallId lost: %+v", req.Url.ToolCallId)
	}
	if req.Url.RequestId != nil {
		t.Fatal("requestId must stay absent for a session-scoped elicitation")
	}

	out, err := json.Marshal(&req)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var got, want map[string]any
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatalf("re-unmarshal: %v", err)
	}
	if err := json.Unmarshal([]byte(in), &want); err != nil {
		t.Fatalf("unmarshal expected: %v", err)
	}
	for k, v := range want {
		if got[k] != v {
			t.Fatalf("field %q: got %v, want %v (full: %s)", k, got[k], v, out)
		}
	}
}

func TestCreateElicitationRequest_RequestScopeRoundTrips(t *testing.T) {
	const in = `{"mode":"form","message":"Fill this in","requestId":42,` +
		`"requestedSchema":{"type":"object","properties":{}}}`

	var req CreateElicitationRequest
	if err := json.Unmarshal([]byte(in), &req); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if req.Form == nil {
		t.Fatalf("expected the form arm, got %+v", req)
	}
	if req.Form.RequestId == nil {
		t.Fatal("requestId lost")
	}
	if req.Form.SessionId != nil {
		t.Fatal("sessionId must stay absent for a request-scoped elicitation")
	}

	out, err := json.Marshal(&req)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatalf("re-unmarshal: %v", err)
	}
	if got["requestId"] != float64(42) {
		t.Fatalf("requestId lost on re-encode: %s", out)
	}
	if _, ok := got["sessionId"]; ok {
		t.Fatalf("absent scope field was emitted: %s", out)
	}
}
