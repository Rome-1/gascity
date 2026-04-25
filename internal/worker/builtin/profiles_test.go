package builtin

import (
	"testing"
)

func TestBuiltinProvidersAndOrder(t *testing.T) {
	providers := BuiltinProviders()
	order := BuiltinProviderOrder()

	if len(providers) != 10 {
		t.Fatalf("len(BuiltinProviders()) = %d, want 10", len(providers))
	}
	if len(order) != 10 {
		t.Fatalf("len(BuiltinProviderOrder()) = %d, want 10", len(order))
	}

	for _, name := range order {
		spec, ok := providers[name]
		if !ok {
			t.Fatalf("BuiltinProviders() missing %q", name)
		}
		if spec.Command == "" {
			t.Fatalf("provider %q has empty Command", name)
		}
		if spec.DisplayName == "" {
			t.Fatalf("provider %q has empty DisplayName", name)
		}
	}
}

func TestCodexSessionIDFlag(t *testing.T) {
	providers := BuiltinProviders()
	codex, ok := providers["codex"]
	if !ok {
		t.Fatal("BuiltinProviders() missing codex")
	}
	if codex.SessionIDFlag != "--session-id" {
		t.Errorf("codex.SessionIDFlag = %q, want %q (Generate-and-Pass strategy needs session-id assignment)", codex.SessionIDFlag, "--session-id")
	}
}

func TestBuiltinProvidersReturnClonedData(t *testing.T) {
	a := BuiltinProviders()
	b := BuiltinProviders()

	a["claude"] = BuiltinProviderSpec{Command: "mutated"}
	if b["claude"].Command == "mutated" {
		t.Fatal("BuiltinProviders() should return a cloned map")
	}

	claude := a["codex"]
	claude.ProcessNames[0] = "mutated"
	a["codex"] = claude
	if b["codex"].ProcessNames[0] == "mutated" {
		t.Fatal("BuiltinProviders() should clone nested slices")
	}
}
