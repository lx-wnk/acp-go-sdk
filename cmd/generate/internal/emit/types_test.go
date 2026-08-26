package emit

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/coder/acp-go-sdk/cmd/generate/internal/load"
)

func mustDefinition(t *testing.T, src string) *load.Definition {
	t.Helper()
	var d load.Definition
	if err := json.Unmarshal([]byte(src), &d); err != nil {
		t.Fatal(err)
	}
	return &d
}

// additionalProperties: false and additionalProperties: true both decode to a bool schema
// carrying no type information, so the emitter used to treat a closed object exactly like
// an open one and hand back map[string]any. No schema in this repository uses the closed
// form, so the wrong type would appear the first time one did, silently.
func TestJenTypeForRejectsClosedObject(t *testing.T) {
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("a closed object was emitted instead of stopping the generator")
		}
		if msg, ok := r.(string); !ok || !strings.Contains(msg, "additionalProperties: false") {
			t.Fatalf("panic does not name the construct: %v", r)
		}
	}()
	jenTypeFor(mustDefinition(t, `{"type":"object","additionalProperties":false}`))
}

// The open form is the common case — every _meta field uses it — and must stay untyped.
func TestJenTypeForOpenObjectStaysUntyped(t *testing.T) {
	got := renderType(t, mustDefinition(t, `{"type":"object","additionalProperties":true}`))
	if got != "map[string]any" {
		t.Fatalf("open object emitted as %q", got)
	}
}

// A schema-valued additionalProperties still names the element type.
func TestJenTypeForTypedMapKeepsItsValueType(t *testing.T) {
	got := renderType(t, mustDefinition(t, `{"type":"object","additionalProperties":{"type":"string"}}`))
	if got != "map[string]string" {
		t.Fatalf("typed map emitted as %q", got)
	}
}

// An absent additionalProperties is the third form and must behave like the open one.
func TestJenTypeForAbsentAdditionalPropertiesStaysUntyped(t *testing.T) {
	got := renderType(t, mustDefinition(t, `{"type":"object"}`))
	if got != "map[string]any" {
		t.Fatalf("absent additionalProperties emitted as %q", got)
	}
}

func renderType(t *testing.T, d *load.Definition) string {
	t.Helper()
	return strings.TrimSpace(fmt.Sprintf("%#v", jenTypeFor(d)))
}
