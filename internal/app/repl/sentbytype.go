package repl

// SentByType counts the outbound datagrams the replication adapter has
// encoded and handed to its socket, by message kind (the OBS-01
// replication-topology seam) — the vocabulary mirrors the wire Body oneof.
// It lives in the core package so the metrics exporter can read the value
// through its Sources seam without importing the socket adapter (D-07); the
// adapter keeps a type alias and its accessor so every udprepl test compiles
// unchanged.
type SentByType struct {
	Hello       uint64
	Data        uint64
	Ack         uint64
	DataRequest uint64
	Sync        uint64
}
