# File and datagram formats

This file is the as-built reference for sup-kv's two machine-facing formats:
the JSONL dump generation a node writes and reloads (`internal/app/persist`
codec) and the UDP replication datagram envelope (`internal/udprepl` codec
over the internal protobuf schema). It supersedes the historical intent
document's one-JSON-document description in `docs/IDEA.md`: the dump is JSON
Lines, not one JSON document.

Every factual claim here is grounded in the codecs' own contracts and pinned
by tests. The example dumps below are **the encoder's own output**: they are
`persist.EncodeTo` bytes over the fixed record set defined in
`internal/app/persist/formats_doc_test.go`, committed verbatim.
`TestFormatsDocEncodeMatchesExample` compares them byte-for-byte — a codec
format change without a doc regeneration fails `make test`, and the failing
test names the first differing line. That is the maintenance contract: to
change the format, change the codec, regenerate the fences, re-commit.
`TestFormatsDocChecksumRecipe` recomputes the example's checksums with the
recipe stated below, and `TestFormatsDocRejectionRules` checks that the
rejection rules name exactly the discard classes `persist.DecodeStats`
defines.

Value-size limits are configured, not format-fixed; the operator-facing
limits prose (`--value-max-bytes`, the 1 024-byte key limit and the
65 507-byte datagram ceiling) lives in `docs/OPERATIONS.md` under
`## Record and datagram limits`, and the flags that set them in
`docs/CONFIGURATION.md`.

## Dump grammar

A dump generation is **JSON Lines**: one JSON object per line, each line
terminated by a newline (a CRLF line ending reads the same as an LF one).

**Line 1 is the header object** with two fields:

- `format_version` — the dump format version this generation was written
  with; the only current value is `1`.
- `record_count` — the number of record lines the writer emitted (the
  completeness census, added while the format stayed at version 1; see
  `## Format-version policy`).

**Every following line is one record object**, sorted by key. The sort is
mandatory: Go map iteration order is random, and byte-identical dump
generations for identical store state are the contract — a reload, a
snapshot copy and a restart must all observe the same bytes. The header
carries no timestamp or other volatile field for the same reason. The
example's lines are in key order whatever order the records were authored
in: the pinning test's fixed record set lists `charlie`, `alpha`, `bravo`.

Record-line fields, in wire order:

| Field | Encoding | Meaning |
|-------|----------|---------|
| `key` | JSON string | the record's key (up to 1 024 bytes; read-side keys above that remain readable — key validation is write-side only) |
| `record_version` | 32 lowercase hex characters (16 bytes) | the record's version — see `## Record-version layout` |
| `data` | base64 string, or `null` | the record's value bytes; `null` when there are none |
| `deleted` | JSON boolean | `true` marks a tombstone: the key was deleted and the record exists to win the LWW join |
| `checksum` | unsigned decimal | the CRC32 over the record — see `## Checksum` |

Two edge shapes the example carries explicitly:

- **A tombstone** is `"data":null,"deleted":true` — the writer stores no
  value bytes on a deleted record, and JSON renders a nil `[]byte` as
  `null`.
- **A live empty value** is `"data":""` — a zero-length non-nil `[]byte`
  renders as the empty base64 string, so a live empty value is
  distinguishable from a tombstone by the `deleted` flag alone.

The writer escapes no HTML: `&`, `<` and `>` in a key stay literal and
greppable. Each line is written in one write call, so a caller-supplied
writer streams the generation without holding it in memory.

A record whose value exceeds the configured value limit can never be
encoded: the encoder rejects it with an error wrapping
`storage.ErrValueTooLarge` naming the key — an invariant tripwire over the
store's own write-side check, never a silent skip.

The three-record example — `alpha` holds the live 11-byte value
`hello-alpha`, `bravo` is a tombstone, `charlie` holds a live empty value;
all three versions were minted by node `0x5060` one millisecond apart at
logical time `1736937600000` (2025-01-15T10:40:00.000Z):

<!-- formats-example -->
```dump
{"format_version":1,"record_count":3}
{"key":"alpha","record_version":"0194698c740070019418000000000000","data":"aGVsbG8tYWxwaGE=","deleted":false,"checksum":2634627855}
{"key":"bravo","record_version":"0194698c740170019418000000000000","data":null,"deleted":true,"checksum":2110500349}
{"key":"charlie","record_version":"0194698c740270019418000000000000","data":"","deleted":false,"checksum":1011420308}

```

The zero-record example — an empty store encodes to the header alone:

<!-- formats-example-empty -->
```dump
{"format_version":1,"record_count":0}

```

## Checksum

Every record line carries a checksum over the record's own fields. The
recipe is the one checksum domain the store defines
(`storage.RecordChecksum`):

