// Command loadgen is a review-only load generator for the sup-kv gRPC API.
// It lives in the scratch copy of the repository (the API package is
// internal) and is not part of the product.
package main

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"

	apiproto "github.com/Djarvur/sup-kv/internal/grpcapi/apiproto/supkvapi/v1"
)

type result struct {
	Mode       string           `json:"mode"`
	Label      string           `json:"label,omitempty"`
	Conc       int              `json:"conc"`
	Conns      int              `json:"conns"`
	Rate       int              `json:"target_rate"`
	Keys       int              `json:"keys"`
	Size       int              `json:"value_size"`
	Wait       bool             `json:"wait_for_replication"`
	Duration   float64          `json:"duration_s"`
	Ops        int64            `json:"ops_ok"`
	Errors     map[string]int64 `json:"errors"`
	Throughput float64          `json:"throughput_ops_s"`
	LatMs      map[string]float64 `json:"latency_ms"`
	ReplMs     map[string]float64 `json:"repl_lag_ms,omitempty"`
	ReplSeen   int64            `json:"repl_events_seen,omitempty"`
	ReplMissed int64            `json:"repl_events_missed,omitempty"`
	Acked      int64            `json:"acked_true,omitempty"`
}

func pct(sorted []time.Duration, p float64) float64 {
	if len(sorted) == 0 {
		return 0
	}
	idx := int(float64(len(sorted))*p/100.0+0.5) - 1
	if idx < 0 {
		idx = 0
	}
	if idx >= len(sorted) {
		idx = len(sorted) - 1
	}
	return float64(sorted[idx].Microseconds()) / 1000.0
}

func summarize(ds []time.Duration) map[string]float64 {
	sort.Slice(ds, func(i, j int) bool { return ds[i] < ds[j] })
	var sum time.Duration
	for _, d := range ds {
		sum += d
	}
	m := map[string]float64{
		"p50": pct(ds, 50), "p90": pct(ds, 90), "p99": pct(ds, 99), "p99.9": pct(ds, 99.9),
	}
	if len(ds) > 0 {
		m["min"] = float64(ds[0].Microseconds()) / 1000.0
		m["max"] = float64(ds[len(ds)-1].Microseconds()) / 1000.0
		m["mean"] = float64(sum.Microseconds()) / 1000.0 / float64(len(ds))
	}
	return m
}

