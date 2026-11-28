// Command durcheck is a review-only durability probe for sup-kv: "write"
// issues unique-key Sets and records every key the server CONFIRMED;
// "verify" reads them back and reports the confirmed keys that are gone.
package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"

	apiproto "github.com/Djarvur/sup-kv/internal/grpcapi/apiproto/supkvapi/v1"
)

func main() {
	addr := flag.String("addr", "127.0.0.1:7901", "node")
	mode := flag.String("mode", "write", "write|verify")
	file := flag.String("file", "/tmp/confirmed.txt", "confirmed-keys file")
	conc := flag.Int("c", 16, "workers")
	dur := flag.Duration("d", 10*time.Second, "write duration")
	prefix := flag.String("prefix", "dur/", "key prefix")
	size := flag.Int("size", 256, "value size")
	flag.Parse()

	cc, err := grpc.NewClient(*addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		panic(err)
	}
	defer cc.Close()
	cl := apiproto.NewSupKVServiceClient(cc)

	switch *mode {
	case "write":
		f, err := os.Create(*file)
		if err != nil {
			panic(err)
		}
		w := bufio.NewWriter(f)
		var mu sync.Mutex
		var n, ok, fail atomic.Int64
		deadline := time.Now().Add(*dur)
		val := make([]byte, *size)
		var wg sync.WaitGroup
		for i := 0; i < *conc; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for time.Now().Before(deadline) {
					k := fmt.Sprintf("%s%09d", *prefix, n.Add(1))
					ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
					_, err := cl.Set(ctx, &apiproto.SetRequest{Key: k, Value: val})
					cancel()
					if err != nil {
						fail.Add(1)
						time.Sleep(5 * time.Millisecond)
						continue
					}
					ok.Add(1)
					mu.Lock()
					fmt.Fprintln(w, k)
					w.Flush() // confirmed keys reach the file before the next op
					mu.Unlock()
				}
			}()
		}
		wg.Wait()
		f.Close()
		fmt.Printf("{\"attempted\":%d,\"confirmed\":%d,\"failed\":%d}\n", n.Load(), ok.Load(), fail.Load())
	case "verify":
		f, err := os.Open(*file)
		if err != nil {
			panic(err)
		}
		defer f.Close()
		sc := bufio.NewScanner(f)
		var total, missing, other int
		var firstMissing []string
		for sc.Scan() {
			total++
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			_, err := cl.Get(ctx, &apiproto.GetRequest{Key: sc.Text()})
			cancel()
			switch status.Code(err) {
			case codes.OK:
			case codes.NotFound:
				missing++
				if len(firstMissing) < 5 {
					firstMissing = append(firstMissing, sc.Text())
				}
			default:
				other++
			}
		}
		fmt.Printf("{\"confirmed_total\":%d,\"missing\":%d,\"other_errors\":%d,\"first_missing\":%q}\n", total, missing, other, firstMissing)
	}
}
