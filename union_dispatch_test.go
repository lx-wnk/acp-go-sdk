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

// AuthMethod's catch-all is the agent arm, documented as the default when no type is
// present. An unrecognised type carrying neither of its required keys cannot be
// represented by any arm, so it is refused rather than reinterpreted as agent-handled
// authentication — the strongest claim the union can make about a method nobody asked for.
func TestAuthMethod_UnknownTypeWithoutAgentKeysIsRefused(t *testing.T) {
	var m AuthMethod
	err := json.Unmarshal([]byte(`{"type":"webauthn_future"}`), &m)
	if err == nil {
		t.Fatalf("an unrecognised auth method decoded silently: %+v", m)
	}
	if m.Agent != nil {
		t.Fatalf("an unrecognised auth method decoded as agent-handled: %+v", m.Agent)
	}
	if !strings.Contains(err.Error(), "AuthMethod") {
		t.Fatalf("error does not name the union: %v", err)
	}
}

// The agent arm is still reachable for a payload that satisfies it.
func TestAuthMethod_UnknownTypeWithAgentKeysStillRoutes(t *testing.T) {
	var m AuthMethod
	if err := json.Unmarshal([]byte(`{"type":"webauthn_future","id":"x","name":"y"}`), &m); err != nil {
		t.Fatalf("a payload satisfying the agent arm was refused: %v", err)
	}
	if m.Agent == nil {
		t.Fatalf("expected the agent arm, got %+v", m)
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

// Eight of the fifteen discriminated unions declare no catch-all arm. An unrecognised
// discriminator used to fall through to the key-match below, which handed the payload to
// whichever arm was declared first and whose required keys happened to be present — and
// then rewrote the discriminator on the way out. A video content block re-marshalled as
// "image". Refusing is visible; silently changing what the peer said is not.
func TestContentBlock_UnknownTypeIsRefused(t *testing.T) {
	var cb ContentBlock
	err := json.Unmarshal([]byte(`{"type":"video","data":"xyz","mimeType":"video/mp4"}`), &cb)
	if err == nil {
		t.Fatalf("an unrecognised content type decoded silently: %+v", cb)
	}
	if !strings.Contains(err.Error(), "ContentBlock") {
		t.Fatalf("error does not name the union: %v", err)
	}
	// The peer controls the value, so it must not appear in the message.
	if strings.Contains(err.Error(), "video") {
		t.Fatalf("error echoes the peer-supplied value: %v", err)
	}
}

// The refusal must not touch values this build does know.
func TestContentBlock_KnownTypesStillDecode(t *testing.T) {
	var cb ContentBlock
	if err := json.Unmarshal([]byte(`{"type":"image","data":"xyz","mimeType":"image/png"}`), &cb); err != nil {
		t.Fatalf("a valid content block was rejected: %v", err)
	}
	if cb.Image == nil {
		t.Fatalf("expected the image arm, got %+v", cb)
	}
}

// "Absent" and "present but empty" are different statements from the peer, and both
// produced the zero value. An explicit empty action therefore reached Accept — implied
// user consent from a message that expressed none. It belongs on the unrecognised-value
// path, because that is what it is.
func TestCreateElicitationResponse_EmptyActionIsNotConsent(t *testing.T) {
	var resp CreateElicitationResponse
	if err := json.Unmarshal([]byte(`{"action":""}`), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Accept != nil {
		t.Fatal("an empty action decoded as user consent")
	}
	if resp.Other == nil {
		t.Fatalf("expected the catch-all arm, got %+v", resp)
	}
}

// An absent discriminator identifies nothing, so it still resolves through key matching.
// This pins the boundary: the change is about presence, not about absence.
func TestCreateElicitationResponse_AbsentActionStillFallsThrough(t *testing.T) {
	var resp CreateElicitationResponse
	if err := json.Unmarshal([]byte(`{}`), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Accept == nil {
		t.Fatalf("absent-discriminator handling changed: %+v", resp)
	}
}

// Where a union has no catch-all arm, an unrecognised discriminator is refused. Where it
// has one whose required keys the payload does not satisfy, the value is just as
// unrepresentable — but it used to fall through to the key-match and be claimed by
// whichever arm happened to fit, landing on Http for an McpServer tagged "docker". The two
// paths now answer the same way.
func TestMcpServer_UnknownTransportWithoutStdioKeysIsRefused(t *testing.T) {
	for _, in := range []string{
		`{"type":"docker","name":"x","command":"/bin/sh"}`,
		`{"type":"docker"}`,
	} {
		var m McpServer
		err := json.Unmarshal([]byte(in), &m)
		if err == nil {
			t.Fatalf("payload %s decoded silently as %+v", in, m)
		}
		if !strings.Contains(err.Error(), "McpServer") {
			t.Fatalf("error does not name the union: %v", err)
		}
		if m.Http != nil {
			t.Fatalf("payload still reached the http arm: %+v", m.Http)
		}
	}
}

// The boundary is unchanged in both directions: a payload that does satisfy the catch-all
// arm still reaches it, and an absent discriminator still resolves through key matching.
func TestMcpServer_CatchAllBoundaryUnchanged(t *testing.T) {
	var complete McpServer
	if err := json.Unmarshal([]byte(`{"type":"docker","name":"x","command":"/bin/sh","args":[],"env":[]}`), &complete); err != nil {
		t.Fatalf("a payload satisfying every stdio required key was refused: %v", err)
	}
	if complete.Stdio == nil {
		t.Fatalf("expected the stdio arm, got %+v", complete)
	}

	var noDisc McpServer
	if err := json.Unmarshal([]byte(`{"name":"x","command":"/bin/sh","args":[],"env":[]}`), &noDisc); err != nil {
		t.Fatalf("absent-discriminator handling changed: %v", err)
	}
	if noDisc.Stdio == nil {
		t.Fatalf("expected key matching to resolve the arm, got %+v", noDisc)
	}
}

// A catch-all arm that requires nothing beyond the discriminator keeps accepting anything,
// which is what makes it a catch-all.
func TestCreateElicitationResponse_CatchAllWithoutRequiredKeysStillRoutes(t *testing.T) {
	var resp CreateElicitationResponse
	if err := json.Unmarshal([]byte(`{"action":"_custom","foo":"bar"}`), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Other == nil {
		t.Fatalf("expected the catch-all arm, got %+v", resp)
	}
}
