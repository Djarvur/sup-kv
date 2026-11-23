// Command client is a standalone example consumer of the sup-kv gRPC API.
//
// It is a separate Go module on purpose, and its module path
// (example.com/sup-kv-client) sits outside the root module's prefix: Go's
// internal rule is prefix-based, so an import of the root module's internal/
// tree from here is a compile error. That makes the example's build the
// compile-time second proof of consumer importability; the enforced boundary
// inside the repository is the depguard rule sup-kv-public-api-no-internal
// (api/** may not import the internal/ tree). It is a demonstration, not a
// client library — see README.md, including why a last-read-value cache
// belongs in the consumer's own library (ADR 0007, decision 14.7).
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	supkvapi "github.com/Djarvur/sup-kv/api/supkvapi/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	// rpcTimeout bounds every unary call so a wedged node fails the example
	// instead of hanging it.
	rpcTimeout = 5 * time.Second

	// watchWindow bounds the Watch demonstration: the example prints one
	// event and then ends, treating the window closing as a clean finish.
	watchWindow = 3 * time.Second
)

func main() {
	addr := flag.String("addr", "127.0.0.1:28011", "gRPC address of a sup-kv node")
	prefix := flag.String("prefix", "example/", "key prefix written, listed and watched by the example")
	flag.Parse()

	if err := run(context.Background(), *addr, *prefix); err != nil {
		fmt.Fprintf(os.Stderr, "client example: %v\n", err)
		os.Exit(1)
	}
}

// run executes the four demonstrated RPC patterns in order and returns the
// first error, so the process exits non-zero on any failure.
func run(ctx context.Context, addr, prefix string) error {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return fmt.Errorf("dial %s: %w", addr, err)
	}
	defer func() { _ = conn.Close() }()

	client := supkvapi.NewSupKVServiceClient(conn)
	key := prefix + "hello"
	value := []byte("hello from the sup-kv client example")

	if err := setWithWait(ctx, client, key, value); err != nil {
		return err
	}
	if err := get(ctx, client, key); err != nil {
		return err
	}
	if err := listPages(ctx, client, prefix); err != nil {
		return err
	}
	return watchOne(ctx, client, prefix)
}

// setWithWait writes one key and asks the node to report the replication
// outcome. wait_for_replication is honest: total_peers is the set known when
// the write was accepted, and acked is true only when every one of them
// acknowledged within wait_timeout_ms.
func setWithWait(ctx context.Context, client supkvapi.SupKVServiceClient, key string, value []byte) error {
	callCtx, cancel := context.WithTimeout(ctx, rpcTimeout)
	defer cancel()

	resp, err := client.Set(callCtx, &supkvapi.SetRequest{
		Key:                key,
		Value:              value,
		WaitForReplication: true,
		WaitTimeoutMs:      1000,
	})
	if err != nil {
		return fmt.Errorf("Set %q: %w", key, err)
	}

	fmt.Printf("Set %s: version=%x acked=%t acked_count=%d total_peers=%d\n",
		key, resp.GetVersion(), resp.GetAcked(), resp.GetAckedCount(), resp.GetTotalPeers())
	return nil
}

// get reads the record back through the point-read RPC.
func get(ctx context.Context, client supkvapi.SupKVServiceClient, key string) error {
	callCtx, cancel := context.WithTimeout(ctx, rpcTimeout)
	defer cancel()

	resp, err := client.Get(callCtx, &supkvapi.GetRequest{Key: key})
	if err != nil {
		return fmt.Errorf("Get %q: %w", key, err)
	}
	rec := resp.GetRecord()
	if rec == nil {
		return fmt.Errorf("Get %q: response carries no record", key)
	}

	fmt.Printf("Get %s: value=%q version=%x\n", rec.GetKey(), rec.GetValue(), rec.GetVersion())
	return nil
}

// listPages walks every page of the prefix. An empty next_page_token is the
// end-of-list signal; a page token is an opaque position, not a snapshot.
func listPages(ctx context.Context, client supkvapi.SupKVServiceClient, prefix string) error {
	token := ""
	for page := 1; ; page++ {
		callCtx, cancel := context.WithTimeout(ctx, rpcTimeout)
		resp, err := client.List(callCtx, &supkvapi.ListRequest{
			Prefix:    prefix,
			PageSize:  2,
			PageToken: token,
		})
		cancel()
		if err != nil {
			return fmt.Errorf("List %q (page %d, token %q): %w", prefix, page, token, err)
		}

		fmt.Printf("List page %d: keys=%v next_page_token=%q\n",
			page, resp.GetKeys(), resp.GetNextPageToken())
		for _, rec := range resp.GetRecords() {
			fmt.Printf("  record key=%s version=%x value=%q deleted=%t\n",
				rec.GetKey(), rec.GetVersion(), rec.GetValue(), rec.GetDeleted())
		}

		token = resp.GetNextPageToken()
		if token == "" {
			return nil
		}
	}
}

// watchOne subscribes to the prefix and prints exactly one record — the
// server sends the matching snapshot first, so the write above is visible
// immediately. The bounded window closing is a clean end for the example.
func watchOne(ctx context.Context, client supkvapi.SupKVServiceClient, prefix string) error {
	watchCtx, cancel := context.WithTimeout(ctx, watchWindow)
	defer cancel()

	stream, err := client.Watch(watchCtx, &supkvapi.WatchRequest{Prefix: prefix})
	if err != nil {
		return fmt.Errorf("Watch %q: %w", prefix, err)
	}

	resp, err := stream.Recv()
	if err != nil {
		if watchCtx.Err() != nil {
			fmt.Println("Watch: bounded window closed with no event (clean end)")
			return nil
		}
		return fmt.Errorf("Watch %q: %w", prefix, err)
	}

	rec := resp.GetRecord()
	fmt.Printf("Watch %s: record key=%s version=%x value=%q deleted=%t\n",
		prefix, rec.GetKey(), rec.GetVersion(), rec.GetValue(), rec.GetDeleted())
	return nil
}
