package config

import "fmt"

// DetectLegacyWorkspaceFields emits one soft-deprecation warning per
// populated v1 [workspace] sub-field that has a v.next replacement.
// Per docs/packv2/skew-analysis.md, these surfaces will be removed
// from [workspace] in a future release. Each warning starts with
// "<source>: workspace.<field> is deprecated:" and includes the
// suggested replacement.
//
// This function runs alongside ValidateSemantics during config load
// and contributes to Provenance.Warnings. Output ordering matches the
// declaration order below so warning text is stable across runs.
//
// Detection rules per field:
//   - workspace.provider: warn when non-empty.
//   - workspace.start_command: warn when non-empty.
//   - workspace.suspended: warn when true (the false zero-value is
//     indistinguishable from unset).
//   - workspace.install_agent_hooks: warn when non-empty.
//   - workspace.global_fragments: warn when non-empty.
func DetectLegacyWorkspaceFields(cfg *City, source string) []string {
	if cfg == nil {
		return nil
	}
	ws := cfg.Workspace

	var warnings []string
	emit := func(field, suggestion string) {
		warnings = append(warnings, fmt.Sprintf(
			"%s: workspace.%s is deprecated: %s",
			source, field, suggestion,
		))
	}

	if ws.Provider != "" {
		emit("provider", "Use `[agent_defaults] provider = ...` instead.")
	}
	if ws.StartCommand != "" {
		emit("start_command", "Use per-agent `start_command` in `agent.toml` instead.")
	}
	if ws.Suspended {
		emit("suspended", "Use `gc suspend`/`gc resume` instead.")
	}
	if len(ws.InstallAgentHooks) > 0 {
		emit("install_agent_hooks", "Use `[agent_defaults]` instead.")
	}
	if len(ws.GlobalFragments) > 0 {
		emit("global_fragments", "Use `[agent_defaults] append_fragments` or explicit `{{ template }}` instead.")
	}
	return warnings
}
