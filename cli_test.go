// End-to-end tests that build the gy binary and exercise it as a real user
// would: via subprocess, asserting exact stdout/stderr/exit code. This is
// the assertion-backed replacement for the eyeball-only test_examples.sh
// demo script.

package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// binPath is set by TestMain once the binary is built.
var binPath string

func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "gy-cli-test")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(dir)

	binPath = filepath.Join(dir, "gy")
	// Pin buildVersion to a fixed value so TestCLIFlagsAndVersion doesn't need
	// updating on every release tag - it's testing the -V flag mechanism, not
	// the current version number.
	build := exec.Command("go", "build", "-ldflags", "-X main.buildVersion=test", "-o", binPath, ".")
	build.Stdout = os.Stdout
	build.Stderr = os.Stderr
	if err := build.Run(); err != nil {
		panic("failed to build gy binary for CLI tests: " + err.Error())
	}

	os.Exit(m.Run())
}

type cliResult struct {
	stdout   string
	stderr   string
	exitCode int
}

func runCLI(t *testing.T, stdin string, args ...string) cliResult {
	t.Helper()
	cmd := exec.Command(binPath, args...)
	if stdin != "" {
		cmd.Stdin = bytes.NewBufferString(stdin)
	} else {
		cmd.Stdin = bytes.NewReader(nil) // always give a closed stdin, never block on the terminal
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			t.Fatalf("failed to run gy: %v", err)
		}
	}
	return cliResult{stdout: stdout.String(), stderr: stderr.String(), exitCode: exitCode}
}

func TestCLIExtractionAndListing(t *testing.T) {
	cases := []struct {
		name string
		args []string
		want string
	}{
		{"simple path", []string{"database.host", "test/simple.yml"}, "database:\n    host: localhost\n"},
		{"trim mode", []string{"-t", "database.port", "test/simple.yml"}, "5432\n"},
		{"nested mapping", []string{"database.credentials", "test/simple.yml"},
			"database:\n    credentials:\n        user: admin\n        password: secret123\n"},
		{"array index wrapped", []string{"users[0].name", "test/arrays.yml"}, "users:\n    - name: Alice\n"},
		{"array trim", []string{"-t", "users[1].email", "test/arrays.yml"}, "bob@example.com\n"},
		{"nested array trim", []string{"-t", "users[0].roles[0]", "test/arrays.yml"}, "admin\n"},
		{"list mode default depth", []string{"-l", "database", "test/simple.yml"}, "host\nport\ntimeout\ncredentials\n"},
		{"list mode with depth", []string{"-l", "--depth", "2", "modules", "test/snmp.yml"},
			"if_mib\n  walk\n  metrics\n  lookups\nsystem_mib\n  walk\n  metrics\nbgp_mib\n  walk\n  metrics\n"},
		{"deep snmp walk", []string{"-t", "modules.if_mib.walk[0]", "test/snmp.yml"}, "1.3.6.1.2.1.2.2.1.1\n"},
		{"kubernetes container name", []string{"spec.template.spec.containers[0].name", "test/kubernetes.yml"},
			"spec:\n    template:\n        spec:\n            containers:\n                - name: nginx\n"},
		{"kubernetes env value trim", []string{"-t", "spec.template.spec.containers[0].env[0].value", "test/kubernetes.yml"}, "example.com\n"},
		{"numbers preserve formatting", []string{"numbers", "test/types.yml"},
			"numbers:\n    integer: 42\n    float: 3.14159\n    negative: -100\n    scientific: 1.23e-4\n    octal: 0o755\n    hex: 0xFF\n"},
		{"numeric-looking key", []string{"-t", "special_keys.1", "test/types.yml"}, "\"numeric key\"\n"},
		{"booleans preserve yaml 1.1 forms", []string{"booleans", "test/types.yml"},
			"booleans:\n    true_value: true\n    false_value: false\n    yes_value: yes\n    no_value: no\n"},
		{"ansible vars by document index", []string{"-t", "[0].vars.app_name", "test/ansible.yml"}, "myapp\n"},
		{"ansible task name by index", []string{"-t", "[0].tasks[0].name", "test/ansible.yml"}, "Create application user\n"},
		{"ansible nested loop item", []string{"-t", "[0].tasks[7].loop[1]", "test/ansible.yml"}, "80\n"},
		{"ansible handler list", []string{"-l", "[0].handlers", "test/ansible.yml"}, "[0]\n[1]\n[2]\n[3]\n"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			res := runCLI(t, "", tc.args...)
			if res.exitCode != 0 {
				t.Fatalf("exit code = %d, want 0; stderr=%q", res.exitCode, res.stderr)
			}
			if res.stdout != tc.want {
				t.Errorf("stdout =\n%q\nwant:\n%q", res.stdout, tc.want)
			}
		})
	}
}

