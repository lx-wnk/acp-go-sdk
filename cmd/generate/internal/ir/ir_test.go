package ir

import (
	"encoding/json"
	"testing"

	"github.com/coder/acp-go-sdk/cmd/generate/internal/load"
)

func mustSchema(t *testing.T, src string) *load.Schema {
	t.Helper()
	var s load.Schema
	if err := json.Unmarshal([]byte(src), &s); err != nil {
		t.Fatal(err)
	}
	return &s
}

// ACP types most identifiers as allOf:[{$ref}] so a property can carry its own
// description. PrimaryType cannot see through that, which left every such required
// field without a Validate check.
func TestResolvedPrimaryType(t *testing.T) {
	schema := mustSchema(t, `{"$defs":{
		"SessionId": {"type":"string"},
		"Alias":     {"$ref":"#/$defs/SessionId"},
		"Blocks":    {"type":"array","items":{"type":"string"}},
		"Shape":     {"type":"object"},
		"Loop":      {"$ref":"#/$defs/Loop"}
	}}`)

	cases := []struct {
		name string
		prop string
		want string
	}{
		{"inline type", `{"type":"string"}`, "string"},
		{"nullable inline type", `{"type":["string","null"]}`, "string"},
		{"bare ref", `{"$ref":"#/$defs/SessionId"}`, "string"},
		{"allOf single ref", `{"description":"d","allOf":[{"$ref":"#/$defs/SessionId"}]}`, "string"},
		{"ref to ref", `{"allOf":[{"$ref":"#/$defs/Alias"}]}`, "string"},
		{"ref to array", `{"allOf":[{"$ref":"#/$defs/Blocks"}]}`, "array"},
		{"ref to object", `{"$ref":"#/$defs/Shape"}`, "object"},
		{"self-referencing ref has no primitive", `{"$ref":"#/$defs/Loop"}`, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var d load.Definition
			if err := json.Unmarshal([]byte(tc.prop), &d); err != nil {
				t.Fatal(err)
			}
			if got := ResolvedPrimaryType(schema, &d); got != tc.want {
				t.Fatalf("ResolvedPrimaryType(%s) = %q, want %q", tc.prop, got, tc.want)
			}
		})
	}
}

// A $ref naming a definition the schema does not contain is a broken schema, not a
// property without a type. Guessing would silently drop a required-field check again.
func TestResolvedPrimaryTypeRejectsDanglingRef(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("a dangling $ref resolved instead of stopping the generator")
		}
	}()
	ResolvedPrimaryType(mustSchema(t, `{"$defs":{}}`), &load.Definition{Ref: "#/$defs/Missing"})
}
