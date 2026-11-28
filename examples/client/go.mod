// The module path is deliberately outside the root module's prefix: Go's
// internal rule is prefix-based, so only a module rooted elsewhere makes an
// import of the root module's internal/ tree a compile error (README.md).
module example.com/sup-kv-client

go 1.27.4

require (
	github.com/Djarvur/sup-kv v0.0.0
	google.golang.org/grpc v1.85.0
)

require (
	golang.org/x/net v0.62.0 // indirect
	golang.org/x/sys v0.49.0 // indirect
	golang.org/x/text v0.43.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20261104224025-d1e00e9f33ab // indirect
	google.golang.org/protobuf v1.36.13-0.20261016172727-e9ee2a76ab8e // indirect
)

replace github.com/Djarvur/sup-kv => ../..
