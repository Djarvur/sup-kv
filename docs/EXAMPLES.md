# sup-kv examples

Every command below is plain `grpcurl` against the shipped `sup-kv` binary — no proto files, no Go client code. The binary serves gRPC server reflection, so `grpcurl` discovers the whole API (`supkvapi.v1.SupKVService`) by itself. Values are proto3 `bytes`, which JSON carries as base64 on the wire: the literal `c3VwLWt2` below is the string `"sup-kv"`. The marked blocks are executed verbatim by CI (`cmd/sup-kv/examples_test.go`), so what you copy from this page is exactly what CI proves works.

### Start a node and make your first call

<!-- examples-verify -->
```bash
sup-kv --node-id 1 --bind-addr 0.0.0.0:28001 --broadcast-addr 127.255.255.255:28002 \
  --grpc-addr 127.0.0.1:28011 --dump-path supkv-ex-a.json & EX_NODE=$!
# Bounded startup wait: the node serves writes once health reports SERVING
# (the startup gate — one second by default on a lone node).
for i in $(seq 1 50); do
  if grpcurl -plaintext -d '{}' 127.0.0.1:28011 grpc.health.v1.Health/Check 2>/dev/null | grep -q '"SERVING"'; then break; fi
  sleep 0.2
done
# Discover the API — zero proto files needed (server reflection).
grpcurl -plaintext 127.0.0.1:28011 list
# Write: value is proto3 bytes ⇒ base64 in JSON (c3VwLWt2 is "sup-kv").
grpcurl -plaintext -d '{"key":"demo/hello","value":"c3VwLWt2"}' \
  127.0.0.1:28011 supkvapi.v1.SupKVService/Set
# Read it back — the value returns in the same base64 form.
grpcurl -plaintext -d '{"key":"demo/hello"}' \
  127.0.0.1:28011 supkvapi.v1.SupKVService/Get
kill $EX_NODE 2>/dev/null; wait $EX_NODE 2>/dev/null || true
```
