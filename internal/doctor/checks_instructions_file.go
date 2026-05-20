package doctor

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/gastownhall/gascity/internal/config"
	"github.com/gastownhall/gascity/internal/fsys"
)

// instructionsFileFallbacks lists the common project-instruction filenames
// in priority order. Providers point at one canonical name via
// `ProviderSpec.InstructionsFile` (defaulting to "AGENTS.md" via
// ResolveProvider). When the canonical file is missing but a fallback is
// present in the rig, the agent silently starts without project context;
// this check surfaces that mismatch.
var instructionsFileFallbacks = []string{
	"AGENTS.md",
	"CLAUDE.md",
	"INSTRUCTIONS.md",
}

// InstructionsFileCheck warns when a rig hosts an agent whose provider's
// `InstructionsFile` is missing from the rig directory while a known
// fallback file (CLAUDE.md ↔ AGENTS.md ↔ INSTRUCTIONS.md) exists in the
// same directory. Silently starting an agent with no project instructions
// is one of the show-stoppers from the non-Claude provider parity audit
// (Gap 7 in docs/research/w-7ed35a727f-parity-audit-classification.md /
// gastownhall/gascity#672).
//
// The check is read-only by default. `gc doctor --fix` symlinks the
// existing fallback to the expected name — symlink rather than copy so
// the source of truth stays in one place.
//
// Scope:
//   - One warning per (rig × expected filename). The agent providers
//     populate the "expected filenames" set; the actual content of those
//     files is not inspected.
//   - Rigs with no Path are skipped (covered by the site-binding migration
//     work; this check fires once the binding lands).
//   - Empty fallback files are treated as present — if a user intentionally
//     ships empty AGENTS.md and CLAUDE.md, the check should not nag.
type InstructionsFileCheck struct {
	cfg      *config.City
	cityPath string
}

// NewInstructionsFileCheck creates a check that surfaces missing
// per-provider instruction files in each rig.
func NewInstructionsFileCheck(cfg *config.City, cityPath string) *InstructionsFileCheck {
	return &InstructionsFileCheck{cfg: cfg, cityPath: cityPath}
}

// Name returns the check identifier.
func (c *InstructionsFileCheck) Name() string { return "instructions-file" }

// Run reports any (rig, expected-filename) pairs where the file is missing
// but a known fallback exists. The fix hint is shown when at least one
// pair is auto-repairable via symlinking.
func (c *InstructionsFileCheck) Run(_ *CheckContext) *CheckResult {
	r := &CheckResult{Name: c.Name()}
	if c.cfg == nil {
		r.Status = StatusOK
		r.Message = "no config; nothing to check"
		return r
	}
	gaps := c.collectGaps()
	if len(gaps) == 0 {
		r.Status = StatusOK
		r.Message = "instructions files look complete"
		return r
	}
	details := make([]string, 0, len(gaps))
	for _, g := range gaps {
		details = append(details, fmt.Sprintf(
			"rig %q (%s): expected %s for provider %q is missing; %s is present",
			g.rigName, g.rigPath, g.expected, g.provider, g.fallback,
		))
	}
	r.Status = StatusWarning
	r.Message = fmt.Sprintf("%d instructions-file gap(s)", len(gaps))
	r.Details = details
	r.FixHint = "run `gc doctor --fix` to symlink the present fallback to the expected filename, or copy the file manually"
	return r
}

// CanFix reports that the check can symlink the fallback to the expected
// name when --fix is requested.
func (c *InstructionsFileCheck) CanFix() bool { return true }

// Fix symlinks the existing fallback to the expected filename for each
// recorded gap. Symlink failures are aggregated and returned as a single
// error; partial success is preserved (we do not roll back files that
// linked successfully).
func (c *InstructionsFileCheck) Fix(_ *CheckContext) error {
	if c.cfg == nil {
		return nil
	}
	gaps := c.collectGaps()
	var errs []error
	for _, g := range gaps {
		target := filepath.Join(g.rigPath, g.expected)
		// Re-check existence in case another check already fixed it.
		if _, err := os.Lstat(target); err == nil {
			continue
		}
		if err := os.Symlink(g.fallback, target); err != nil {
			errs = append(errs, fmt.Errorf("rig %q: symlink %s -> %s: %w", g.rigName, g.fallback, target, err))
		}
	}
	return errors.Join(errs...)
}

// instructionsFileGap is one missing-but-recoverable expected file in a
// rig.
type instructionsFileGap struct {
	rigName  string
	rigPath  string
	provider string
	expected string // canonical filename the provider wants
	fallback string // name of a sibling file actually present in rigPath
}

// collectGaps walks every rig × provider-in-use combination and records
// the gaps. Output is sorted (rig, expected, provider) so warnings are
// stable across runs.
func (c *InstructionsFileCheck) collectGaps() []instructionsFileGap {
	specs := providersInUse(c.cfg)
	if len(specs) == 0 {
		return nil
	}
	// Map each provider in use to the InstructionsFile name it expects,
	// applying the same "default to AGENTS.md when empty" rule that
	// ResolveProvider uses at runtime.
	expectedByProvider := make(map[string]string, len(specs))
	for name, spec := range specs {
		want := strings.TrimSpace(spec.InstructionsFile)
		if want == "" {
			want = "AGENTS.md"
		}
		expectedByProvider[name] = want
	}

	type rigEntry struct {
		name string
		path string
	}
	var rigs []rigEntry
	for _, r := range c.cfg.Rigs {
		p := strings.TrimSpace(r.Path)
		if p == "" {
			continue
		}
		if !filepath.IsAbs(p) {
			p = filepath.Join(c.cityPath, p)
		}
		rigs = append(rigs, rigEntry{name: r.Name, path: filepath.Clean(p)})
	}
	if len(rigs) == 0 {
		return nil
	}

	var gaps []instructionsFileGap
	seen := map[string]struct{}{}
	for _, rig := range rigs {
		for prov, want := range expectedByProvider {
			if instructionsFileExists(rig.path, want) {
				continue
			}
			fb := firstFallback(rig.path, want)
			if fb == "" {
				continue
			}
			key := rig.name + "|" + want
			if _, dup := seen[key]; dup {
				continue
			}
			seen[key] = struct{}{}
			gaps = append(gaps, instructionsFileGap{
				rigName:  rig.name,
				rigPath:  rig.path,
				provider: prov,
				expected: want,
				fallback: fb,
			})
		}
	}
	sort.Slice(gaps, func(i, j int) bool {
		if gaps[i].rigName != gaps[j].rigName {
			return gaps[i].rigName < gaps[j].rigName
		}
		return gaps[i].expected < gaps[j].expected
	})
	return gaps
}

// instructionsFileExists reports whether name (a filename, not a path)
// exists as a regular file or symlink inside dir. Empty files count as
// present — users sometimes ship empty placeholders intentionally.
func instructionsFileExists(dir, name string) bool {
	info, err := fsys.OSFS{}.Stat(filepath.Join(dir, name))
	if err != nil {
		return false
	}
	return !info.IsDir()
}

// firstFallback returns the name of the first known instruction file
// present in dir that is NOT want, in priority order. Empty string when
// no fallback is present.
func firstFallback(dir, want string) string {
	for _, name := range instructionsFileFallbacks {
		if name == want {
			continue
		}
		if instructionsFileExists(dir, name) {
			return name
		}
	}
	return ""
}
