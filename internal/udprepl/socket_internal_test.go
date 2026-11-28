package udprepl

// White-box on purpose (the *_internal_test.go filename is the configured
// testpackage exemption, the syncsize_internal_test.go precedent): the
// buffer-configuration seam is unexported mechanical adapter code, and this
// file drives it over a real loopback socket so the read-back contract — the
// kernel's effective values, never the requested constants — is proven rather
// than assumed (RESEARCH Pattern 7). The two-process 1/20/60 KiB delivery
// acceptance lives in internal/stand/stand_buffers_test.go (XPORT-05).

import (
	"math"
	"net"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"testing"
)

// TestSocketBuffers drives configureSocketBuffers over a real loopback
// socket: both requested sizes are applied, the effective kernel values come
// back positive, the send floor holds, and the returned values equal a direct
// getsockopt read — the read-back contract SocketBuffers reports.
func TestSocketBuffers(t *testing.T) {
	t.Parallel()
	var lc net.ListenConfig
	packetConn, err := lc.ListenPacket(t.Context(), "udp4", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("bind loopback udp4 socket: %v", err)
	}
	conn := packetConn.(*net.UDPConn)
	t.Cleanup(func() { _ = conn.Close() })

	recv, send, err := configureSocketBuffers(conn)
	if err != nil {
		t.Fatalf("configureSocketBuffers: %v", err)
	}
	if recv <= 0 || send <= 0 {
		t.Fatalf("effective buffers = (recv %d, send %d), want both positive", recv, send)
	}
	err = checkSendBuffer(send)
	if err != nil {
		t.Fatalf("effective send buffer %d fails the one-maximal-datagram floor: %v", send, err)
	}
	readbackRecv, err := socketOption(conn, syscall.SO_RCVBUF)
	if err != nil {
		t.Fatalf("read SO_RCVBUF back again: %v", err)
	}
	readbackSend, err := socketOption(conn, syscall.SO_SNDBUF)
	if err != nil {
		t.Fatalf("read SO_SNDBUF back again: %v", err)
	}
	if recv != readbackRecv || send != readbackSend {
		t.Fatalf("configureSocketBuffers returned (%d, %d) but getsockopt reads (%d, %d) — the accessor would report the request, not the kernel",
			recv, send, readbackRecv, readbackSend)
	}
	// The kernel doubles and clamps: asserting an exact doubled value would
	// pin this runner's rmem_max/wmem_max (RESEARCH Pitfall 13). Only the
	// platform-independent shape holds; the observed values are logged for
	// the summary.
	t.Logf("effective socket buffers: SO_RCVBUF=%d (requested %d), SO_SNDBUF=%d (requested %d)",
		recv, socketRecvBuffer, send, socketSendBuffer)
}

// TestCheckSendBuffer pins the send-buffer floor: one byte below the maximal
// datagram fails with both sizes named, the maximal datagram itself passes,
// and a healthy doubled request passes.
func TestCheckSendBuffer(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		actual  int
		wantErr bool
	}{
		{name: "one byte below the maximal datagram", actual: datagramBufSize - 1, wantErr: true},
		{name: "exactly the maximal datagram", actual: datagramBufSize, wantErr: false},
		{name: "the doubled request", actual: 2 * datagramBufSize, wantErr: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := checkSendBuffer(tt.actual)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("checkSendBuffer(%d) = nil, want an error", tt.actual)
				}
				for _, want := range []string{strconv.Itoa(tt.actual), strconv.Itoa(datagramBufSize)} {
					if !strings.Contains(err.Error(), want) {
						t.Fatalf("checkSendBuffer(%d) error %q does not name %s", tt.actual, err, want)
					}
				}
				return
			}
			if err != nil {
				t.Fatalf("checkSendBuffer(%d) = %v, want nil", tt.actual, err)
			}
		})
	}
}

// TestConfigureSocketBuffers_rejectsAbsurdRequestWidth pins the request-width
// guard that runs before any setsockopt call (T-3-01): non-positive requests
// and requests whose doubled read-back would overflow a 32-bit int are
// rejected outright.
func TestConfigureSocketBuffers_rejectsAbsurdRequestWidth(t *testing.T) {
	t.Parallel()
	const maxSafe = math.MaxInt32 / 2
	tests := []struct {
		name      string
		requested int
		wantErr   bool
	}{
		{name: "zero", requested: 0, wantErr: true},
		{name: "negative", requested: -1, wantErr: true},
		{name: "one past the safe width", requested: maxSafe + 1, wantErr: true},
		{name: "max int32", requested: math.MaxInt32, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := validateBufferRequest(tt.requested)
			if tt.wantErr != (err != nil) {
				t.Fatalf("validateBufferRequest(%d) error = %v, want error = %t", tt.requested, err, tt.wantErr)
			}
		})
	}
}

