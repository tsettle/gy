// Unit tests for gy's internal path parsing and node manipulation.
// Run with: go test ./...

package main

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestSplitPath(t *testing.T) {
	cases := []struct {
		pattern string
		want    []string
	}{
		{"", nil},
		{"a", []string{"a"}},
		{"a.b.c", []string{"a", "b", "c"}},
		{"a[0]", []string{"a", "[0]"}},
		{"a[0].b", []string{"a", "[0]", "b"}},
		{"[0]", []string{"[0]"}},
		{"[0][1]", []string{"[0]", "[1]"}},
		{"a..b", []string{"a", "b"}},
		{"users[0].roles[1]", []string{"users", "[0]", "roles", "[1]"}},
		{"special_keys.1", []string{"special_keys", "1"}},
	}

	for _, tc := range cases {
		t.Run(tc.pattern, func(t *testing.T) {
			got := splitPath(tc.pattern)
			if !stringSlicesEqual(got, tc.want) {
				t.Errorf("splitPath(%q) = %v, want %v", tc.pattern, got, tc.want)
			}
		})
	}
}

func stringSlicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

const sampleYAML = `
app:
  name: MyApp
  debug: false
database:
  host: localhost
  port: 5432
  credentials:
    user: admin
    password: secret
services:
  - name: web
    port: 8080
  - name: api
    port: 3000
`

func mustParse(t *testing.T, src string) *yaml.Node {
	t.Helper()
	var node yaml.Node
	if err := yaml.Unmarshal([]byte(src), &node); err != nil {
		t.Fatalf("failed to parse fixture YAML: %v", err)
	}
	return &node
}

func marshal(t *testing.T, node *yaml.Node) string {
	t.Helper()
	if node == nil {
		return "<nil>"
	}
	out, err := yaml.Marshal(node)
	if err != nil {
		t.Fatalf("failed to marshal node: %v", err)
	}
	return string(out)
}

func TestExtractPath(t *testing.T) {
	root := mustParse(t, sampleYAML)

	cases := []struct {
		name    string
		pattern string
		want    string // trimmed scalar value; empty means "check kind instead"
	}{
		{"top level scalar", "app.name", "MyApp"},
		{"nested scalar", "database.credentials.user", "admin"},
		{"leading dot", ".app.name", "MyApp"},
		{"array index then field", "services[0].name", "web"},
		{"second array element", "services[1].port", "3000"},
		{"boolean scalar", "app.debug", "false"},
		{"numeric scalar", "database.port", "5432"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := extractPath(root, tc.pattern)
			if got == nil {
				t.Fatalf("extractPath(%q) = nil, want scalar %q", tc.pattern, tc.want)
			}
			if got.Value != tc.want {
				t.Errorf("extractPath(%q) = %q, want %q", tc.pattern, got.Value, tc.want)
			}
		})
	}

	t.Run("root pattern returns whole document", func(t *testing.T) {
		got := extractPath(root, ".")
		if got == nil || got.Kind != yaml.DocumentNode {
			t.Fatalf("extractPath(\".\") = %v, want the document node", got)
		}
	})

	t.Run("empty pattern returns whole document", func(t *testing.T) {
		got := extractPath(root, "")
		if got == nil || got.Kind != yaml.DocumentNode {
			t.Fatalf("extractPath(\"\") = %v, want the document node", got)
		}
	})

	t.Run("missing key returns nil", func(t *testing.T) {
		if got := extractPath(root, "app.nonexistent"); got != nil {
			t.Errorf("extractPath(missing key) = %v, want nil", got)
		}
	})

	t.Run("missing nested path returns nil", func(t *testing.T) {
		if got := extractPath(root, "database.credentials.token"); got != nil {
			t.Errorf("extractPath(missing nested key) = %v, want nil", got)
		}
	})

	t.Run("array index out of bounds returns nil", func(t *testing.T) {
		if got := extractPath(root, "services[5]"); got != nil {
			t.Errorf("extractPath(out of bounds index) = %v, want nil", got)
		}
	})

	t.Run("negative array index returns nil", func(t *testing.T) {
		if got := extractPath(root, "services[-1]"); got != nil {
			t.Errorf("extractPath(negative index) = %v, want nil", got)
		}
	})

	t.Run("indexing into a mapping returns nil", func(t *testing.T) {
		if got := extractPath(root, "app[0]"); got != nil {
			t.Errorf("extractPath(index into mapping) = %v, want nil", got)
		}
	})

	t.Run("field access into a sequence returns nil", func(t *testing.T) {
		if got := extractPath(root, "services.name"); got != nil {
			t.Errorf("extractPath(field into sequence) = %v, want nil", got)
		}
	})

	t.Run("nested array element", func(t *testing.T) {
		nested := mustParse(t, "matrix:\n  - [1, 2]\n  - [3, 4]\n")
		got := extractPath(nested, "matrix[1][0]")
		if got == nil || got.Value != "3" {
			t.Errorf("extractPath(matrix[1][0]) = %v, want 3", got)
		}
	})

	t.Run("numeric-looking string keys are matched as strings", func(t *testing.T) {
		typesDoc := mustParse(t, "special_keys:\n  1: \"numeric key\"\n  \"key.with.dots\": value\n")
		got := extractPath(typesDoc, "special_keys.1")
		if got == nil || got.Value != "numeric key" {
			t.Errorf("extractPath(special_keys.1) = %v, want %q", got, "numeric key")
		}
	})
}

