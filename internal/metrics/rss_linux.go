//go:build linux

package metrics

import (
	"os"
	"strconv"
	"strings"
)

// statmResidentField is the zero-based index of /proc/self/statm's resident
// field: the count of pages currently mapped into the process's resident
// set. Field 0 is the total program size and field 1 the resident count, so
// reading field 1 (never field 0) is what makes this a current reading.
const statmResidentField = 1

// statmPath is the kernel file the Linux RSS probe reads: the same
// self-referential view every process has of its own mappings.
const statmPath = "/proc/self/statm"

// currentRSS returns this process's CURRENT resident set size in bytes on
// Linux, read from /proc/self/statm's resident page count times the page size
// (OBSV-03, D-05). A lifetime peak — what [syscall.Getrusage]'s Maxrss
// reports — can only grow, so a gauge over it cannot describe present memory
// pressure. Any read or parse failure returns 0: a scrape must not fail
// because a kernel file is unavailable.
func currentRSS() float64 {
	raw, err := os.ReadFile(statmPath)
	if err != nil {
		return 0
	}
	rss, ok := parseStatm(raw)
	if !ok {
		return 0
	}
	return rss
}

// parseStatm parses one /proc/self/statm reading and returns the resident size
// in bytes, reporting ok=false for a truncated or non-numeric reading — the
// two shapes the parser table pins.
func parseStatm(raw []byte) (float64, bool) {
	fields := strings.Fields(string(raw))
	if len(fields) <= statmResidentField {
		return 0, false
	}
	pages, err := strconv.ParseUint(fields[statmResidentField], 10, 64)
	if err != nil {
		return 0, false
	}
	return float64(pages) * float64(os.Getpagesize()), true
}
