package config

import (
	"strings"
	"testing"
)

func TestValidateFormulasDirEmpty(t *testing.T) {
	cfg := &City{}
	warnings, err := ValidateFormulasDir(cfg, "city.toml")
	if err != nil {
		t.Fatalf("expected no error for empty dir, got: %v", err)
	}
	if len(warnings) != 0 {
		t.Fatalf("expected no warnings for empty dir, got: %v", warnings)
	}
}

func TestValidateFormulasDirNilCity(t *testing.T) {
	warnings, err := ValidateFormulasDir(nil, "city.toml")
	if err != nil {
		t.Fatalf("expected no error for nil city, got: %v", err)
	}
	if len(warnings) != 0 {
		t.Fatalf("expected no warnings for nil city, got: %v", warnings)
	}
}

func TestValidateFormulasDirCanonicalSoftWarns(t *testing.T) {
	cfg := &City{Formulas: FormulasConfig{Dir: "formulas"}}
	warnings, err := ValidateFormulasDir(cfg, "city.toml")
	if err != nil {
		t.Fatalf("expected no error for canonical dir, got: %v", err)
	}
	if len(warnings) != 1 {
		t.Fatalf("expected exactly 1 warning, got %d: %v", len(warnings), warnings)
	}
	if !strings.Contains(warnings[0], "city.toml") {
		t.Errorf("warning should include source %q, got: %s", "city.toml", warnings[0])
	}
	if !strings.Contains(warnings[0], "deprecated") {
		t.Errorf("warning should mention deprecation, got: %s", warnings[0])
	}
	if !strings.Contains(warnings[0], "[formulas]") {
		t.Errorf("warning should reference [formulas], got: %s", warnings[0])
	}
}

func TestValidateFormulasDirNonCanonicalHardErrors(t *testing.T) {
	for _, value := range []string{".gc/formulas", "my-formulas", "../formulas", "/abs/formulas"} {
		t.Run(value, func(t *testing.T) {
			cfg := &City{Formulas: FormulasConfig{Dir: value}}
			warnings, err := ValidateFormulasDir(cfg, "city.toml")
			if err == nil {
				t.Fatalf("expected error for non-canonical dir %q, got nil", value)
			}
			if len(warnings) != 0 {
				t.Errorf("expected no warnings on hard error, got: %v", warnings)
			}
			msg := err.Error()
			if !strings.Contains(msg, value) {
				t.Errorf("error should include the bad value %q, got: %s", value, msg)
			}
			if !strings.Contains(msg, "city.toml") {
				t.Errorf("error should include source, got: %s", msg)
			}
			if !strings.Contains(msg, "fixed convention") {
				t.Errorf("error should mention fixed convention, got: %s", msg)
			}
		})
	}
}
