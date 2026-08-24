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