// TestSocketBuffers_boundaries pins the floor and request-width boundaries at
// their exact adjacency points (XPORT-05): 65 507 bytes accepted, 65 506
// rejected, and every width above MaxInt32/2 rejected before any kernel call,
// while MaxInt32/2 itself stays accepted (its doubled read-back still fits a
// 32-bit int). The kernel read-back subtest documents the doubling/clamping
// shape without pinning this runner's rmem_max/wmem_max.
func TestSocketBuffers_boundaries(t *testing.T) {
	t.Parallel()
	t.Run("floor adjacency", func(t *testing.T) {
		t.Parallel()
		tests := []struct {
			name    string
			actual  int
			wantErr bool
		}{
			{name: "65506 rejected", actual: 65506, wantErr: true},
			{name: "65507 accepted", actual: 65507, wantErr: false},
			{name: "131014 accepted", actual: 131014, wantErr: false},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				err := checkSendBuffer(tt.actual)
				if tt.wantErr != (err != nil) {
					t.Fatalf("checkSendBuffer(%d) error = %v, want error = %t", tt.actual, err, tt.wantErr)
				}
				if tt.wantErr && !strings.Contains(err.Error(), strconv.Itoa(datagramBufSize)) {
					t.Fatalf("checkSendBuffer(%d) error %q does not name the maximal datagram %d", tt.actual, err, datagramBufSize)
				}
			})
		}
	})
	t.Run("request width", func(t *testing.T) {
		t.Parallel()
		tests := []struct {
			name      string
			requested int
			wantErr   bool
		}{
			{name: "zero", requested: 0, wantErr: true},
			{name: "negative", requested: -1, wantErr: true},
			{name: "the largest safe width", requested: maxBufferRequestWidth, wantErr: false},
			{name: "one past the largest safe width", requested: maxBufferRequestWidth + 1, wantErr: true},
			{name: "max int32", requested: math.MaxInt32, wantErr: true},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				err := validateBufferRequest(tt.requested)
				if tt.wantErr != (err != nil) {
					t.Fatalf("validateBufferRequest(%d) error = %v, want error = %t", tt.requested, err, tt.wantErr)
				}
			})
		}
	})
	t.Run("kernel read-back", func(t *testing.T) {
		t.Parallel()
		var lc net.ListenConfig
		packetConn, err := lc.ListenPacket(t.Context(), "udp4", "127.0.0.1:0")
		if err != nil {
			t.Fatalf("bind loopback udp4 socket: %v", err)
		}
		conn := packetConn.(*net.UDPConn)
		t.Cleanup(func() { _ = conn.Close() })
		recv, send, err := configureSocketBuffers(conn)
		if err != nil {
			t.Fatalf("configureSocketBuffers: %v", err)
		}
		// The kernel doubles and clamps; a runner's rmem_max/wmem_max may
		// clamp below the requested size and still return a positive value
		// (RESEARCH Pitfall 13). Assert the platform-independent shape only.
		if recv <= 0 || send <= 0 {
			t.Fatalf("effective buffers = (recv %d, send %d), want both positive", recv, send)
		}
		t.Logf("kernel read-back: SO_RCVBUF=%d (requested %d), SO_SNDBUF=%d (requested %d)",
			recv, socketRecvBuffer, send, socketSendBuffer)
	})
}

// TestSocketBuffers_concurrent proves the accessor is immutable state: eight
// goroutines reading one adapter's values while another reader polls a second
// adapter see stable values and the race detector reports no data race.
func TestSocketBuffers_concurrent(t *testing.T) {
	t.Parallel()
	first := &Adapter{recvBuf: 2 * socketRecvBuffer, sendBuf: 2 * socketSendBuffer}
	second := &Adapter{recvBuf: socketRecvBuffer, sendBuf: socketSendBuffer}
	const wantRecv, wantSend = 2 * socketRecvBuffer, 2 * socketSendBuffer

	var wg sync.WaitGroup
	for range 8 {
		wg.Go(func() {
			for range 100 {
				recv, send := first.SocketBuffers()
				if recv != wantRecv || send != wantSend {
					t.Errorf("SocketBuffers() = (%d, %d), want (%d, %d)", recv, send, wantRecv, wantSend)
					return
				}
			}
		})
	}
	wg.Go(func() {
		for range 100 {
			recv, send := second.SocketBuffers()
			if recv != socketRecvBuffer || send != socketSendBuffer {
				t.Errorf("second SocketBuffers() = (%d, %d), want (%d, %d)", recv, send, socketRecvBuffer, socketSendBuffer)
				return
			}
		}
	})
	wg.Wait()
}
