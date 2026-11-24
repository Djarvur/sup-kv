package testutil //nolint:testpackage // the band table, the registry key and the ephemeral-range assertion are unexported implementation detail; an external test package cannot reach them.

import (
	"runtime"
	"strconv"
	"strings"
	"testing"
)

const (
	// testBandLo starts the tests' spare slice of the private window: above
	// the four package bands and still below the kernel's default ephemeral
	// range, so a deterministic test pick cannot collide with a package band.
	testBandLo = 32100

	// defaultLinuxLow is the kernel's default ip_local_port_range low bound:
	// the assumption the allocator's ephemeral-range assertion encodes.
	defaultLinuxLow = 32768
)

// TestBandsPartitionThePrivateWindow pins the partition that makes
// cross-process collisions impossible by construction: four pairwise
// disjoint half-open bands inside [28100, 32100) and above the [27900, 28100)
// doc-port reserve.
func TestBandsPartitionThePrivateWindow(t *testing.T) {
	t.Parallel()
	const (
		reserveHi = 28100 // the first port above the [27900, 28100) doc-port reserve
		windowHi  = 32100 // the first port above the private window
	)
	bands := []Band{StandBand(), CmdBand(), UDPReplBand(), GRPCApiBand()}
	for _, band := range bands {
		if band.lo >= band.hi {
			t.Fatalf("band %s is empty: lo must be below hi", band)
		}
		if band.lo < reserveHi || band.hi > windowHi {
			t.Fatalf("band %s leaves the private window [%d, %d): the doc-port reserve and the ephemeral range must stay untouched", band, reserveHi, windowHi)
		}
	}
	for i := range bands {
		for j := range bands {
			if i == j {
				continue
			}
			if bands[i].lo < bands[j].hi && bands[j].lo < bands[i].hi {
				t.Fatalf("bands %s and %s overlap: separate test processes must never pick the same number band", bands[i], bands[j])
			}
		}
	}
}

// TestUDPPortPickIsInBandAndDistinct pins the claim-while-held contract: 64
// consecutive picks on one test yield 64 distinct in-band ports, because the
// registry holds each pick until the test's cleanup.
func TestUDPPortPickIsInBandAndDistinct(t *testing.T) {
	t.Parallel()
	band := StandBand()
	const picks = 64
	seen := make(map[int]bool, picks)
	for range picks {
		port := UDPPort(t, band)
		if port < band.lo || port >= band.hi {
			t.Fatalf("UDPPort(t, %s) = %d, outside the band", band, port)
		}
		if seen[port] {
			t.Fatalf("UDPPort(t, %s) returned %d twice: the registry must hold the first pick until cleanup", band, port)
		}
		seen[port] = true
	}
}

// TestRegistryKeysByNetwork pins the registry key: TCP and UDP may share a
// port number (the kernel namespaces them separately), while a second claim
// of the same network:port must fail.
func TestRegistryKeysByNetwork(t *testing.T) {
	t.Parallel()
	band := Band{name: "registry", lo: testBandLo, hi: testBandLo + 1}

	udpPort, err := claim(band, "udp4")
	if err != nil {
		t.Fatalf("claim(udp4, %s): %v", band, err)
	}
	if udpPort != testBandLo {
		t.Fatalf("claim(udp4, %s) = %d, want the band's only candidate %d", band, udpPort, testBandLo)
	}
	t.Cleanup(func() { release("udp4", udpPort) })

	tcpPort, err := claim(band, "tcp")
	if err != nil {
		t.Fatalf("claim(tcp, %s) after a udp4 claim of the same number: %v — the registry key includes the network", band, err)
	}
	if tcpPort != udpPort {
		t.Fatalf("claim(tcp, %s) = %d, want %d: a TCP claim must not be blocked by a UDP claim of the same number", band, tcpPort, udpPort)
	}
	t.Cleanup(func() { release("tcp", tcpPort) })

	if _, err := claim(band, "udp4"); err == nil {
		t.Fatalf("a second udp4 claim of %d succeeded: the registry must key by network:port", udpPort)
	}
}

