package config

import "fmt"

// ValidateFormulasDir enforces that [formulas].dir is a fixed convention.
//
// The directory `formulas/` is now a fixed convention, matching the layout
// used by scripts/, commands/, doctor/, and agents/. The [formulas].dir
// option is being retired:
//
//   - Empty (omitted) is the canonical form: no warning, no error.
//   - "formulas" is accepted with a soft deprecation warning so existing
//     configs continue to load while users migrate.
//   - Any other value is a hard error: the convention cannot be overridden.
//
// See docs/packv2/doc-conformance-matrix.md ("Formula directory path") and
// docs/packv2/skew-analysis.md (FormulasConfig) for the migration rationale.
func ValidateFormulasDir(cfg *City, source string) ([]string, error) {
	if cfg == nil {
		return nil, nil
	}
	switch cfg.Formulas.Dir {
	case "":
		return nil, nil
	case "formulas":
		return []string{fmt.Sprintf(
			"%s: [formulas].dir is deprecated and will be removed; %q is now a fixed convention. Remove the [formulas] block.",
			source, "formulas/",
		)}, nil
	default:
		return nil, fmt.Errorf(
			"%s: [formulas].dir = %q is no longer supported; %q is a fixed convention; remove the [formulas] block",
			source, cfg.Formulas.Dir, "formulas/",
		)
	}
}