func main() {
	var (
		addr    = flag.String("addr", "127.0.0.1:7901", "target node gRPC address")
		addr2   = flag.String("addr2", "", "second node gRPC address (repl mode: watch target)")
		mode    = flag.String("mode", "set", "set|get|mixed|repl|preload")
		conc    = flag.Int("c", 1, "concurrent workers")
		conns   = flag.Int("conns", 1, "gRPC connections (workers are spread round-robin)")
		dur     = flag.Duration("d", 10*time.Second, "duration")
		keys    = flag.Int("keys", 1000, "key space size")
		size    = flag.Int("size", 1024, "value size in bytes")
		rate    = flag.Int("rate", 0, "open-loop target ops/s (0 = closed loop)")
		wait    = flag.Bool("wait", false, "set wait_for_replication on writes")
		prefix  = flag.String("prefix", "bench/", "key prefix")
		readPct = flag.Int("read-pct", 90, "mixed mode: percentage of reads")
		label   = flag.String("label", "", "free-form label copied into the result")
		timeout = flag.Duration("timeout", 10*time.Second, "per-RPC deadline")
	)
	flag.Parse()

	ccs := make([]apiproto.SupKVServiceClient, *conns)
	for i := range ccs {
		cc, err := grpc.NewClient(*addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			fmt.Fprintln(os.Stderr, "dial:", err)
			os.Exit(1)
		}
		defer cc.Close()
		ccs[i] = apiproto.NewSupKVServiceClient(cc)
	}

	payload := make([]byte, *size)
	_, _ = rand.Read(payload)

	keyOf := func(i int) string { return fmt.Sprintf("%s%07d", *prefix, i) }

	res := result{Mode: *mode, Label: *label, Conc: *conc, Conns: *conns, Rate: *rate, Keys: *keys, Size: *size, Wait: *wait, Errors: map[string]int64{}}

	if *mode == "preload" {
		start := time.Now()
		var wg sync.WaitGroup
		var next atomic.Int64
		var okN, errN atomic.Int64
		for w := 0; w < *conc; w++ {
			wg.Add(1)
			go func(w int) {
				defer wg.Done()
				cl := ccs[w%len(ccs)]
				for {
					i := int(next.Add(1)) - 1
					if i >= *keys {
						return
					}
					ctx, cancel := context.WithTimeout(context.Background(), *timeout)
					_, err := cl.Set(ctx, &apiproto.SetRequest{Key: keyOf(i), Value: payload})
					cancel()
					if err != nil {
						errN.Add(1)
					} else {
						okN.Add(1)
					}
				}
			}(w)
		}
		wg.Wait()
		res.Ops = okN.Load()
		res.Errors["any"] = errN.Load()
		res.Duration = time.Since(start).Seconds()
		res.Throughput = float64(res.Ops) / res.Duration
		out, _ := json.Marshal(res)
		fmt.Println(string(out))
		return
	}

	// repl mode: Watch on addr2, measure Set-start -> event-on-peer lag.
	var (
		replMu   sync.Mutex
		replLags []time.Duration
		sent     sync.Map // seq(uint64) -> time.Time
		replSeen atomic.Int64
		t0       = time.Now()
	)
	if *mode == "repl" {
		if *addr2 == "" {
			fmt.Fprintln(os.Stderr, "repl mode needs -addr2")
			os.Exit(1)
		}
		cc2, err := grpc.NewClient(*addr2, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			fmt.Fprintln(os.Stderr, "dial2:", err)
			os.Exit(1)
		}
		defer cc2.Close()
		wctx, wcancel := context.WithCancel(context.Background())
		defer wcancel()
		st, err := apiproto.NewSupKVServiceClient(cc2).Watch(wctx, &apiproto.WatchRequest{Prefix: *prefix})
		if err != nil {
			fmt.Fprintln(os.Stderr, "watch:", err)
			os.Exit(1)
		}
		go func() {
			for {
				ev, err := st.Recv()
				if err != nil {
					return
				}
				now := time.Now()
				v := ev.GetRecord().GetValue()
				if len(v) < 16 {
					continue
				}
				if binary.BigEndian.Uint64(v[0:8]) != uint64(t0.UnixNano()) {
					continue // snapshot leftovers from earlier runs
				}
				seq := binary.BigEndian.Uint64(v[8:16])
				if ts, ok := sent.LoadAndDelete(seq); ok {
					replMu.Lock()
					replLags = append(replLags, now.Sub(ts.(time.Time)))
					replMu.Unlock()
					replSeen.Add(1)
				}
			}
		}()
		time.Sleep(300 * time.Millisecond) // let the snapshot phase drain
	}

	var (
		mu     sync.Mutex
		lats   []time.Duration
		okN    atomic.Int64
		ackedN atomic.Int64
		seqN   atomic.Uint64
		errMu  sync.Mutex
	)
	record := func(d time.Duration, err error) {
		if err != nil {
			c := status.Code(err)
			name := c.String()
			if c == codes.Unknown {
				name = "Unknown:" + strings.SplitN(err.Error(), "\n", 2)[0]
			}
			errMu.Lock()
			res.Errors[name]++
			errMu.Unlock()
			return
		}
		okN.Add(1)
		mu.Lock()
		lats = append(lats, d)
		mu.Unlock()
	}

	doOp := func(cl apiproto.SupKVServiceClient, n uint64, scheduled time.Time) {
		ctx, cancel := context.WithTimeout(context.Background(), *timeout)
		defer cancel()
		k := keyOf(int(n % uint64(*keys)))
		isRead := *mode == "get" || (*mode == "mixed" && int(n%100) < *readPct)
		start := scheduled
		if start.IsZero() {
			start = time.Now()
		}
		if isRead {
			_, err := cl.Get(ctx, &apiproto.GetRequest{Key: k})
			if status.Code(err) == codes.NotFound {
				err = nil
			}
			record(time.Since(start), err)
			return
		}
		val := payload
		if *mode == "repl" {
			val = make([]byte, len(payload))
			copy(val, payload)
			if len(val) < 16 {
				val = make([]byte, 16)
			}
			seq := seqN.Add(1)
			binary.BigEndian.PutUint64(val[0:8], uint64(t0.UnixNano()))
			binary.BigEndian.PutUint64(val[8:16], seq)
			sent.Store(seq, time.Now())
		}
		resp, err := cl.Set(ctx, &apiproto.SetRequest{Key: k, Value: val, WaitForReplication: *wait})
		if err == nil && resp.GetAcked() {
			ackedN.Add(1)
		}
		record(time.Since(start), err)
	}

	begin := time.Now()
	deadline := begin.Add(*dur)
	var wg sync.WaitGroup
	if *rate <= 0 {
		var n atomic.Uint64
		for w := 0; w < *conc; w++ {
			wg.Add(1)
			go func(w int) {
				defer wg.Done()
				cl := ccs[w%len(ccs)]
				for time.Now().Before(deadline) {
					doOp(cl, n.Add(1), time.Time{})
				}
			}(w)
		}
	} else {
		// Open loop: operations are scheduled on a fixed timeline; latency is
		// measured from the scheduled time, so queueing behind a slow server
		// is counted (no coordinated omission).
		type job struct {
			n  uint64
			at time.Time
		}
		ch := make(chan job, *rate*int(dur.Seconds()+1))
		for w := 0; w < *conc; w++ {
			wg.Add(1)
			go func(w int) {
				defer wg.Done()
				cl := ccs[w%len(ccs)]
				for j := range ch {
					doOp(cl, j.n, j.at)
				}
			}(w)
		}
		interval := time.Second / time.Duration(*rate)
		var i uint64
		for {
			at := begin.Add(time.Duration(i) * interval)
			if at.After(deadline) {
				break
			}
			if d := time.Until(at); d > 0 {
				time.Sleep(d)
			}
			ch <- job{n: i, at: at}
			i++
		}
		close(ch)
	}
	wg.Wait()
	elapsed := time.Since(begin)

	if *mode == "repl" {
		// Give in-flight replication a grace period to arrive.
		grace := time.Now().Add(3 * time.Second)
		for time.Now().Before(grace) && replSeen.Load() < okN.Load() {
			time.Sleep(20 * time.Millisecond)
		}
		replMu.Lock()
		res.ReplMs = summarize(replLags)
		replMu.Unlock()
		res.ReplSeen = replSeen.Load()
		var missed int64
		sent.Range(func(_, _ any) bool { missed++; return true })
		res.ReplMissed = missed
	}

	res.Ops = okN.Load()
	res.Acked = ackedN.Load()
	res.Duration = elapsed.Seconds()
	res.Throughput = float64(res.Ops) / elapsed.Seconds()
	res.LatMs = summarize(lats)
	out, _ := json.Marshal(res)
	fmt.Println(string(out))
}
