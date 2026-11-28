//go:build linux

package stand //nolint:testpackage // emptyGateArg is unexported harness logic; the scenario files share the package by design (D-03).

// the empty-gate derivation's form table (IN-05): the harness must recognize
// every flag spelling Go's flag package accepts, so a scenario's explicit
// `--startup-gate 30s` (or any single-dash variant) derives its empty twin
// instead of silently leaving the node's empty gate at the 1 s harness
// default — the race RESEARCH Pitfall 1 exists to prevent.

import "testing"

// TestEmptyGateArg_forms pins the derivation over every accepted spelling,
// the explicit-empty-gate override in both forms, and the later-wins rule.
func TestEmptyGateArg_forms(t *testing.T) {
	t.Parallel()
	derived := standStartupGate.String()
	rows := []struct {
		name  string
		extra []string
		want  string
	}{
		{"no extras derives the harness default", nil, "--startup-gate-empty=" + derived},
		{"double-dash equals form", []string{"--startup-gate=30s"}, "--startup-gate-empty=30s"},
		{"double-dash pair form", []string{"--startup-gate", "30s"}, "--startup-gate-empty=30s"},
		{"single-dash equals form", []string{"-startup-gate=45s"}, "--startup-gate-empty=45s"},
		{"single-dash pair form", []string{"-startup-gate", "45s"}, "--startup-gate-empty=45s"},
		{"a bare empty gate wins", []string{"--startup-gate-empty"}, ""},
		{"the double-dash empty-gate equals form wins", []string{"--startup-gate-empty=2s"}, ""},
		{"the single-dash empty-gate form wins", []string{"-startup-gate-empty=2s"}, ""},
		{"the last normal gate wins", []string{"--startup-gate=10s", "--startup-gate", "20s"}, "--startup-gate-empty=20s"},
		{"an unrelated flag is not a gate", []string{"--startup-gate=30s", "--sync-interval=10ms"}, "--startup-gate-empty=30s"},
		{"a value that looks like a gate is not one", []string{"--startup-gate=30s", "--log-format=startup-gate"}, "--startup-gate-empty=30s"},
	}
	for _, row := range rows {
		t.Run(row.name, func(t *testing.T) {
			t.Parallel()
			if got := emptyGateArg(row.extra); got != row.want {
				t.Fatalf("emptyGateArg(%v) = %q, want %q — every flag spelling must derive (or defer) the same way (IN-05)",
					row.extra, got, row.want)
			}
		})
	}
}