func TestWrapInPath(t *testing.T) {
	root := mustParse(t, sampleYAML)

	t.Run("wraps scalar back in its mapping ancestry", func(t *testing.T) {
		extracted := extractPath(root, "app.name")
		wrapped := wrapInPath(root, "app.name", extracted)
		got := marshal(t, wrapped)
		want := "app:\n    name: MyApp\n"
		if got != want {
			t.Errorf("wrapInPath(app.name) =\n%s\nwant:\n%s", got, want)
		}
	})

	t.Run("array index reconstruction pads preceding elements with null", func(t *testing.T) {
		extracted := extractPath(root, "services[1].name")
		wrapped := wrapInPath(root, "services[1].name", extracted)
		got := marshal(t, wrapped)
		// Documents current behavior: gy fabricates `null` placeholders for
		// skipped indices when reconstructing the ancestor path. This means
		// the output implies two real null entries exist at services[0],
		// which they don't in the source document.
		want := "services:\n    - null\n    - name: api\n"
		if got != want {
			t.Errorf("wrapInPath(services[1].name) =\n%s\nwant:\n%s", got, want)
		}
	})

	t.Run("unparseable index falls back to the extracted node", func(t *testing.T) {
		extracted := extractPath(root, "app.name")
		wrapped := wrapInPath(root, "app[bad].name", extracted)
		if wrapped != extracted {
			t.Errorf("wrapInPath with unparseable index should return the extracted node unchanged")
		}
	})
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}
	os.Stdout = w

	fn()

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatalf("failed to read captured stdout: %v", err)
	}
	return buf.String()
}

func TestListNode(t *testing.T) {
	root := mustParse(t, sampleYAML)

	t.Run("lists mapping keys at depth 1", func(t *testing.T) {
		target := extractPath(root, "database")
		out := captureStdout(t, func() {
			listNode(target, "", 1, 0)
		})
		want := "host\nport\ncredentials\n"
		if out != want {
			t.Errorf("listNode(database, depth=1) = %q, want %q", out, want)
		}
	})

	t.Run("unlimited depth (0) recurses fully", func(t *testing.T) {
		target := extractPath(root, "database")
		out := captureStdout(t, func() {
			listNode(target, "", 0, 0)
		})
		if !strings.Contains(out, "credentials\n") || !strings.Contains(out, "  user\n") {
			t.Errorf("listNode(database, depth=0) did not recurse into credentials, got %q", out)
		}
	})

	t.Run("lists sequence indices", func(t *testing.T) {
		target := extractPath(root, "services")
		out := captureStdout(t, func() {
			listNode(target, "", 1, 0)
		})
		want := "[0]\n[1]\n"
		if out != want {
			t.Errorf("listNode(services, depth=1) = %q, want %q", out, want)
		}
	})

	t.Run("scalar node lists nothing", func(t *testing.T) {
		target := extractPath(root, "app.name")
		out := captureStdout(t, func() {
			listNode(target, "", 0, 0)
		})
		if out != "" {
			t.Errorf("listNode(scalar) = %q, want empty output", out)
		}
	})
}

func TestExtractPathHandlesYAMLAnchorsAndAliases(t *testing.T) {
	// yaml.Node is a low-level AST: merge keys (<<) and aliases (*name) are
	// NOT resolved the way they would be by yaml.Unmarshal into a struct or
	// map. This test documents that current, real limitation rather than an
	// assumption about how anchors "should" behave.
	doc := mustParse(t, `
defaults: &defaults
  pool: 5
production:
  <<: *defaults
  pool: 25
regions: &regions
  - us-east-1
  - us-west-2
deployment:
  active_regions: *regions
`)

	t.Run("merge key is left unresolved as a literal mapping entry", func(t *testing.T) {
		got := extractPath(doc, "production")
		out := marshal(t, got)
		if !strings.Contains(out, "<<: *defaults") {
			t.Errorf("expected unresolved merge key in output, got:\n%s", out)
		}
	})

	t.Run("alias is left unresolved as a literal '*name' scalar", func(t *testing.T) {
		got := extractPath(doc, "deployment.active_regions")
		if got == nil {
			t.Fatalf("extractPath(deployment.active_regions) = nil")
		}
		out := marshal(t, got)
		if strings.TrimSpace(out) != "*regions" {
			t.Errorf("expected unresolved alias literal '*regions', got:\n%s", out)
		}
	})
}
