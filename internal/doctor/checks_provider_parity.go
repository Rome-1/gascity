package doctor

import (
	"fmt"
	"sort"
	"strings"

	"github.com/gastownhall/gascity/internal/config"
)

// ProviderParityCheck flags providers used by configured agents whose
// capability fields are likely to silently no-op at runtime. The signal
// is structural — it inspects the resolved ProviderSpec for each
// provider that at least one agent in the city references, and warns
// when:
//
//   - ResumeFlag and ResumeCommand are both empty: every session restart
//     will silently drop the session-id and start a fresh process
//     (resolveResumeCommand short-circuits, gap 1 of #672).
//
// SupportsHooks=false is intentionally NOT flagged — many providers
// genuinely lack a hook surface and Gas Town has an alternative drain
// path (NeedsNudgePoller) for them. Flagging it would be noise.
//
// Each warning names the provider, what is missing, and the consequence.
// Built-in providers reuse the canonical preset; city-defined providers
// in cfg.Providers override or extend them.
type ProviderParityCheck struct {
	cfg *config.City
}

// NewProviderParityCheck creates a check that flags provider capability
// gaps for every provider referenced by a configured agent.
func NewProviderParityCheck(cfg *config.City) *ProviderParityCheck {
	return &ProviderParityCheck{cfg: cfg}
}

// Name returns the check identifier.
func (c *ProviderParityCheck) Name() string { return "provider-parity" }

// Run inspects each provider referenced by at least one agent and
// reports capability gaps as warnings.
func (c *ProviderParityCheck) Run(_ *CheckContext) *CheckResult {
	r := &CheckResult{Name: c.Name()}
	if c.cfg == nil {
		r.Status = StatusOK
		r.Message = "no config; nothing to check"
		return r
	}

	specs := providersInUse(c.cfg)
	if len(specs) == 0 {
		r.Status = StatusOK
		r.Message = "no providers referenced by configured agents"
		return r
	}

	names := make([]string, 0, len(specs))
	for name := range specs {
		names = append(names, name)
	}
	sort.Strings(names)

	var details []string
	for _, name := range names {
		spec := specs[name]
		if spec.ResumeFlag == "" && spec.ResumeCommand == "" {
			details = append(details, fmt.Sprintf(
				"provider %q has no ResumeFlag or ResumeCommand: session restarts will silently drop the session-id and start a fresh process",
				name))
		}
	}

	if len(details) == 0 {
		r.Status = StatusOK
		r.Message = "provider capabilities look complete"
		return r
	}
	r.Status = StatusWarning
	r.Message = fmt.Sprintf("%d provider capability gap(s)", len(details))
	r.Details = details
	r.FixHint = "populate resume_flag (or resume_command) in the provider spec; see internal/config/provider.go for the built-in presets and gastownhall/gascity#672 (non-Claude provider parity)"
	return r
}

// CanFix returns false — provider capability fields are config-managed.
func (c *ProviderParityCheck) CanFix() bool { return false }

// Fix is a no-op.
func (c *ProviderParityCheck) Fix(_ *CheckContext) error { return nil }

// providersInUse returns the resolved ProviderSpec for every provider
// that at least one agent in the city references (directly via
// agent.Provider or implicitly via workspace.Provider). Resolution
// follows the same order as ResolveProvider: city-defined overrides
// extend built-ins. Agents that pin StartCommand are skipped because
// they bypass ProviderSpec entirely.
func providersInUse(cfg *config.City) map[string]config.ProviderSpec {
	builtins := config.BuiltinProviders()
	out := map[string]config.ProviderSpec{}

	add := func(name string) {
		name = strings.TrimSpace(name)
		if name == "" {
			return
		}
		if _, seen := out[name]; seen {
			return
		}
		if city, ok := cfg.Providers[name]; ok {
			if builtin, hasBuiltin := builtins[name]; hasBuiltin {
				out[name] = config.MergeProviderOverBuiltin(builtin, city)
				return
			}
			out[name] = city
			return
		}
		if builtin, ok := builtins[name]; ok {
			out[name] = builtin
		}
	}

	for _, a := range cfg.Agents {
		if a.StartCommand != "" {
			continue
		}
		add(a.Provider)
	}
	add(cfg.Workspace.Provider)
	return out
}
