package testutil

// Port allocation for the socket-owning test packages. The private window
// and its four package bands are below the kernel's ephemeral range, so a
// picked port is unreachable to every :0 bind; the RED stage of this file's
// TDD cycle carries the old throwaway-:0 picker verbatim (the class being
// removed), so the behavior tests fail on the picks they receive instead of
// on a missing symbol.

import (
	"context"
	"fmt"
	"net"
	"testing"
)

// The private port window, partitioned per socket-owning test package.
// Every band is half-open and lies above the [27900, 28100) doc-port
// reserve: the fixed README.md and docs/EXAMPLES.md ports must stay
// reachable and disjoint from every test pick.
const (
	bandStandLo   = 28100
	bandStandHi   = 29100
	bandCmdLo     = 29100
	bandCmdHi     = 30100
	bandUDPReplLo = 30100
	bandUDPReplHi = 31100
	bandGRPCApiLo = 31100
	bandGRPCApiHi = 32100

	// portCandidates bounds the picker's retries before it reports the band
	// exhausted or off-suite-occupied.
	portCandidates = 32
)

// Band is one test package's half-open slice of the private port window.
// The bounds are unexported: the four constructor functions are the only
// producers, so a package can never invent a band that overlaps a sibling's.
type Band struct {
	name string
	lo   int
	hi   int
}

// StandBand returns internal/stand's band inside the private window.
func StandBand() Band { return Band{name: "internal/stand", lo: bandStandLo, hi: bandStandHi} }

// CmdBand returns cmd/sup-kv's band inside the private window.
func CmdBand() Band { return Band{name: "cmd/sup-kv", lo: bandCmdLo, hi: bandCmdHi} }

// UDPReplBand returns internal/udprepl's band inside the private window.
func UDPReplBand() Band { return Band{name: "internal/udprepl", lo: bandUDPReplLo, hi: bandUDPReplHi} }

// GRPCApiBand returns internal/grpcapi's band inside the private window.
func GRPCApiBand() Band { return Band{name: "internal/grpcapi", lo: bandGRPCApiLo, hi: bandGRPCApiHi} }

// String names the band and its range for failure messages.
func (b Band) String() string { return fmt.Sprintf("%s [%d, %d)", b.name, b.lo, b.hi) }

// UDPPort returns one free UDP port from band.
func UDPPort(tb testing.TB, band Band) int {
	tb.Helper()
	port, err := claim(band, "udp4")
	if err != nil {
		tb.Fatalf("%v", err)
	}
	return port
}

// TCPPort returns one free TCP port from band.
func TCPPort(tb testing.TB, band Band) int {
	tb.Helper()
	port, err := claim(band, "tcp")
	if err != nil {
		tb.Fatalf("%v", err)
	}
	return port
}

// claim picks one free port: the pre-fix throwaway :0 bind, which reads a
// port from the kernel's ephemeral pool and ignores the band.
func claim(_ Band, network string) (int, error) {
	conn, err := (&net.ListenConfig{}).ListenPacket(context.Background(), network, "0.0.0.0:0")
	if err != nil {
		return 0, fmt.Errorf("bind throwaway %s socket: %w", network, err)
	}
	defer func() { _ = conn.Close() }()
	switch addr := conn.LocalAddr().(type) {
	case *net.UDPAddr:
		return addr.Port, nil
	case *net.TCPAddr:
		return addr.Port, nil
	default:
		return 0, fmt.Errorf("unexpected %s local address %T", network, conn.LocalAddr())
	}
}

// release drops a claim; the pre-fix picker holds no claims.
func release(_ string, _ int) {}

// ephemeralOK reports whether band stays below the kernel's ephemeral range;
// the pre-fix picker never asserted the assumption.
func ephemeralOK(_ int, _ Band) error { return nil }

// checkEphemeral asserts the host's ephemeral range against band; the
// pre-fix picker never read the sysctl.
func checkEphemeral(_ Band) error { return nil }