// TestRegistrySkipsTakenCandidate pins that a pre-claimed candidate is never
// returned: in a two-port band with one claim held, the picker must hand out
// the other number.
func TestRegistrySkipsTakenCandidate(t *testing.T) {
	t.Parallel()
	band := Band{name: "skip", lo: testBandLo + 10, hi: testBandLo + 12}

	held, err := claim(band, "udp4")
	if err != nil {
		t.Fatalf("claim(udp4, %s): %v", band, err)
	}
	t.Cleanup(func() { release("udp4", held) })

	picked := UDPPort(t, band)
	if picked < band.lo || picked >= band.hi {
		t.Fatalf("UDPPort(t, %s) = %d, outside the band", band, picked)
	}
	if picked == held {
		t.Fatalf("UDPPort(t, %s) returned %d, already claimed: the picker must skip a taken candidate", band, picked)
	}
}

// TestEphemeralRangeAssertion pins the sysctl assumption: the assertion
// fails when an injected low bound reaches into the band, passes one port
// above it, and on Linux the host's real range must sit above every band.
func TestEphemeralRangeAssertion(t *testing.T) {
	t.Parallel()
	bands := []Band{StandBand(), CmdBand(), UDPReplBand(), GRPCApiBand()}
	for _, band := range bands {
		if err := ephemeralOK(defaultLinuxLow, band); err != nil {
			t.Fatalf("ephemeralOK(%d, %s) = %v, want nil: the whole window sits below the default ephemeral range", defaultLinuxLow, band, err)
		}
		if err := ephemeralOK(band.hi, band); err == nil {
			t.Fatalf("ephemeralOK(%d, %s) = nil, want an error: a low bound at the band's upper edge reaches into %s", band.hi, band, band)
		}
		if err := ephemeralOK(band.hi+1, band); err != nil {
			t.Fatalf("ephemeralOK(%d, %s) = %v, want nil: one port above the band is still sound", band.hi+1, band, err)
		}
	}
	if runtime.GOOS != "linux" {
		t.Log("non-Linux host: checkEphemeral skips the /proc assertion with a comment (darwin is development-only, 14.10); the injected-bound assertions above stand on every platform")
		return
	}
	for _, band := range bands {
		if err := checkEphemeral(band); err != nil {
			t.Fatalf("checkEphemeral(%s) on this host: %v — the host's ip_local_port_range must sit strictly above the private window", band, err)
		}
	}
}

// TestUDPPortBandExhaustionNeverReturnsZero pins the failure contract: an
// empty band and a fully claimed band both fail naming the band and the
// candidate count, and no zero port escapes to a caller.
func TestUDPPortBandExhaustionNeverReturnsZero(t *testing.T) {
	t.Parallel()
	empty := Band{name: "empty", lo: testBandLo + 20, hi: testBandLo + 20}
	port, err := claim(empty, "udp4")
	if err == nil {
		t.Fatalf("claim(%s) = (%d, nil), want an error: an empty band has no candidate", empty, port)
	}
	if !strings.Contains(err.Error(), "empty") {
		t.Fatalf("claim(%s) error %q must name the empty band", empty, err)
	}
	if port != 0 {
		t.Fatalf("claim(%s) returned port %d with its error, want 0", empty, port)
	}

	full := Band{name: "full", lo: testBandLo + 30, hi: testBandLo + 31}
	held, err := claim(full, "udp4")
	if err != nil {
		t.Fatalf("claim(udp4, %s): %v", full, err)
	}
	t.Cleanup(func() { release("udp4", held) })

	port, err = claim(full, "udp4")
	if err == nil {
		t.Fatalf("claim(%s) = (%d, nil), want an error: every candidate is claimed", full, port)
	}
	if port != 0 {
		t.Fatalf("claim(%s) returned port %d with its error, want 0", full, port)
	}
	for _, want := range []string{full.String(), strconv.Itoa(portCandidates)} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("claim(%s) error %q must name %q: the failure names the band and the attempt count", full, err, want)
		}
	}
}