func TestCLIRealWorldFixtures(t *testing.T) {
	cases := []struct {
		name string
		args []string
		want string
	}{
		{"docker-compose nested condition", []string{"-t", "services.app.depends_on.db.condition", "test/docker-compose.yml"}, "service_healthy\n"},
		{"docker-compose service list", []string{"-l", "--depth", "1", "services", "test/docker-compose.yml"}, "web\napp\ndb\ncache\n"},
		{"github actions matrix entry", []string{"-t", "jobs.build.strategy.matrix.go-version[1]", "test/github-actions.yml"}, "\"1.21\"\n"},
		{"helm ingress host", []string{"ingress.hosts[0].host", "test/helm-values.yml"},
			"ingress:\n    hosts:\n        - host: app.example.com\n"},
		{"helm resource limit", []string{"-t", "resources.limits.memory", "test/helm-values.yml"}, "512Mi\n"},
		{"prometheus second scrape job", []string{"-t", "scrape_configs[1].job_name", "test/prometheus.yml"}, "kubernetes-pods\n"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			res := runCLI(t, "", tc.args...)
			if res.exitCode != 0 {
				t.Fatalf("exit code = %d, want 0; stderr=%q", res.exitCode, res.stderr)
			}
			if res.stdout != tc.want {
				t.Errorf("stdout =\n%q\nwant:\n%q", res.stdout, tc.want)
			}
		})
	}
}

func TestCLIFlowAndBlockStyle(t *testing.T) {
	t.Run("extracting from a JSON source preserves JSON's flow style", func(t *testing.T) {
		res := runCLI(t, "", "database.host", "test/config.json")
		if res.exitCode != 0 {
			t.Fatalf("exit code = %d, want 0; stderr=%q", res.exitCode, res.stderr)
		}
		want := "{\"database\": {\"host\": \"localhost\"}}\n"
		if res.stdout != want {
			t.Errorf("stdout = %q, want %q", res.stdout, want)
		}
	})

	t.Run("extracting an array element from JSON preserves flow style", func(t *testing.T) {
		res := runCLI(t, "", "services[1]", "test/config.json")
		if res.exitCode != 0 {
			t.Fatalf("exit code = %d, want 0; stderr=%q", res.exitCode, res.stderr)
		}
		want := "{\"services\": [\"api\"]}\n"
		if res.stdout != want {
			t.Errorf("stdout = %q, want %q", res.stdout, want)
		}
	})

	t.Run("--flow forces flow-style output on a block YAML source", func(t *testing.T) {
		res := runCLI(t, "", "--flow", "database", "test/simple.yml")
		if res.exitCode != 0 {
			t.Fatalf("exit code = %d, want 0; stderr=%q", res.exitCode, res.stderr)
		}
		want := "{database: {host: localhost, port: 5432, timeout: 30, credentials: {user: admin, password: secret123}}}\n"
		if res.stdout != want {
			t.Errorf("stdout = %q, want %q", res.stdout, want)
		}
	})

	t.Run("--block forces clean block-style output on a JSON source, no leftover quotes", func(t *testing.T) {
		res := runCLI(t, "", "--block", "database", "test/config.json")
		if res.exitCode != 0 {
			t.Fatalf("exit code = %d, want 0; stderr=%q", res.exitCode, res.stderr)
		}
		want := "database:\n    host: localhost\n    port: 5432\n    credentials:\n" +
			"        user: admin\n        password: secret123\n"
		if res.stdout != want {
			t.Errorf("stdout = %q, want %q", res.stdout, want)
		}
	})

	t.Run("-j short flag behaves like --flow", func(t *testing.T) {
		res := runCLI(t, "", "-j", "database", "test/simple.yml")
		if res.exitCode != 0 {
			t.Fatalf("exit code = %d, want 0; stderr=%q", res.exitCode, res.stderr)
		}
		want := "{database: {host: localhost, port: 5432, timeout: 30, credentials: {user: admin, password: secret123}}}\n"
		if res.stdout != want {
			t.Errorf("stdout = %q, want %q", res.stdout, want)
		}
	})

	t.Run("-y short flag behaves like --block", func(t *testing.T) {
		res := runCLI(t, "", "-y", "database", "test/config.json")
		if res.exitCode != 0 {
			t.Fatalf("exit code = %d, want 0; stderr=%q", res.exitCode, res.stderr)
		}
		want := "database:\n    host: localhost\n    port: 5432\n    credentials:\n" +
			"        user: admin\n        password: secret123\n"
		if res.stdout != want {
			t.Errorf("stdout = %q, want %q", res.stdout, want)
		}
	})

	t.Run("--flow and --block together is an error on stderr", func(t *testing.T) {
		res := runCLI(t, "", "--flow", "--block", "database", "test/simple.yml")
		if res.exitCode != 1 {
			t.Errorf("exit code = %d, want 1", res.exitCode)
		}
		if res.stdout != "" {
			t.Errorf("stdout = %q, want empty", res.stdout)
		}
		want := "Error: --flow/-j and --block/-y are mutually exclusive\n"
		if res.stderr != want {
			t.Errorf("stderr = %q, want %q", res.stderr, want)
		}
	})

	t.Run("mixing the long and short forms is still an error", func(t *testing.T) {
		res := runCLI(t, "", "-j", "-y", "database", "test/simple.yml")
		if res.exitCode != 1 {
			t.Errorf("exit code = %d, want 1", res.exitCode)
		}
		want := "Error: --flow/-j and --block/-y are mutually exclusive\n"
		if res.stderr != want {
			t.Errorf("stderr = %q, want %q", res.stderr, want)
		}
	})
}

