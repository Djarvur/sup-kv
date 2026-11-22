package supkvapi_test

import (
	"testing"

	"google.golang.org/protobuf/reflect/protoreflect"

	supkvapi "github.com/Djarvur/sup-kv/api/supkvapi/v1"
)

// fieldPin is one expected field of a message: number, name, kind and
// cardinality, plus the full message name for message-typed fields.
type fieldPin struct {
	name     protoreflect.Name
	number   protoreflect.FieldNumber
	kind     protoreflect.Kind
	message  protoreflect.FullName
	repeated bool
}

// messagePin is one expected message: its name and its complete field set.
type messagePin struct {
	name   protoreflect.Name
	fields []fieldPin
}

// assertFields pins every field of one message and asserts no field was added
// or removed: the wire contract is a published artifact.
func assertFields(t *testing.T, msg protoreflect.MessageDescriptor, pins []fieldPin) {
	t.Helper()

	if got, want := msg.Fields().Len(), len(pins); got != want {
		t.Errorf("%s has %d fields, want %d", msg.FullName(), got, want)
	}

	for _, pin := range pins {
		field := msg.Fields().ByNumber(pin.number)
		if field == nil {
			t.Errorf("%s has no field number %d (%q)", msg.FullName(), pin.number, pin.name)

			continue
		}

		if got := field.Name(); got != pin.name {
			t.Errorf("%s field %d name = %q, want %q", msg.FullName(), pin.number, got, pin.name)
		}

		if got := field.Kind(); got != pin.kind {
			t.Errorf("%s.%s kind = %v, want %v", msg.FullName(), pin.name, got, pin.kind)
		}

		if got := field.IsList(); got != pin.repeated {
			t.Errorf("%s.%s repeated = %v, want %v", msg.FullName(), pin.name, got, pin.repeated)
		}

		if pin.message == "" {
			continue
		}

		if field.Message() == nil {
			t.Errorf("%s.%s is not a message field", msg.FullName(), pin.name)

			continue
		}

		if got := field.Message().FullName(); got != pin.message {
			t.Errorf("%s.%s message = %q, want %q", msg.FullName(), pin.name, got, pin.message)
		}
	}
}

// TestSchema_serviceDescriptor pins the package, the service and its seven
// methods: Watch is the only server-streaming method, no method is
// client-streaming.
func TestSchema_serviceDescriptor(t *testing.T) {
	t.Parallel()

	file := supkvapi.File_supkvapi_v1_supkvapi_proto
	if got, want := file.Package(), protoreflect.FullName("supkvapi.v1"); got != want {
		t.Fatalf("package = %q, want %q", got, want)
	}

	service := file.Services().ByName("SupKVService")
	if service == nil {
		t.Fatal("service SupKVService is missing from the wire descriptor")
	}

	want := []struct {
		name          protoreflect.Name
		serverStreams bool
	}{
		{name: "Get"},
		{name: "Set"},
		{name: "Delete"},
		{name: "List"},
		{name: "Watch", serverStreams: true},
		{name: "SnapshotCreate"},
		{name: "SnapshotLoad"},
	}

	methods := service.Methods()
	if got := methods.Len(); got != len(want) {
		t.Fatalf("service %s has %d methods, want %d", service.FullName(), got, len(want))
	}

	for i, pin := range want {
		method := methods.Get(i)
		if got := method.Name(); got != pin.name {
			t.Errorf("method %d = %q, want %q", i, got, pin.name)
		}

		if got := method.IsStreamingServer(); got != pin.serverStreams {
			t.Errorf("%s server-streaming = %v, want %v", pin.name, got, pin.serverStreams)
		}

		if method.IsStreamingClient() {
			t.Errorf("%s is client-streaming, want unary client", pin.name)
		}
	}
}

// TestSchema_recordFields pins the Record message: key=1, version=2, value=3,
// deleted=4, with the cardinalities and kinds the wire has carried since the
// first release.
func TestSchema_recordFields(t *testing.T) {
	t.Parallel()

	assertFields(t, (&supkvapi.Record{}).ProtoReflect().Descriptor(), []fieldPin{
		{name: "key", number: 1, kind: protoreflect.StringKind},
		{name: "version", number: 2, kind: protoreflect.BytesKind},
		{name: "value", number: 3, kind: protoreflect.BytesKind},
		{name: "deleted", number: 4, kind: protoreflect.BoolKind},
	})
}

