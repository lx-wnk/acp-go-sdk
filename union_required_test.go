package acp

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

// Once the discriminator names an arm, a missing required key is a malformed payload,
// not a reason to fall through. The check runs against the raw key set because after
// decoding into the struct an absent required field and a zero value look identical.
func TestCreateElicitationRequest_RequiredKeys(t *testing.T) {
	tests := []struct {
		name    string
		payload string
		wantErr string
		wantArm func(CreateElicitationRequest) bool
	}{
		{
			name:    "url arm without its required keys",
			payload: `{"mode":"url","message":"Sign in"}`,
			wantErr: "requires elicitationId",
		},
		{
			name:    "form arm without its required keys",
			payload: `{"mode":"form","message":"m"}`,
			wantErr: "requires requestedSchema",
		},
		{
			name:    "complete url arm",
			payload: `{"mode":"url","message":"m","elicitationId":"e1","url":"https://example.com"}`,
			wantArm: func(r CreateElicitationRequest) bool { return r.Url != nil },
		},
		{
			name:    "complete form arm",
			payload: `{"mode":"form","message":"m","requestedSchema":{"type":"object","properties":{}}}`,
			wantArm: func(r CreateElicitationRequest) bool { return r.Form != nil },
		},
		{
			// The catch-all has no required keys beyond the discriminator, so an
			// unrecognised mode must still be accepted rather than rejected.
			name:    "unknown mode still reaches the catch-all",
			payload: `{"mode":"_custom","message":"m"}`,
			wantArm: func(r CreateElicitationRequest) bool { return r.Other != nil },
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var req CreateElicitationRequest
			err := json.Unmarshal([]byte(tc.payload), &req)

			if tc.wantErr != "" {
				if err == nil {
					t.Fatalf("expected an error, got %+v", req)
				}
				if !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("error %q does not mention %q", err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !tc.wantArm(req) {
				t.Fatalf("wrong arm: %+v", req)
			}
		})
	}
}

// The generated dispatch turns the decode error into InvalidParams, so a malformed
// elicitation request is refused on the wire instead of reaching a handler with
// empty required fields.
func TestClientDispatch_RejectsIncompleteElicitation(t *testing.T) {
	conn := &ClientSideConnection{client: &clientFuncs{}}
	_, reqErr := conn.handle(context.Background(), ClientMethodElicitationCreate,
		json.RawMessage(`{"mode":"url","message":"Sign in"}`))
	if reqErr == nil {
		t.Fatal("expected a request error for a payload missing required keys")
	}
	if reqErr.Code != -32602 {
		t.Fatalf("got code %d, want -32602 (invalid params)", reqErr.Code)
	}
}

// The parent schema's required list applies to every arm. CreateElicitationRequest
// declares message there while each arm declares only mode, so an enforcement site that
// reads the arm's own list never checks it and a payload with no message at all decoded
// clean.
func TestCreateElicitationRequest_ParentRequiredKeyIsEnforced(t *testing.T) {
	var req CreateElicitationRequest
	err := json.Unmarshal([]byte(`{"mode":"url","elicitationId":"e1","url":"https://example.com"}`), &req)
	if err == nil {
		t.Fatal("a payload missing the parent-required message decoded without error")
	}
	if !strings.Contains(err.Error(), "message") {
		t.Fatalf("error does not name the missing key: %v", err)
	}
}

// A required key sent as an explicit null is present but carries nothing. Treating
// presence as satisfaction decoded it to the zero value, indistinguishable from a value
// the peer actually sent.
func TestCreateElicitationRequest_ExplicitNullDoesNotSatisfyRequired(t *testing.T) {
	for _, in := range []string{
		`{"mode":"url","message":"m","elicitationId":null,"url":"https://example.com"}`,
		`{"mode":"url","message":"m","elicitationId" :  null ,"url":"https://example.com"}`,
	} {
		var req CreateElicitationRequest
		err := json.Unmarshal([]byte(in), &req)
		if err == nil {
			t.Fatalf("an explicit null satisfied a required key: %s", in)
		}
		if !strings.Contains(err.Error(), "elicitationId") {
			t.Fatalf("error does not name the key: %v", err)
		}
	}
}

// Tightening the check must not reject payloads the schema accepts.
func TestCreateElicitationRequest_ValidPayloadsStillDecode(t *testing.T) {
	for _, in := range []string{
		`{"mode":"url","message":"m","elicitationId":"e1","url":"https://example.com"}`,
		`{"mode":"url","message":"m","elicitationId":"e1","url":"https://example.com","sessionId":"s1"}`,
	} {
		var req CreateElicitationRequest
		if err := json.Unmarshal([]byte(in), &req); err != nil {
			t.Fatalf("valid payload rejected: %s: %v", in, err)
		}
	}
	// The catch-all arm has no discriminator value of its own and stays exempt.
	var other CreateElicitationRequest
	if err := json.Unmarshal([]byte(`{"mode":"_custom","message":"m"}`), &other); err != nil {
		t.Fatalf("catch-all payload rejected: %v", err)
	}
	if other.Other == nil {
		t.Fatal("expected the catch-all arm")
	}
}
