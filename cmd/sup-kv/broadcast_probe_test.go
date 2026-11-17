package main_test

// The quickstart-broadcast probe of Phase 7 (DOCS-01's macOS leg —
// 07-RESEARCH Open Question 1): it answers ONE binary question — does the
// exact broadcast-address derive line the README quickstart will use
// deliver a datagram to a local socket on THIS platform? Phase 6 proved
// macOS lo0 carries no BROADCAST flag, so loopback broadcast (the Linux
// quickstart form) does not fan out on darwin; whether the runner's subnet
// broadcast (the `ipconfig getoption en0 broadcast` arm of the derive line)
// or limited broadcast (255.255.255.255) fan out locally was never probed.
// The derived row is the verdict: not reached is a failure carrying the
// full probe table (on linux a loopback regression, on darwin the blocking
// DOCS-01 decision made visible); reached is a pass that logs the working
// address. The loopback and limited-broadcast rows are informational table
// rows on every platform — they tell the Option A/B/C decision what else
// works. This test intentionally does NOT skip on darwin: a green macos-15
// cell after this plan is the Option-A evidence, a red one with the table
// is the Option-B/C evidence (the 507228a loud-skip precedent follows the
// human decision, not this probe).

import (
	"fmt"
	"net"
	"os/exec"
	"runtime"
	"strings"
	"testing"
	"time"
)

// probeAttempts bounds the fresh-port retries when a probe cannot even
// bind: an off-suite holder in the private window is a pick-to-bind race,
// never a capability verdict (harness soundness is assertable, capability
// is not).
const probeAttempts = 5

// probeSendBroadcast probes whether ONE datagram sent to dstIP:dstPort
// reaches a socket bound to 0.0.0.0:dstPort — probeLoopbackSend generalized
// over the destination address. The two failure classes stay distinct so
// the caller can act: not reached within the bounded read deadline (the
// platform lacks the fan-out — a capability result), versus a sendErr (a
// bind or send failure — a probe error, retried on a fresh port).
func probeSendBroadcast(t *testing.T, dstIP string, dstPort int) (reached bool, sendErr string) {
	t.Helper()
	rx, err := logProbeSocket(fmt.Sprintf("0.0.0.0:%d", dstPort))
	if err != nil {
		return false, "rx bind: " + err.Error()
	}
	defer func() { _ = rx.Close() }()
	tx, err := logProbeSocket("0.0.0.0:0")
	if err != nil {
		return false, "tx bind: " + err.Error()
	}
	defer func() { _ = tx.Close() }()
	dst := &net.UDPAddr{IP: net.ParseIP(dstIP), Port: dstPort}
	_, err = tx.WriteToUDP([]byte("sup-kv quickstart broadcast probe"), dst)
	if err != nil {
		return false, "send to " + dst.String() + ": " + err.Error()
	}
	err = rx.SetReadDeadline(time.Now().Add(logProbeWait))
	if err != nil {
		return false, "read deadline: " + err.Error()
	}
	buf := make([]byte, 64)
	for {
		n, _, readErr := rx.ReadFromUDP(buf)
		if readErr != nil {
			return false, "" // deadline hit: not reached, and the probe itself was sound.
		}
		if n > 0 {
			return true, ""
		}
	}
}

// probeBroadcastForm probes one broadcast address form on a fresh
// private-window port, retrying bind/send errors up to probeAttempts
// times: a fresh port each round is what makes an off-suite holder in the
// window transient. Exhausting the attempts is a harness failure and fails
// the test — never silently reported as a capability result.
func probeBroadcastForm(t *testing.T, dstIP string) (reached bool, sendErr string) {
	t.Helper()
	for attempt := 1; ; attempt++ {
		reached, sendErr = probeSendBroadcast(t, dstIP, scenarioUDPPort(t))
		if sendErr == "" {
			return reached, ""
		}
		if attempt >= probeAttempts {
			t.Fatalf("probe of %s could not get a sound setup in %d attempts: %s", dstIP, probeAttempts, sendErr)
		}
	}
}