// TestSchema_listPageFields pins the List pagination contract: the request
// carries an additive page_size=2 and page_token=3 after prefix=1, and the
// response carries next_page_token=2 and the page's records=3 after the
// retained keys=1.
func TestSchema_listPageFields(t *testing.T) {
	t.Parallel()

	assertFields(t, (&supkvapi.ListRequest{}).ProtoReflect().Descriptor(), []fieldPin{
		{name: "prefix", number: 1, kind: protoreflect.StringKind},
		{name: "page_size", number: 2, kind: protoreflect.Int32Kind},
		{name: "page_token", number: 3, kind: protoreflect.StringKind},
	})
	assertFields(t, (&supkvapi.ListResponse{}).ProtoReflect().Descriptor(), []fieldPin{
		{name: "keys", number: 1, kind: protoreflect.StringKind, repeated: true},
		{name: "next_page_token", number: 2, kind: protoreflect.StringKind},
		{name: "records", number: 3, kind: protoreflect.MessageKind, message: "supkvapi.v1.Record", repeated: true},
	})
}

// TestSchema_otherMessages pins every remaining message so the move and the
// additive List fields cannot renumber, retype or drop an existing field.
func TestSchema_otherMessages(t *testing.T) {
	t.Parallel()

	file := supkvapi.File_supkvapi_v1_supkvapi_proto
	pins := []messagePin{
		{
			name:   "GetRequest",
			fields: []fieldPin{{name: "key", number: 1, kind: protoreflect.StringKind}},
		},
		{
			name:   "GetResponse",
			fields: []fieldPin{{name: "record", number: 1, kind: protoreflect.MessageKind, message: "supkvapi.v1.Record"}},
		},
		{
			name: "SetRequest",
			fields: []fieldPin{
				{name: "key", number: 1, kind: protoreflect.StringKind},
				{name: "value", number: 2, kind: protoreflect.BytesKind},
				{name: "create_if_absent", number: 3, kind: protoreflect.BoolKind},
				{name: "wait_for_replication", number: 4, kind: protoreflect.BoolKind},
				{name: "wait_timeout_ms", number: 5, kind: protoreflect.Int64Kind},
			},
		},
		{
			name: "SetResponse",
			fields: []fieldPin{
				{name: "version", number: 1, kind: protoreflect.BytesKind},
				{name: "acked", number: 2, kind: protoreflect.BoolKind},
				{name: "acked_count", number: 3, kind: protoreflect.Uint32Kind},
				{name: "total_peers", number: 4, kind: protoreflect.Uint32Kind},
			},
		},
		{
			name: "DeleteRequest",
			fields: []fieldPin{
				{name: "key", number: 1, kind: protoreflect.StringKind},
				{name: "wait_for_replication", number: 2, kind: protoreflect.BoolKind},
				{name: "wait_timeout_ms", number: 3, kind: protoreflect.Int64Kind},
			},
		},
		{
			name: "DeleteResponse",
			fields: []fieldPin{
				{name: "version", number: 1, kind: protoreflect.BytesKind},
				{name: "acked", number: 2, kind: protoreflect.BoolKind},
				{name: "acked_count", number: 3, kind: protoreflect.Uint32Kind},
				{name: "total_peers", number: 4, kind: protoreflect.Uint32Kind},
			},
		},
		{
			name:   "WatchRequest",
			fields: []fieldPin{{name: "prefix", number: 1, kind: protoreflect.StringKind}},
		},
		{
			name:   "WatchResponse",
			fields: []fieldPin{{name: "record", number: 1, kind: protoreflect.MessageKind, message: "supkvapi.v1.Record"}},
		},
		{
			name:   "SnapshotCreateRequest",
			fields: []fieldPin{{name: "name", number: 1, kind: protoreflect.StringKind}},
		},
		{name: "SnapshotCreateResponse"},
		{
			name:   "SnapshotLoadRequest",
			fields: []fieldPin{{name: "name", number: 1, kind: protoreflect.StringKind}},
		},
		{name: "SnapshotLoadResponse"},
	}

	for _, pin := range pins {
		msg := file.Messages().ByName(pin.name)
		if msg == nil {
			t.Errorf("message %q is missing from the wire descriptor", pin.name)

			continue
		}

		assertFields(t, msg, pin.fields)
	}
}
