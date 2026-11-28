# Security policy

## Reporting a vulnerability

Do not open a public issue for a security problem. Report it through a
**GitHub private security advisory** on the Djarvur/sup-kv repository
(Security tab → Report a vulnerability) — a private disclosure channel that
reaches the maintainers directly. Include the affected version, the
deployment shape (node count, bind addresses) and a reproduction. You will
get an acknowledgment and, for accepted reports, a fix and a coordinated
disclosure.

## Supported versions

Security fixes land on the current release line, **v1.1.x**. Older release
lines do not receive security fixes — upgrade to the latest v1.1.x.

## Scope and trust model

sup-kv ships for one deployment shape: a small cluster (two to a handful of
nodes) on **one trusted LAN broadcast segment**. The full operator contract
— what the system promises, what it does not, and the reasoning — is
[docs/OPERATIONS.md](docs/OPERATIONS.md)'s trust model, which this section
links rather than re-derives; the summary below states the same contract
with the same words, never softer or stronger.

The three absences, stated together exactly as the operator reference states
them: **no authentication** — no client or peer identity is verified anywhere
in the binary; **no authorization** — any client that reaches a node's gRPC
address can read and write everything; **no transport security** — the
replication wire and the gRPC listener are plaintext, and the optional mTLS
certificate triple on the metrics endpoint (`SUPKV_METRICS_CERT`,
`SUPKV_METRICS_KEY`, `SUPKV_METRICS_CLIENT_CA`) is the only TLS surface the
binary has.

Consequences every deployer should weigh:

- **CRC32 in the datagram envelope guards integrity only** — it detects
  accidental corruption in transit; it is not tamper protection.
  Authenticating datagrams (HMAC with a shared key) is documented future
  work, not a v1 feature.
- **gRPC server reflection is always on.** It exposes the full service
  schema to any client that reaches the gRPC port — accepted deliberately:
  the default `--grpc-addr` binds loopback only, the segment is trusted by
  design, and the schema is public in this repository.
- **The metrics/debug endpoint is off by default** and is the only surface
  that can be TLS-protected (opt-in, via the `SUPKV_METRICS_*` environment
  variables); when enabled it also exposes expvar and pprof under `/debug`.
- The default gRPC bind is loopback; binding a wider address is safe only on
  a segment where every reachable client is trusted.

If your threat model includes untrusted parties on the replication segment
or the client network, sup-kv's v1 trust model does not cover you — that is
a recorded, accepted limitation (see the operator reference and ADR 0004's
registry links), not a bug to report.
