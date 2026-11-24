package testutil

// Port allocation for the socket-owning test packages. The window and its
// four package bands sit strictly below the kernel's ephemeral range, so a
// picked port is unreachable to every :0 bind; the picker holds a bind probe
// while it claims the candidate, so no sibling picker in this process can
// take it between pick and bind; and the bands are disjoint per package, so
// separate test processes cannot collide either. The old throwaway-:0 picker
// this replaces left a window between the bind that read the port and the
// bind that advertised it — the pick-to-bind race class (08-RESEARCH F-4).

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math/rand/v2"
	"net"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"testing"
)

// The private port window, partitioned per socket-owning test package.
// Every band is half-open and lies above the [27900, 28100) doc-port
// reserve: the fixed README.md and docs/EXAMPLES.md ports must stay
// reachable and disjoint from every test pick. The window ends at 32100,
// below the kernel's default ephemeral range (ip_local_port_range starts at
// 32768 on Linux), so a foreign :0 bind can never land inside a band.
const (
	bandStandLo   = 28100
	bandStandHi   = 29100
	bandCmdLo     = 29100
	bandCmdHi     = 30100
	bandUDPReplLo = 30100
	bandUDPReplHi = 31100
	bandGRPCApiLo = 31100
	bandGRPCApiHi = 32100

	// portCandidates bounds a pick's retries before it reports the band
	// exhausted or off-suite-occupied.
	portCandidates = 32

	// ephemeralRangePath is the Linux sysctl holding the kernel's ephemeral
	// port range; the allocator's structural guarantee depends on its low
	// bound sitting above the private window.
	ephemeralRangePath = "/proc/sys/net/ipv4/ip_local_port_range"

	// ephemeralRangeFields is how many bounds that sysctl prints.
	ephemeralRangeFields = 2
)

// The allocator's failures: a caller sees a named band and attempt count, and
// never a zero port.
var (
	errBandEmpty      = errors.New("testutil: the requested port band is empty")
	errNoFreePort     = errors.New("testutil: no free port in the requested band")
	errEphemeralRange = errors.New("testutil: the private port window reaches the kernel's ephemeral range")
)

// portRegistry is the shared claim set: one mutex-guarded map keyed by
// network:port, so a TCP and a UDP claim of the same number never collide
// (the kernel namespaces protocols separately) while two claims of the same
// network and port do. Entries live until the owning test's cleanup, so a
// pick stays reserved across the test's whole lifetime.
type portRegistry struct {
	mu    sync.Mutex
	taken map[string]bool
}

// portClaims is the process-wide claim set every package's picker shares.
var portClaims = portRegistry{taken: make(map[string]bool)} //nolint:gochecknoglobals // the shared claim map IS the registry contract: every picker in this process must see one set of claims.

// Band is one test package's half-open slice of the private port window.
// The bounds are unexported: the four constructor functions are the only
// producers, so a package cannot invent a band that overlaps a sibling's.
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

// UDPPort returns one free UDP port from band, reserved for tb until its
// cleanup, and fails the test naming the band when no candidate is free.
func UDPPort(tb testing.TB, band Band) int {
	tb.Helper()
	return pick(tb, band, "udp4")
}

// TCPPort returns one free TCP port from band, reserved for tb until its
// cleanup, and fails the test naming the band when no candidate is free. TCP
// and UDP picks may share a number: the registry key includes the network.
func TCPPort(tb testing.TB, band Band) int {
	tb.Helper()
	return pick(tb, band, "tcp")
}

// pick claims one free port for tb and registers the release as its cleanup,
// so the claim outlives the call and every later pick in the same test
// skips it. claim reports through the returned error, so a pick failure names
// the band and the attempt count and never escapes as a zero port.
func pick(tb testing.TB, band Band, network string) int {
	tb.Helper()
	port, err := claim(band, network)
	if err != nil {
		tb.Fatalf("%v", err)
	}
	tb.Cleanup(func() { release(network, port) })
	return port
}