// deriveQuickstartBroadcast derives the broadcast address exactly as the
// README quickstart's derive line will: on darwin the primary arm reads
// `ipconfig getoption en0 broadcast`, on every other platform the || echo
// fallback arm is the literal loopback broadcast. A derive failure (error
// or empty output) returns an empty address with the reason — on darwin
// that is itself probe evidence (the derive line's primary arm is broken
// there), surfaced by the caller's table, not hidden by a default.
func deriveQuickstartBroadcast(t *testing.T) (addr, note string) {
	t.Helper()
	if runtime.GOOS != "darwin" {
		return "127.255.255.255", "the || echo fallback arm (non-darwin)"
	}
	out, err := exec.CommandContext(t.Context(), "ipconfig", "getoption", "en0", "broadcast").Output()
	if err != nil {
		return "", fmt.Sprintf("ipconfig getoption en0 broadcast failed: %v", err)
	}
	addr = strings.TrimSpace(string(out))
	if addr == "" {
		return "", "ipconfig getoption en0 broadcast printed an empty address"
	}
	return addr, "the ipconfig getoption en0 broadcast arm (darwin)"
}

// probeRow is one line of the probe table the test prints.
type probeRow struct {
	form    string
	addr    string
	reached bool
	sendErr string
}

// String renders the row for both the passing log and the fatal table.
func (r probeRow) String() string {
	verdict := "not reached"
	if r.reached {
		verdict = "reached"
	}
	if r.sendErr != "" {
		verdict = "probe error: " + r.sendErr
	}
	return fmt.Sprintf("| %s | %s | %s |", r.form, r.addr, verdict)
}

// TestProbeQuickstartBroadcast runs the derive line of the future README
// quickstart and probes whether its address delivers to a local socket on
// this platform, alongside the two informational forms (loopback and
// limited broadcast). The DERIVED row is the verdict; the table always
// names GOOS/GOARCH, the derived address and every form's result.
func TestProbeQuickstartBroadcast(t *testing.T) {
	t.Parallel()
	derived, deriveNote := deriveQuickstartBroadcast(t)

	rows := make([]probeRow, 0, 3)
	if derived == "" {
		rows = append(rows, probeRow{form: "derived (quickstart line)", addr: "DERIVE FAILED: " + deriveNote})
	} else {
		reached, sendErr := probeBroadcastForm(t, derived)
		rows = append(rows, probeRow{form: "derived (quickstart line)", addr: derived + " (" + deriveNote + ")", reached: reached, sendErr: sendErr})
	}
	loopReached, loopErr := probeBroadcastForm(t, "127.255.255.255")
	rows = append(rows, probeRow{form: "loopback broadcast", addr: "127.255.255.255", reached: loopReached, sendErr: loopErr})
	limitedReached, limitedErr := probeBroadcastForm(t, "255.255.255.255")
	rows = append(rows, probeRow{form: "limited broadcast", addr: "255.255.255.255", reached: limitedReached, sendErr: limitedErr})

	table := fmt.Sprintf("quickstart broadcast probe on %s/%s — derived BCAST %q (%s)\n",
		runtime.GOOS, runtime.GOARCH, derived, deriveNote)
	for _, row := range rows {
		table += row.String() + "\n"
	}
	t.Log(table)

	if rows[0].reached {
		t.Logf("the quickstart derive line's broadcast address %q delivers to a local socket on this platform", derived)
		return
	}
	t.Fatalf("the quickstart derive line's broadcast address does NOT deliver to a local socket on %s/%s — "+
		"on linux this is a loopback-broadcast regression (load-bearing for every scenario), on darwin the blocking DOCS-01 decision.\n%s",
		runtime.GOOS, runtime.GOARCH, table)
}
