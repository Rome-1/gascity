package scripts_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// buildEnv returns the env vars these tests must pass through to the shard
// runner: PATH for finding `go`, HOME for the runner's own temp dir, plus
// the Go build/toolchain locations so the child `go test` invocations can
// reuse the parent's caches and the resolved toolchain. Without these the
// child go test runs against an empty fake HOME, re-resolves and re-downloads
// the toolchain, and easily exceeds the test's outer timeout. These are
// build-system vars, not "provider env" — they're orthogonal to what the
// test is exercising.
//
// We query `go env` rather than reading the process env directly because
// these values are typically configured in `~/.config/go/env`, not exported
// as shell variables, so `os.LookupEnv` would miss them.
func buildEnv(t *testing.T) []string {
	t.Helper()
	env := []string{
		"PATH=" + os.Getenv("PATH"),
		"HOME=" + t.TempDir(),
		"GO_TEST_TIMEOUT=1m",
	}
	goVars := []string{"GOPATH", "GOCACHE", "GOMODCACHE", "GOROOT", "GOTOOLCHAIN"}
	cmd := exec.Command("go", append([]string{"env"}, goVars...)...)
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("resolve go env: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) != len(goVars) {
		t.Fatalf("go env returned %d lines, want %d: %q", len(lines), len(goVars), out)
	}
	for i, name := range goVars {
		value := strings.Trim(lines[i], "\"'")
		if value != "" {
			env = append(env, name+"="+value)
		}
	}
	// Forward TMPDIR explicitly; some sandboxes mount /tmp noexec and the
	// runner picks a writable+executable tmpdir based on this.
	if v, ok := os.LookupEnv("TMPDIR"); ok {
		env = append(env, "TMPDIR="+v)
	}
	return env
}

func TestGoTestShardPreservesAcceptanceAuthEnv(t *testing.T) {
	repoRoot := filepath.Dir(t.TempDir())
	if wd, err := os.Getwd(); err == nil {
		repoRoot = filepath.Dir(wd)
	}

	cmd := exec.Command(
		filepath.Join(repoRoot, "scripts", "test-go-test-shard"),
		"./scripts/testdata/test-go-test-shard/env_required",
		"1",
		"1",
	)
	cmd.Dir = repoRoot
	cmd.Env = append(buildEnv(t), "ANTHROPIC_AUTH_TOKEN=synthetic-token")

	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("test-go-test-shard failed: %v\n%s", err, out)
	}
}

func TestGoTestShardRunsWithoutPreservedProviderEnv(t *testing.T) {
	repoRoot := filepath.Dir(t.TempDir())
	if wd, err := os.Getwd(); err == nil {
		repoRoot = filepath.Dir(wd)
	}

	cmd := exec.Command(
		filepath.Join(repoRoot, "scripts", "test-go-test-shard"),
		"./scripts/testdata/test-go-test-shard/no_extra_env",
		"1",
		"1",
	)
	cmd.Dir = repoRoot
	cmd.Env = buildEnv(t)

	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("test-go-test-shard failed without preserved provider env: %v\n%s", err, out)
	}
}
