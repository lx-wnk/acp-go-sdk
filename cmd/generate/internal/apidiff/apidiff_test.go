package apidiff

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func write(t *testing.T, dir, name, src string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestSnapshotAndCompare(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "types_gen.go", `package acp
type Kept struct{ A string }
type Gone struct{ A string }
type Reshaped struct{ A string }
type Client interface{ Old(x int) error }
func (k Kept) Method() {}
`)
	before, err := Snapshot(dir, []string{"types_gen.go"})
	if err != nil {
		t.Fatal(err)
	}

	write(t, dir, "types_gen.go", `package acp
type Kept struct{ A string }
type Added struct{ B string }
type Reshaped struct{ A string; B string }
type Client interface{ Old(x int) error; New(y int) error }
func (k Kept) Method() {}
`)
	after, err := Snapshot(dir, []string{"types_gen.go"})
	if err != nil {
		t.Fatal(err)
	}

	d := Compare(before, after)
	want := map[string][]string{
		"removed": {"Gone"},
		"added":   {"Added", "Client.New"},
		"changed": {"Reshaped"},
	}
	if strings.Join(d.Removed, ",") != strings.Join(want["removed"], ",") {
		t.Fatalf("removed = %v", d.Removed)
	}
	if strings.Join(d.Added, ",") != strings.Join(want["added"], ",") {
		t.Fatalf("added = %v", d.Added)
	}
	if strings.Join(d.Changed, ",") != strings.Join(want["changed"], ",") {
		t.Fatalf("changed = %v", d.Changed)
	}
}

// An interface gaining a method is the break a consumer actually hits, so it must not be
// invisible just because the interface type itself still exists.
func TestInterfaceMethodAdditionIsReported(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "client_gen.go", "package acp\ntype Client interface{ A() error }\n")
	before, _ := Snapshot(dir, []string{"client_gen.go"})
	write(t, dir, "client_gen.go", "package acp\ntype Client interface{ A() error; B() error }\n")
	after, _ := Snapshot(dir, []string{"client_gen.go"})
	d := Compare(before, after)
	if len(d.Added) != 1 || d.Added[0] != "Client.B" {
		t.Fatalf("expected Client.B to be reported added, got %v", d.Added)
	}
}

// An unchanged API writes nothing, and regenerating the same version twice must not stack
// duplicate entries.
func TestWriteIsIdempotentAndSkipsEmpty(t *testing.T) {
	dir := t.TempDir()
	if err := Write(dir, "1.0.0", Delta{}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "CHANGELOG.md")); !os.IsNotExist(err) {
		t.Fatal("an empty delta wrote a changelog entry")
	}
	d := Delta{Removed: []string{"Gone"}}
	for i := 0; i < 2; i++ {
		if err := Write(dir, "1.0.0", d); err != nil {
			t.Fatal(err)
		}
	}
	b, err := os.ReadFile(filepath.Join(dir, "CHANGELOG.md"))
	if err != nil {
		t.Fatal(err)
	}
	if n := strings.Count(string(b), "## 1.0.0"); n != 1 {
		t.Fatalf("expected one entry for the version, got %d", n)
	}
}
