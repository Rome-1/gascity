package config

import (
	"strings"
	"testing"

	"github.com/gastownhall/gascity/internal/fsys"
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

func TestLoadWithIncludesFormulasDirWarningsUseDefiningSource(t *testing.T) {
	for _, tc := range []struct {
		name   string
		files  map[string][]byte
		source string
	}{
		{
			name: "city",
			files: map[string][]byte{
				"/city/city.toml": []byte(`
[workspace]
name = "test"

[formulas]
dir = "formulas"
`),
			},
			source: "/city/city.toml",
		},
		{
			name: "pack",
			files: map[string][]byte{
				"/city/city.toml": []byte(`
[workspace]
name = "test"
`),
				"/city/pack.toml": []byte(`
[pack]
name = "test"
schema = 2

[formulas]
dir = "formulas"
`),
			},
			source: "/city/pack.toml",
		},
		{
			name: "fragment",
			files: map[string][]byte{
				"/city/city.toml": []byte(`
include = ["fragment.toml"]

[workspace]
name = "test"
`),
				"/city/fragment.toml": []byte(`
[formulas]
dir = "formulas"
`),
			},
			source: "/city/fragment.toml",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fs := fsys.NewFake()
			for path, data := range tc.files {
				fs.Files[path] = data
			}

			_, prov, err := LoadWithIncludes(fs, "/city/city.toml")
			if err != nil {
				t.Fatalf("LoadWithIncludes: %v", err)
			}

			warning := findFormulasDirWarning(prov.Warnings)
			if warning == "" {
				t.Fatalf("expected formulas dir warning, got: %v", prov.Warnings)
			}
			if !strings.Contains(warning, tc.source) {
				t.Fatalf("warning source = %q, want %q in %q", warning, tc.source, warning)
			}
			if !strings.Contains(warning, "deprecated") {
				t.Fatalf("warning should mention deprecation, got: %s", warning)
			}
		})
	}
}

func TestLoadWithIncludesFormulasDirErrorsUseDefiningSource(t *testing.T) {
	for _, tc := range []struct {
		name   string
		files  map[string][]byte
		source string
	}{
		{
			name: "city",
			files: map[string][]byte{
				"/city/city.toml": []byte(`
[workspace]
name = "test"

[formulas]
dir = ".gc/formulas"
`),
			},
			source: "/city/city.toml",
		},
		{
			name: "pack",
			files: map[string][]byte{
				"/city/city.toml": []byte(`
[workspace]
name = "test"
`),
				"/city/pack.toml": []byte(`
[pack]
name = "test"
schema = 2

[formulas]
dir = ".gc/formulas"
`),
			},
			source: "/city/pack.toml",
		},
		{
			name: "fragment",
			files: map[string][]byte{
				"/city/city.toml": []byte(`
include = ["fragment.toml"]

[workspace]
name = "test"
`),
				"/city/fragment.toml": []byte(`
[formulas]
dir = ".gc/formulas"
`),
			},
			source: "/city/fragment.toml",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fs := fsys.NewFake()
			for path, data := range tc.files {
				fs.Files[path] = data
			}

			_, _, err := LoadWithIncludes(fs, "/city/city.toml")
			if err == nil {
				t.Fatal("expected LoadWithIncludes error, got nil")
			}
			msg := err.Error()
			if !strings.Contains(msg, tc.source) {
				t.Fatalf("error source = %q, want %q in %q", msg, tc.source, msg)
			}
			if !strings.Contains(msg, ".gc/formulas") {
				t.Fatalf("error should include bad value, got: %s", msg)
			}
		})
	}
}

func findFormulasDirWarning(warnings []string) string {
	for _, warning := range warnings {
		if strings.Contains(warning, "[formulas].dir") {
			return warning
		}
	}
	return ""
}
