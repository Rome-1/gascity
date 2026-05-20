package doctor

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gastownhall/gascity/internal/config"
)

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll(%q): %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile(%q): %v", path, err)
	}
}

func TestInstructionsFileCheck_NilConfig(t *testing.T) {
	r := NewInstructionsFileCheck(nil, "").Run(&CheckContext{})
	if r.Status != StatusOK {
		t.Fatalf("Status = %v, want StatusOK", r.Status)
	}
}

func TestInstructionsFileCheck_NoRigs(t *testing.T) {
	cfg := &config.City{Agents: []config.Agent{{Name: "a", Provider: "claude"}}}
	r := NewInstructionsFileCheck(cfg, t.TempDir()).Run(&CheckContext{})
	if r.Status != StatusOK {
		t.Fatalf("Status = %v, want StatusOK; details=%v", r.Status, r.Details)
	}
}

func TestInstructionsFileCheck_AllFilesPresent(t *testing.T) {
	city := t.TempDir()
	rig := filepath.Join(city, "rigs", "demo")
	writeFile(t, filepath.Join(rig, "AGENTS.md"), "agent instructions\n")
	writeFile(t, filepath.Join(rig, "CLAUDE.md"), "claude instructions\n")

	cfg := &config.City{
		Agents: []config.Agent{{Name: "a", Provider: "claude"}},
		Rigs:   []config.Rig{{Name: "demo", Path: rig}},
	}
	r := NewInstructionsFileCheck(cfg, city).Run(&CheckContext{})
	if r.Status != StatusOK {
		t.Fatalf("Status = %v, want StatusOK; details=%v", r.Status, r.Details)
	}
}

func TestInstructionsFileCheck_FlagsMissingButRecoverable(t *testing.T) {
	city := t.TempDir()
	rig := filepath.Join(city, "rigs", "demo")
	// Rig only ships CLAUDE.md; an agent on a non-Claude provider that
	// expects AGENTS.md should be flagged.
	writeFile(t, filepath.Join(rig, "CLAUDE.md"), "claude instructions\n")

	cfg := &config.City{
		Agents: []config.Agent{{Name: "a", Provider: "codex"}},
		Rigs:   []config.Rig{{Name: "demo", Path: rig}},
	}
	r := NewInstructionsFileCheck(cfg, city).Run(&CheckContext{})
	if r.Status != StatusWarning {
		t.Fatalf("Status = %v, want StatusWarning; details=%v", r.Status, r.Details)
	}
	if len(r.Details) != 1 {
		t.Fatalf("Details = %d, want 1: %v", len(r.Details), r.Details)
	}
	if !strings.Contains(r.Details[0], `rig "demo"`) || !strings.Contains(r.Details[0], "AGENTS.md") || !strings.Contains(r.Details[0], "CLAUDE.md") {
		t.Errorf("Details[0] missing expected components: %q", r.Details[0])
	}
	if !NewInstructionsFileCheck(cfg, city).CanFix() {
		t.Error("expected CanFix() = true")
	}
}

func TestInstructionsFileCheck_NoFallbackNoWarning(t *testing.T) {
	// A rig with no instruction files at all should not be flagged —
	// we cannot recover, and emitting a warning that the user cannot
	// act on is noise.
	city := t.TempDir()
	rig := filepath.Join(city, "rigs", "bare")
	if err := os.MkdirAll(rig, 0o755); err != nil {
		t.Fatal(err)
	}
	cfg := &config.City{
		Agents: []config.Agent{{Name: "a", Provider: "codex"}},
		Rigs:   []config.Rig{{Name: "bare", Path: rig}},
	}
	r := NewInstructionsFileCheck(cfg, city).Run(&CheckContext{})
	if r.Status != StatusOK {
		t.Fatalf("Status = %v, want StatusOK; details=%v", r.Status, r.Details)
	}
}

func TestInstructionsFileCheck_FixSymlinksFallback(t *testing.T) {
	city := t.TempDir()
	rig := filepath.Join(city, "rigs", "demo")
	writeFile(t, filepath.Join(rig, "CLAUDE.md"), "claude instructions\n")

	cfg := &config.City{
		Agents: []config.Agent{{Name: "a", Provider: "codex"}},
		Rigs:   []config.Rig{{Name: "demo", Path: rig}},
	}
	c := NewInstructionsFileCheck(cfg, city)
	if err := c.Fix(&CheckContext{}); err != nil {
		t.Fatalf("Fix() error: %v", err)
	}
	link, err := os.Readlink(filepath.Join(rig, "AGENTS.md"))
	if err != nil {
		t.Fatalf("AGENTS.md not symlinked: %v", err)
	}
	if link != "CLAUDE.md" {
		t.Errorf("AGENTS.md link target = %q, want CLAUDE.md", link)
	}
	// Re-run: check now passes.
	r := c.Run(&CheckContext{})
	if r.Status != StatusOK {
		t.Errorf("after Fix, Status = %v, want StatusOK; details=%v", r.Status, r.Details)
	}
}

func TestInstructionsFileCheck_RelativeRigPathResolves(t *testing.T) {
	city := t.TempDir()
	rigRel := filepath.Join("rigs", "rel")
	writeFile(t, filepath.Join(city, rigRel, "CLAUDE.md"), "x")

	cfg := &config.City{
		Agents: []config.Agent{{Name: "a", Provider: "codex"}},
		Rigs:   []config.Rig{{Name: "rel", Path: rigRel}},
	}
	r := NewInstructionsFileCheck(cfg, city).Run(&CheckContext{})
	if r.Status != StatusWarning {
		t.Fatalf("Status = %v, want StatusWarning; details=%v", r.Status, r.Details)
	}
}

func TestInstructionsFileCheck_DeterministicOrdering(t *testing.T) {
	city := t.TempDir()
	for _, name := range []string{"alpha", "zulu"} {
		writeFile(t, filepath.Join(city, "rigs", name, "CLAUDE.md"), "x")
	}
	cfg := &config.City{
		Agents: []config.Agent{{Name: "a", Provider: "codex"}},
		Rigs: []config.Rig{
			{Name: "zulu", Path: filepath.Join(city, "rigs", "zulu")},
			{Name: "alpha", Path: filepath.Join(city, "rigs", "alpha")},
		},
	}
	r := NewInstructionsFileCheck(cfg, city).Run(&CheckContext{})
	if r.Status != StatusWarning {
		t.Fatalf("Status = %v; details=%v", r.Status, r.Details)
	}
	if len(r.Details) != 2 {
		t.Fatalf("Details = %d, want 2: %v", len(r.Details), r.Details)
	}
	if !strings.Contains(r.Details[0], `rig "alpha"`) {
		t.Errorf("Details[0] should mention alpha first: %q", r.Details[0])
	}
	if !strings.Contains(r.Details[1], `rig "zulu"`) {
		t.Errorf("Details[1] should mention zulu second: %q", r.Details[1])
	}
}
