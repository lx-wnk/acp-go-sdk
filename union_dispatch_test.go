package acp

import (
	"encoding/json"
	"strings"
	"testing"
)

// A discriminator value this build does not recognise belongs to the catch-all arm.
// Before the generator emitted a default case, the switch fell through to key-based
// matching and the first arm whose required keys were satisfied claimed the payload —
// for CreateElicitationResponse that is Accept, turning an unknown action into user
// consent.
func TestCreateElicitationResponse_UnknownActionUsesCatchAll(t *testing.T) {
	var resp CreateElicitationResponse
	if err := json.Unmarshal([]byte(`{"action":"_custom"}`), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Accept != nil {
		t.Fatal("an unrecognised action must not decode as Accept")
	}
	if resp.Other == nil {
		t.Fatalf("expected the catch-all arm, got %+v", resp)
	}

	out, err := json.Marshal(&resp)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if got := string(out); !strings.Contains(got, `"_custom"`) {
		t.Fatalf("re-marshal rewrote the action: %s", got)
	}
}

func TestCreateElicitationResponse_KnownActions(t *testing.T) {
	tests := []struct {
		action string
		check  func(CreateElicitationResponse) bool
	}{
		{"accept", func(r CreateElicitationResponse) bool { return r.Accept != nil }},
		{"decline", func(r CreateElicitationResponse) bool { return r.Decline != nil }},
		{"cancel", func(r CreateElicitationResponse) bool { return r.Cancel != nil }},
	}

	for _, tc := range tests {
		t.Run(tc.action, func(t *testing.T) {
			var resp CreateElicitationResponse
			if err := json.Unmarshal([]byte(`{"action":"`+tc.action+`"}`), &resp); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			if !tc.check(resp) {
				t.Fatalf("action %q reached the wrong arm: %+v", tc.action, resp)
			}
			if resp.Other != nil {
				t.Fatalf("action %q fell through to the catch-all", tc.action)
			}
		})
	}
}

// Same mechanism on a second union, where the wrong arm also rewrote the payload:
// an unknown property type used to come back as "string".
func TestElicitationPropertySchema_UnknownTypeUsesCatchAll(t *testing.T) {
	var schema ElicitationPropertySchema
	if err := json.Unmarshal([]byte(`{"type":"_future"}`), &schema); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if schema.String != nil {
		t.Fatal("an unrecognised type must not decode as a string property")
	}
	if schema.Other == nil {
		t.Fatalf("expected the catch-all arm, got %+v", schema)
	}

	out, err := json.Marshal(&schema)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if got := string(out); !strings.Contains(got, `"_future"`) {
		t.Fatalf("re-marshal rewrote the type: %s", got)
	}
}

// A union with no arm set cannot produce valid JSON. Returning empty bytes with a nil
// error left json.Marshal to fail with "unexpected end of JSON input", naming neither
// the type nor the cause.
func TestUnionMarshal_NoVariantSetNamesTheType(t *testing.T) {
	var resp CreateElicitationResponse
	_, err := resp.MarshalJSON()
	if err == nil {
		t.Fatal("expected an error when no variant is set")
	}
	if !strings.Contains(err.Error(), "CreateElicitationResponse") {
		t.Fatalf("error does not name the type: %v", err)
	}
}

// The catch-all arm is a variant with its own required keys, not a bucket. McpServer's
// catch-all is the stdio arm, whose Command and Args a host may hand to exec, so an
// unrecognised transport must not claim it on the strength of the discriminator alone.
//
// This check bounds the payload, it does not reject it: a caller that supplies every
// stdio required key still reaches the stdio arm under a foreign type. Closing that
// needs a dedicated extension arm, which the schema does not define for McpServer.
func TestMcpServer_UnknownTransportNeedsTheCatchAllRequiredKeys(t *testing.T) {
	var partial McpServer
	if err := json.Unmarshal([]byte(`{"type":"docker","name":"x","command":"/bin/sh"}`), &partial); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if partial.Stdio != nil {
		t.Fatalf("payload missing stdio required keys reached the stdio arm: %+v", partial.Stdio)
	}

	var complete McpServer
	if err := json.Unmarshal([]byte(`{"type":"docker","name":"x","command":"/bin/sh","args":[],"env":[]}`), &complete); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if complete.Stdio == nil {
		t.Fatal("a payload satisfying every stdio required key still reaches the stdio arm")
	}
}

// AuthMethod's catch-all is the agent arm, documented as the default when no type is
// present. An unrecognised type that carries neither of its required keys must not be
// reinterpreted as agent-handled authentication.
func TestAuthMethod_UnknownTypeNeedsTheCatchAllRequiredKeys(t *testing.T) {
	var m AuthMethod
	if err := json.Unmarshal([]byte(`{"type":"webauthn_future"}`), &m); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if m.Agent != nil {
		t.Fatalf("an unrecognised auth method decoded as agent-handled: %+v", m.Agent)
	}
}

// The catch-all arm's schema sets additionalProperties: true, and the type's own doc
// comment tells clients to preserve the raw payload when storing, replaying, proxying or
// forwarding. Building the wire form from the named fields alone dropped every key this
// build does not know, so the type could not honour that.
func TestCreateElicitationResponse_CatchAllPreservesUnknownKeys(t *testing.T) {
	const in = `{"action":"_custom","foo":"bar","nested":{"a":1}}`
	var resp CreateElicitationResponse
	if err := json.Unmarshal([]byte(in), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	out, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	for _, key := range []string{`"foo"`, `"bar"`, `"nested"`} {
		if !strings.Contains(string(out), key) {
			t.Fatalf("re-marshal dropped %s: %s", key, out)
		}
	}
}

// A named field still wins over the captured copy, so a caller can edit what it does
// understand without the original shadowing the change.
func TestCreateElicitationResponse_CatchAllFieldEditWins(t *testing.T) {
	var resp CreateElicitationResponse
	if err := json.Unmarshal([]byte(`{"action":"_custom","foo":"bar"}`), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	resp.Other.Action = "_edited"
	out, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if !strings.Contains(string(out), `"_edited"`) {
		t.Fatalf("the captured copy shadowed the edited field: %s", out)
	}
	if !strings.Contains(string(out), `"foo"`) {
		t.Fatalf("editing a field dropped the unknown keys: %s", out)
	}
}

// A value built in Go rather than decoded carries no captured copy, and must still
// marshal from its named fields.
func TestCreateElicitationResponse_CatchAllWithoutRawMarshals(t *testing.T) {
	resp := CreateElicitationResponse{Other: &CreateElicitationResponseOther{Action: "_built"}}
	out, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if !strings.Contains(string(out), `"_built"`) {
		t.Fatalf("unexpected wire form: %s", out)
	}
}

// A union holding two arms cannot be expressed in JSON. Validate permits it for anyOf,
// which is faithful to the schema, so marshal is the only place to report it — and
// silently encoding the first arm ships a message the caller believes it verified.
func TestUnionMarshal_MultipleVariantsSet(t *testing.T) {
	s := ElicitationContentValueString("hi")
	i := ElicitationContentValueInteger(7)
	v := ElicitationContentValue{String: &s, Integer: &i}

	if err := v.Validate(); err != nil {
		t.Fatalf("anyOf Validate should still accept multiple arms, got %v", err)
	}
	if _, err := v.MarshalJSON(); err == nil {
		t.Fatal("expected marshal to refuse a union with two arms set")
	} else if !strings.Contains(err.Error(), "multiple variants set") {
		t.Fatalf("unexpected error: %v", err)
	}
}
