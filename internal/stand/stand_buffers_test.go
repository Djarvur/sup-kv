//go:build linux

package stand //nolint:testpackage // the scenario reuses the harness's unexported pair/node/port helpers by design (the stand_test.go contract); testpackage's external-package rule cannot express that pairing.

// The XPORT-05 real-socket delivery acceptance: two real sup-kv binaries
// carry 1 KiB, 20 KiB and 60 KiB values across loopback UDP, and both log
// their EFFECTIVE SO_RCVBUF/SO_SNDBUF sizes at startup. The delivery bound is
// derived from the flags the nodes actually run with, never a magic deadline;
// the startup assertion never depends on the kernel's exact doubled/clamped
// numbers (RESEARCH Pitfall 13). darwin's ~9 KiB send limit is documented by
// ADR 0010, not fixed — hence the linux build tag.

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"

	apiproto "github.com/Djarvur/sup-kv/internal/grpcapi/apiproto/supkvapi/v1"
)

const (
	// buffersSyncResponseRate is the --sync-response-rate both nodes run
	// with — passed explicitly by the test, so the paced egress term in the
	// delivery bound is the flag the nodes actually run with.
	buffersSyncResponseRate = 500

	// buffersKeyFmt scopes the scenario's keys.
	buffersKeyFmt = "stand/buffers/key-%d"
)

// buffersValueSizes are XPORT-05's acceptance sizes in bytes: 1 KiB, 20 KiB
// and exactly the default value_max_bytes ceiling of 60 KiB (61 440 bytes).
var buffersValueSizes = [...]int{1024, 20480, 61440}

// TestStand_bufferSizesDeliver proves on real Linux sockets that values of 1,
// 20 and 60 KiB cross between two nodes byte-exact, and that both nodes
// report a positive effective send buffer at startup.
func TestStand_bufferSizesDeliver(t *testing.T) {
	t.Parallel()
	p := newStandPair(t)
	// Both nodes run with the live log capture: the startup buffer line is
	// asserted on both. The response rate is passed explicitly so the paced
	// term in the delivery bound is the flag the nodes actually run with.
	rateFlag := "--sync-response-rate=" + strconv.Itoa(buffersSyncResponseRate)
	nodeA := startStandLoggedNode(t, p, standNodeAID, rateFlag)
	nodeB := startStandLoggedNode(t, p, standNodeBID, rateFlag)
	awaitServing(t, p.grpcA, servingDeadline)
	awaitServing(t, p.grpcB, servingDeadline)

	clientA := standClient(t, p.grpcA)
	clientB := standClient(t, p.grpcB)

	start := time.Now()
	keys := make([]string, len(buffersValueSizes))
	for i, size := range buffersValueSizes {
		keys[i] = fmt.Sprintf(buffersKeyFmt, i)
		standSetWith(t, clientA, keys[i], strings.Repeat("v", size))
	}

	// The delivery bound derives from the flags both nodes run with: two
	// sync rounds for the listing/request/reply ladder, the paced egress
	// term for N records at the design-default response rate (negligible at
	// N=3, but expressed, never a magic constant), node B's startup-gate
	// window, and the discovery/scheduling margin at the shrunken hello
	// interval.
	paced := time.Duration(len(buffersValueSizes)) * (time.Second / buffersSyncResponseRate)
	deadline := 2*standSyncInterval + paced + standStartupGate + 20*standHelloInterval
	waitFor(t, deadline, fmt.Sprintf("the 1/20/60 KiB values delivered byte-exact to node B (bound %v)", deadline), func() bool {
		for i, size := range buffersValueSizes {
			if !buffersServing(t, clientB, keys[i], size) {
				return false
			}
		}
		return true
	})
	for i, size := range buffersValueSizes {
		value, found := standGet(t, clientB, keys[i])
		if !found || len(value) != size {
			t.Fatalf("node B serves %q with found=%t value-bytes=%d, want %d", keys[i], found, len(value), size)
		}
	}
	t.Logf("delivered 1024/20480/61440-byte values in %v (derived bound %v)", time.Since(start), deadline)

	// Both startup lines: positive effective send buffers, the XPORT-05
	// observability half. The kernel's exact numbers are logged, never
	// asserted.
	startupBound := standStartupGate + 20*standHelloInterval
	for _, node := range []struct {
		name string
		proc *standNode
	}{{"node A", nodeA}, {"node B", nodeB}} {
		waitFor(t, startupBound, node.name+"'s Socket buffers startup line with positive sizes", func() bool {
			recv, send, ok := parseStandSocketBuffers(node.proc.stderr.String())
			return ok && recv > 0 && send > 0
		})
		recv, send, ok := parseStandSocketBuffers(node.proc.stderr.String())
		if !ok || recv <= 0 || send <= 0 {
			t.Fatalf("%s Socket buffers line = (recv %d, send %d, ok %t), want both positive", node.name, recv, send, ok)
		}
		t.Logf("%s effective socket buffers: recv_buf=%d send_buf=%d", node.name, recv, send)
	}

	signalAndWaitExit(t, nodeA)
	signalAndWaitExit(t, nodeB)
	assertNoPanic(t, nodeA, "the buffer-delivery node A")
	assertNoPanic(t, nodeB, "the buffer-delivery node B")
}

// buffersServing reports whether client serves key with exactly size value
// bytes. Transient RPC errors read as not-yet (the standServing shape), so
// the caller's waitFor deadline governs, never a fast failure.
func buffersServing(t *testing.T, client apiproto.SupKVServiceClient, key string, size int) bool {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), checkTimeout)
	defer cancel()
	resp, err := client.Get(ctx, &apiproto.GetRequest{Key: key})
	if err != nil {
		return false
	}
	rec := resp.GetRecord()
	return rec.GetKey() == key && !rec.GetDeleted() && len(rec.GetValue()) == size
}

// parseStandSocketBuffers extracts recv_buf and send_buf from a node's startup
// "Socket buffers" JSON log line (the default handler format). ok reports the
// line appeared with both integer attrs present; the caller asserts their
// positivity.
func parseStandSocketBuffers(output string) (recv, send int, ok bool) {
	for line := range strings.SplitSeq(output, "\n") {
		if !strings.Contains(line, "Socket buffers") {
			continue
		}
		var entry struct {
			Msg     string `json:"msg"`
			RecvBuf *int   `json:"recv_buf"`
			SendBuf *int   `json:"send_buf"`
		}
		if json.Unmarshal([]byte(line), &entry) != nil || entry.Msg != "Socket buffers" {
			continue
		}
		if entry.RecvBuf == nil || entry.SendBuf == nil {
			continue
		}
		return *entry.RecvBuf, *entry.SendBuf, true
	}
	return 0, 0, false
}
