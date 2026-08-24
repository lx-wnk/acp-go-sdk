// Package apidiff records what a schema bump does to the generated public API.
//
// The module version tracks the upstream schema tag, not the Go API, so Go's resolver
// reads v1.20.0 -> v1.21.0 as an ordinary minor upgrade and takes it automatically. A
// consumer then meets a removed type as a compile error with nothing to explain it. The
// generator is the only thing that knows every name it emits, on both sides of a bump,
// so it is the right place to write that down.
package apidiff

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// API maps an exported identifier to a short description of its shape, so a name that
// survives a bump in a different form is reported as changed rather than untouched.
type API map[string]string

// Snapshot reads the generated files already on disk and records their exported surface.
// Missing files are not an error: the first run has nothing to compare against.
func Snapshot(dir string, files []string) (API, error) {
	api := API{}
	fset := token.NewFileSet()
	for _, name := range files {
		path := filepath.Join(dir, name)
		src, err := os.ReadFile(path)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, err
		}
		file, err := parser.ParseFile(fset, path, src, parser.SkipObjectResolution)
		if err != nil {
			// A half-written or hand-edited file must not fail the generator; it only
			// costs this run its comparison.
			continue
		}
		collect(api, file)
	}
	return api, nil
}

func collect(api API, file *ast.File) {
	for _, decl := range file.Decls {
		switch d := decl.(type) {
		case *ast.GenDecl:
			for _, spec := range d.Specs {
				ts, ok := spec.(*ast.TypeSpec)
				if !ok || !ts.Name.IsExported() {
					continue
				}
				api[ts.Name.Name] = describe(ts.Type)
				if iface, ok := ts.Type.(*ast.InterfaceType); ok {
					for _, m := range iface.Methods.List {
						for _, n := range m.Names {
							if n.IsExported() {
								api[ts.Name.Name+"."+n.Name] = "interface method"
							}
						}
					}
				}
			}
		case *ast.FuncDecl:
			if d.Name == nil || !d.Name.IsExported() {
				continue
			}
			name := d.Name.Name
			if d.Recv != nil && len(d.Recv.List) == 1 {
				recv := strings.TrimPrefix(typeName(d.Recv.List[0].Type), "*")
				if recv == "" || !ast.IsExported(recv) {
					continue
				}
				name = recv + "." + name
			}
			api[name] = "func"
		}
	}
}

func describe(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.StructType:
		names := make([]string, 0, len(t.Fields.List))
		for _, f := range t.Fields.List {
			for _, n := range f.Names {
				if n.IsExported() {
					names = append(names, n.Name)
				}
			}
		}
		sort.Strings(names)
		return "struct{" + strings.Join(names, ",") + "}"
	case *ast.InterfaceType:
		return "interface"
	default:
		return typeName(expr)
	}
}

func typeName(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		return "*" + typeName(t.X)
	case *ast.SelectorExpr:
		return typeName(t.X) + "." + t.Sel.Name
	case *ast.ArrayType:
		return "[]" + typeName(t.Elt)
	case *ast.MapType:
		return "map[" + typeName(t.Key) + "]" + typeName(t.Value)
	default:
		return "type"
	}
}

// Delta is the difference between two snapshots.
type Delta struct {
	Added   []string
	Removed []string
	Changed []string
}

// Empty reports whether the bump left the public API untouched.
func (d Delta) Empty() bool {
	return len(d.Added) == 0 && len(d.Removed) == 0 && len(d.Changed) == 0
}

// Compare reports what changed between two snapshots.
func Compare(before, after API) Delta {
	var d Delta
	for name, shape := range after {
		prev, existed := before[name]
		switch {
		case !existed:
			d.Added = append(d.Added, name)
		case prev != shape:
			d.Changed = append(d.Changed, name)
		}
	}
	for name := range before {
		if _, kept := after[name]; !kept {
			d.Removed = append(d.Removed, name)
		}
	}
	sort.Strings(d.Added)
	sort.Strings(d.Removed)
	sort.Strings(d.Changed)
	return d
}

// Write prepends an entry for version to CHANGELOG.md, newest first. A run whose snapshot
// had nothing to compare against, or that changed nothing, writes no entry.
func Write(outDir, version string, d Delta) error {
	if version == "" || d.Empty() {
		return nil
	}
	path := filepath.Join(outDir, "CHANGELOG.md")
	existing, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	header := "## " + version + "\n"
	if strings.Contains(string(existing), "\n"+header) || strings.HasPrefix(string(existing), header) {
		// Regenerating the same version must not stack duplicate entries.
		return nil
	}

	var b strings.Builder
	b.WriteString(header)
	b.WriteString("\nGenerated from the ACP schema. The module version tracks the schema tag, so a\nminor bump may still require code changes.\n")
	section(&b, "Removed", d.Removed)
	section(&b, "Changed", d.Changed)
	section(&b, "Added", d.Added)
	b.WriteString("\n")

	body := string(existing)
	const title = "# Changelog\n"
	if body == "" {
		return os.WriteFile(path, []byte(title+"\n"+b.String()), 0o644)
	}
	body = strings.TrimPrefix(body, title)
	return os.WriteFile(path, []byte(title+"\n"+b.String()+strings.TrimLeft(body, "\n")), 0o644)
}

func section(b *strings.Builder, label string, names []string) {
	if len(names) == 0 {
		return
	}
	fmt.Fprintf(b, "\n### %s (%d)\n\n", label, len(names))
	for _, n := range names {
		fmt.Fprintf(b, "- `%s`\n", n)
	}
}
