//go:build race

package testutil

// RaceEnabled reports whether this test binary was built with the race
// detector: the race_on.go side of the build-tag pair, compiled under -race.
// It answers a constant, so no test body probes the runtime to decide it.
func RaceEnabled() bool { return true }