func TestCLIStdinAndPiping(t *testing.T) {
	t.Run("reads from stdin when only a pattern is given", func(t *testing.T) {
		simple, err := os.ReadFile("test/simple.yml")
		if err != nil {
			t.Fatalf("failed to read fixture: %v", err)
		}
		res := runCLI(t, string(simple), "app.name")
		if res.exitCode != 0 {
			t.Fatalf("exit code = %d, want 0; stderr=%q", res.exitCode, res.stderr)
		}
		want := "app:\n    name: MyApp\n"
		if res.stdout != want {
			t.Errorf("stdout = %q, want %q", res.stdout, want)
		}
	})

	t.Run("no args round-trips stdin", func(t *testing.T) {
		res := runCLI(t, "a: 1\n")
		if res.exitCode != 0 {
			t.Fatalf("exit code = %d, want 0; stderr=%q", res.exitCode, res.stderr)
		}
		if res.stdout != "a: 1\n" {
			t.Errorf("stdout = %q, want %q", res.stdout, "a: 1\n")
		}
	})

	t.Run("single filename arg with no pattern round-trips the file", func(t *testing.T) {
		res := runCLI(t, "", "test/simple.yml")
		if res.exitCode != 0 {
			t.Fatalf("exit code = %d, want 0; stderr=%q", res.exitCode, res.stderr)
		}
		want := "app:\n    name: MyApp\n    version: 1.2.3\n    debug: false\n" +
			"database:\n    host: localhost\n    port: 5432\n    timeout: 30\n" +
			"    credentials:\n        user: admin\n        password: secret123\n" +
			"cache:\n    enabled: true\n    ttl: 3600\n"
		if res.stdout != want {
			t.Errorf("stdout =\n%q\nwant:\n%q", res.stdout, want)
		}
	})
}

func TestCLIFlagsAndVersion(t *testing.T) {
	res := runCLI(t, "", "-V")
	if res.exitCode != 0 {
		t.Fatalf("exit code = %d, want 0", res.exitCode)
	}
	if res.stdout != "gy version test\n" {
		t.Errorf("stdout = %q, want %q", res.stdout, "gy version test\n")
	}
}

func TestCLIErrorHandling(t *testing.T) {
	t.Run("more than two positional args prints usage on stderr and exits 1", func(t *testing.T) {
		res := runCLI(t, "", "a", "b", "c")
		if res.exitCode != 1 {
			t.Errorf("exit code = %d, want 1", res.exitCode)
		}
		if res.stdout != "" {
			t.Errorf("stdout = %q, want empty (usage error belongs on stderr)", res.stdout)
		}
		want := "Usage: gy [--trim|-t] [--list|-l] [--depth N] [--flow|-j] [--block|-y] [pattern] [filename]\n"
		if res.stderr != want {
			t.Errorf("stderr = %q, want %q", res.stderr, want)
		}
	})

	t.Run("unknown path exits 1 with a clean message on stderr", func(t *testing.T) {
		res := runCLI(t, "", "nope.nope", "test/simple.yml")
		if res.exitCode != 1 {
			t.Errorf("exit code = %d, want 1", res.exitCode)
		}
		if res.stdout != "" {
			t.Errorf("stdout = %q, want empty (error output belongs on stderr)", res.stdout)
		}
		if res.stderr != "Path not found: nope.nope\n" {
			t.Errorf("stderr = %q, want %q", res.stderr, "Path not found: nope.nope\n")
		}
	})

	t.Run("missing input file exits 1 with a clean message, no panic", func(t *testing.T) {
		res := runCLI(t, "", "a", "test/does-not-exist.yml")
		if res.exitCode != 1 {
			t.Errorf("exit code = %d, want 1", res.exitCode)
		}
		if bytes.Contains([]byte(res.stderr), []byte("panic:")) {
			t.Errorf("stderr = %q, should not contain a panic trace", res.stderr)
		}
		want := "Error: open test/does-not-exist.yml: no such file or directory\n"
		if res.stderr != want {
			t.Errorf("stderr = %q, want %q", res.stderr, want)
		}
		if res.stdout != "" {
			t.Errorf("stdout = %q, want empty", res.stdout)
		}
	})

	t.Run("malformed YAML exits 1 with a clean parse error, no panic", func(t *testing.T) {
		res := runCLI(t, "key: [unterminated", "a")
		if res.exitCode != 1 {
			t.Errorf("exit code = %d, want 1", res.exitCode)
		}
		if bytes.Contains([]byte(res.stderr), []byte("panic:")) {
			t.Errorf("stderr = %q, should not contain a panic trace", res.stderr)
		}
		if !bytes.Contains([]byte(res.stderr), []byte("Error: failed to parse YAML")) {
			t.Errorf("stderr = %q, want it to contain a clean parse error", res.stderr)
		}
	})
}