> **CRC32-IEEE**, starting from zero, updated sequentially over
> 1. the key's bytes,
> 2. the 16 record-version bytes (in packed order — the `record_version`
>    hex string decoded),
> 3. the data bytes (none when `data` is `null` or `""`),
> 4. one flag byte: `0x00` for a live record, `0x01` for a tombstone.

The checksum is stored as an unsigned decimal JSON number. When a dump is
read back, every accepted record's checksum is re-checked against the line's
own fields; a mismatch discards that record (`class=checksum`, below) and
never its siblings.

The checksum is corruption detection, never tamper protection: anyone who
can rewrite a dump file can recompute a matching checksum. Tamper resistance
is out of scope by the accepted trust model — the same holds for the
datagram CRC (below).

## Rejection rules

Reading a generation back classifies every fault into exactly one of two
levels.

**Per-record faults discard only their record** and count it in the matching
class (`persist.DecodeStats`; the class names are the metric-label
vocabulary `supkv_dump_records_discarded_total` uses). The sibling lines
still load; one torn record never collapses its generation:

- **`class=malformed`** — the line is not a JSON object (a syntax error, a
  non-object value). A duplicate `key` in a hand-edited file is not a fault:
  the last line for a key wins.
- **`class=version`** — `record_version` is not exactly 16 bytes of hex.
  The dump loader does not re-validate the UUIDv7 layout or any future
  bound here; that validation lives on the replication receive path (see
  `## Record-version layout`), so a dump is the one place an older or
  non-conforming version still loads.
- **`class=value_limit`** — the record's data exceeds the configured value
  limit. The limit is the same `Config.ValueLimit()` value the store
  enforces on writes; it is applied after the whole line is read, so the
  reader's own line bound never shrinks with the configuration — a dump
  written at the default limit still reads under a tiny new limit, with
  over-limit records counted instead of faulting the generation.
- **`class=checksum`** — the checksum does not cover the line: recomputing
  the `## Checksum` recipe over the line's own fields does not reproduce
  the `checksum` value.

**Generation-level faults reject the whole generation** — no records, zero
stats — so the startup fallback chain (main → `.prev` → empty start)
advances:

- An empty or header-unreadable input: the first line is missing or is not
  a valid header object.
- **An unknown `format_version`** (see `## Format-version policy`).
- **A record line longer than the format maximum** — a bound derived from
  the datagram ceiling's base64 expansion plus the worst-case JSON escaping
  of a maximal key. No supported writer can emit a longer line, so one is a
  generation fault, not an isolatable record.
- **An unterminated final fragment that does not parse** — a cut mid-line
  must not collapse into a shorter successful load. A fragment that does
  parse as a record is accepted.
- **Fewer record lines than `record_count`** — the generation was cut at a
  line boundary, which is indistinguishable from a shorter complete file
  without the census, so the census makes it an **incomplete** generation.

The `record_count` census rules in full:

- Fewer record lines than the announced `record_count` is the incomplete
  generation fault above.
- More record lines than the announced count is **tolerated** — hand edits
  and duplicate keys parse normally.
- An absent `record_count` field is the documented **legacy** path: a v1.0
  dump carries no census, so a line-boundary cut of one still loads — the
  documented compatibility limitation — and its final unterminated tail
  must still parse.

## Format-version policy

- **Version 1 is the only current format version.** The header announces it
  in `format_version`.
- `record_count` was added to the same version as an optional field rather
  than bumping the version: an old reader ignores an unknown additive
  field, whereas a version bump would make a downgrade reject every new
  dump.
- **An unknown `format_version` makes the generation unreadable**: the
  reader returns a generation-level fault and the fallback chain advances.
  A future reader upgrade announces a higher version; this build never
  reads one.
- **v1.0 files load unchanged** through the legacy path above (no
  `record_count`, no completeness census). No migration step exists or is
  needed: the record-line shape has not changed since v1.0.

## Datagram envelope and CRC coverage

Replication travels as **one protobuf `Envelope` per UDP datagram**. The
envelope fields:

| Field | Type / domain | Meaning |
|-------|---------------|---------|
| `proto_version` | `uint32` | the protocol version the sender speaks; this build speaks `2` |
| `node_id` | `uint32` on the wire, 16-bit domain enforced | the sending node's configured `node_id`; a value above `0xFFFF` is undecodable |
| `run_id` | exactly 16 bytes | the sending process's run identity; a new `run_id` from a known node means it restarted; a wrong length is undecodable |
| `crc32` | `uint32` | the IEEE CRC32 over `body` |
| `body` | bytes | one marshaled `Body` — a oneof carrying exactly one protocol message (`Hello`, `DataReplication`, `Ack`, `DataRequest`, `DataSync`) |