// claim reserves one free port from band for network: up to portCandidates
// random candidates, each bind-probed while the claim is still held, so a
// returned port cannot be taken by a sibling picker between pick and bind.
// The probe bind proves the port is free at the kernel; holding the claim
// while it is open is what closes the window. A candidate held by an
// off-suite process fails its probe and is skipped.
func claim(band Band, network string) (int, error) {
	if band.hi <= band.lo {
		return 0, fmt.Errorf("%w: band %s has no candidate", errBandEmpty, band)
	}
	err := checkEphemeral(band)
	if err != nil {
		return 0, err
	}
	// testing.TB carries no Context method, so the probe uses the background
	// context; the bind is synchronous and bounded by the kernel call.
	ctx := context.Background()
	for range portCandidates {
		candidate := band.lo + rand.IntN(band.hi-band.lo) //nolint:gosec // G404: the port draw is a convenience pick, never a security function.
		probe, err := probeBind(ctx, network, candidate)
		if err != nil {
			continue
		}
		taken := reserve(network, candidate)
		_ = probe.Close()
		if taken {
			continue
		}
		return candidate, nil
	}
	return 0, fmt.Errorf(
		"%w: band %s has no free %s port after %d candidates (every candidate is claimed or an off-suite process holds it)",
		errNoFreePort,
		band,
		network,
		portCandidates,
	)
}

// probeBind binds a throwaway socket for the candidate on the wildcard
// address and returns it open, so the caller can claim the candidate while
// the bind is still held. The stream and datagram families take different
// constructors: a TCP listener and a UDP packet connection are both proof
// that the port is free.
func probeBind(ctx context.Context, network string, port int) (io.Closer, error) {
	addr := "0.0.0.0:" + strconv.Itoa(port)
	if network == "tcp" {
		listener, err := (&net.ListenConfig{}).Listen(ctx, network, addr)
		if err != nil {
			return nil, err
		}
		return listener, nil
	}
	conn, err := (&net.ListenConfig{}).ListenPacket(ctx, network, addr)
	if err != nil {
		return nil, err
	}
	return conn, nil
}

// reserve records the network:port claim and reports whether a sibling picker
// already held it. It must be called while the candidate's probe bind is
// still held.
func reserve(network string, port int) bool {
	key := network + ":" + strconv.Itoa(port)
	portClaims.mu.Lock()
	defer portClaims.mu.Unlock()
	if portClaims.taken[key] {
		return true
	}
	portClaims.taken[key] = true
	return false
}

// release drops the claim a test held until its cleanup.
func release(network string, port int) {
	key := network + ":" + strconv.Itoa(port)
	portClaims.mu.Lock()
	defer portClaims.mu.Unlock()
	delete(portClaims.taken, key)
}

// checkEphemeral asserts the structural assumption behind the bands: the
// kernel's ephemeral range starts above the private window. Linux reads the
// sysctl; every other platform skips the check with a comment because darwin
// is development-only (14.10) and has no /proc sysctl to read.
func checkEphemeral(band Band) error {
	if runtime.GOOS != "linux" {
		// darwin is development-only (14.10): there is no
		// /proc/sys/net/ipv4/ip_local_port_range to assert against, so the
		// guarantee is documented and unasserted rather than guessed.
		return nil
	}
	raw, err := os.ReadFile(ephemeralRangePath)
	if err != nil {
		return fmt.Errorf("read %s: %w", ephemeralRangePath, err)
	}
	fields := strings.Fields(string(raw))
	if len(fields) != ephemeralRangeFields {
		return fmt.Errorf("%w: %s has %d fields, want %d", errEphemeralRange, ephemeralRangePath, len(fields), ephemeralRangeFields)
	}
	low, err := strconv.Atoi(fields[0])
	if err != nil {
		return fmt.Errorf("%s low bound %q: %w", ephemeralRangePath, fields[0], err)
	}
	return ephemeralOK(low, band)
}

// ephemeralOK reports whether band stays strictly below the ephemeral range
// that starts at low. A low bound at or inside the band's upper edge means
// the kernel can hand out a port the bands assume are private, so the
// caller fails loudly instead of flaking. low is a parameter so the
// assertion is testable without touching the host.
func ephemeralOK(low int, band Band) error {
	if low <= band.hi {
		return fmt.Errorf(
			"%w: %s reports low bound %d, at or inside band %s",
			errEphemeralRange,
			ephemeralRangePath,
			low,
			band,
		)
	}
	return nil
}