**CRC32 coverage: the CRC covers the marshaled body bytes only**, not the
whole envelope. The header fields are readable and classifiable even when
the body is corrupt, which is what makes the verdict order below possible.

**The receiver's verdict order is fixed** and each verdict names a discard
class on the replication side (catalogued in `docs/OPERATIONS.md`):

1. An empty datagram or an envelope that fails to unmarshal is
   undecodable — nothing can be said about it.
2. A wire-domain violation (`node_id` above `0xFFFF`, a `run_id` that is
   not 16 bytes) is undecodable.
3. **`proto_version` is checked before the CRC and the body**: a foreign
   protocol version is a mismatch verdict carrying the envelope fields
   over, with the body never decoded — a foreign version's body need not
   follow this version's framing contract, so its CRC need not be trusted.
4. A CRC mismatch (`crc32.ChecksumIEEE(body)` not equal to the announced
   `crc32`) is a bad-CRC verdict; the body is not unmarshaled.
5. Only a passing CRC unmarshals the body: a body that fails to unmarshal,
   carries an unknown oneof case, or carries a version field that is not
   exactly 16 bytes is undecodable; otherwise the datagram is decoded.

Like the dump checksum, the datagram CRC32 is corruption detection only:
the replication segment is trusted, and authentication/encryption (HMAC) is
documented future work. The datagram sizing rules — the 65 507-byte ceiling
the combined record must fit, and the resulting 64 408-byte value maximum
at a maximal key — are in `docs/OPERATIONS.md` under
`## Record and datagram limits`.

## Record-version layout

A `record_version` is a 16-byte **RFC 9562 UUIDv7** carrying the writing
node's hybrid logical clock reading and identity:

| Octets | Content |
|--------|---------|
| 0–5 | the 48-bit logical milliseconds (`l`, the HLC reading) — the `unix_ts_ms` field |
| 6–7 | the RFC 9562 version nibble `7` in the high bits of octet 6, then the 12-bit counter in the remaining `rand_a` bits |
| 8–9 | the RFC 9562 variant bits `0b10` in the top two bits of octet 8, then the node ID's high 14 bits |
| 10 | the node ID's low 2 bits in the top bits; the remaining bits are zero |
| 11–15 | zero — the 46 `rand_b` filler bits this implementation leaves zero |

Properties that matter when reading a dump by hand:

- **Byte-wise comparison equals `(l, counter, nodeID)` order** — the
  big-endian layout puts the fields in exactly that priority, which is the
  total order the LWW join and the sync comparison sort by.
- **Rendering is 32 lowercase hex characters**, two per octet, in octet
  order. The example's `alpha` version `0194698c740070019418000000000000`
  unpacks as `l=1736937600000, counter=1, node=0x5060`.
- The variant bits carry no per-record information; the 46 filler bits are
  deliberately **not** rejected anywhere, so a conforming UUIDv7 minted by
  another vendor passes.
- The dump loader does not validate this layout (see the `class=version`
  rule above). The replication receive path does: it checks the RFC 9562
  nibble and variant bits, then rejects a reading beyond the configured
  `--max-clock-lead` future bound — a crafted version can never poison the
  hybrid logical clock.

## Manual decoding

Reading the three-record example by hand, with standard tools only:

1. **The header line** says the generation is format version `1` and
   announces `record_count` `3` — expect exactly three record lines; fewer
   is an incomplete generation.
2. **`alpha`** is a live record (`deleted` `false`) whose `data`
   `aGVsbG8tYWxwaGE=` base64-decodes to the 11 bytes `hello-alpha`. Its
   version decodes to logical time `1736937600000`, counter `1`, node
   `0x5060` (see `## Record-version layout`).
3. **Verify `alpha`'s checksum** `2634627855` with the recipe: CRC32-IEEE
   from zero over the 5 bytes `alpha`, the 16 version bytes, the 11 data
   bytes, and the live flag `0x00`.
4. **`bravo`** is the tombstone shape: `deleted` `true`, `data` `null` —
   there are no data bytes, and its flag byte is `0x01`. Its checksum
   `2110500349` verifies over its own key, version and that flag.
5. **`charlie`** is the live empty-value shape: `data` `""` (zero data
   bytes, flag `0x00`), `deleted` `false`. Its checksum `1011420308`
   verifies over its key, version and flag — equal to `bravo`'s only in
   having no data bytes, not in value.
6. **Cross-check the census**: three record lines read, `record_count` `3`
   — the generation is complete.

For automated inspection, `persist.Decode` performs exactly these steps and
reports per-record discard classes; the node's own startup load is that
same code path, so what an operator reconstructs by hand is what the binary
computed.
