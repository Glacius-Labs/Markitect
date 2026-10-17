package frame_test

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/agentd/frame"
)

func TestFramesRoundTripInOrder(t *testing.T) {
	var buf bytes.Buffer
	in := []frame.Frame{
		{Channel: frame.Control, Payload: []byte(`{"type":"hello"}`)},
		{Channel: frame.ProviderOut, Payload: []byte("line one\n")},
		{Channel: frame.ProviderErr, Payload: nil},
		{Channel: frame.Tool, Payload: bytes.Repeat([]byte{'x'}, frame.MaxPayload)},
	}
	for _, f := range in {
		if err := frame.Write(&buf, f); err != nil {
			t.Fatalf("write %s: %v", f.Channel, err)
		}
	}
	for i, want := range in {
		got, err := frame.Read(&buf)
		if err != nil {
			t.Fatalf("read frame %d: %v", i, err)
		}
		if got.Channel != want.Channel || !bytes.Equal(got.Payload, want.Payload) {
			t.Fatalf("frame %d = %s/%d bytes, want %s/%d bytes", i, got.Channel, len(got.Payload), want.Channel, len(want.Payload))
		}
	}
	if _, err := frame.Read(&buf); err != io.EOF {
		t.Fatalf("read after last frame = %v, want io.EOF", err)
	}
}

func TestReadRejectsDamagedInput(t *testing.T) {
	header := func(c byte, n uint32) []byte {
		b := []byte{c, 0, 0, 0, 0}
		binary.BigEndian.PutUint32(b[1:], n)
		return b
	}
	cases := []struct {
		name  string
		input []byte
		want  error
	}{
		{"cut header", []byte{'o', 0, 0}, io.ErrUnexpectedEOF},
		{"cut payload", append(header('o', 4), 'a', 'b'), io.ErrUnexpectedEOF},
		{"oversized", header('o', frame.MaxPayload+1), frame.ErrTooLarge},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := frame.Read(bytes.NewReader(tc.input)); !errors.Is(err, tc.want) {
				t.Fatalf("Read = %v, want %v", err, tc.want)
			}
		})
	}
	if _, err := frame.Read(bytes.NewReader(header('z', 0))); err == nil || !strings.Contains(err.Error(), "unknown") {
		t.Fatalf("unknown channel accepted: %v", err)
	}
}

func TestWriteRefusesInvalidFrames(t *testing.T) {
	if err := frame.Write(io.Discard, frame.Frame{Channel: 'z'}); err == nil {
		t.Fatal("unknown channel written")
	}
	big := frame.Frame{Channel: frame.ProviderOut, Payload: make([]byte, frame.MaxPayload+1)}
	if err := frame.Write(io.Discard, big); !errors.Is(err, frame.ErrTooLarge) {
		t.Fatalf("oversized write = %v", err)
	}
}

func TestControlMessagesNeedExactlyTheirBody(t *testing.T) {
	start := frame.ControlMessage{Type: frame.TypeStart, Start: &frame.Start{Argv: []string{"codex", "app-server"}, Dir: "/work"}}
	payload, err := frame.EncodeControl(start)
	if err != nil {
		t.Fatal(err)
	}
	got, err := frame.DecodeControl(payload)
	if err != nil || got.Start.Argv[1] != "app-server" {
		t.Fatalf("decode = %+v, %v", got, err)
	}
	bad := []frame.ControlMessage{
		{Type: frame.TypeStart, Start: &frame.Start{Dir: "/work"}},
		{Type: frame.TypeStart},
		{Type: frame.TypeHello, Hello: &frame.Hello{}, Exited: &frame.Exited{}},
		{Type: frame.TypeStop, Stop: &frame.Stop{GraceMillis: -1}},
		{Type: "restart", Failure: &frame.Failure{}},
	}
	for _, c := range bad {
		if _, err := frame.EncodeControl(c); err == nil {
			t.Fatalf("invalid control message encoded: %+v", c)
		}
	}
	for _, raw := range []string{
		`{"type":"hello","hello":{"protocol":1,"version":"x"},"extra":1}`,
		`{"type":"hello","hello":{"protocol":1,"version":"x"}} {}`,
		`{"type":"stop"}`,
	} {
		if _, err := frame.DecodeControl([]byte(raw)); err == nil {
			t.Fatalf("invalid control payload decoded: %s", raw)
		}
	}
}

func TestToolRepliesCarryExactlyResultOrError(t *testing.T) {
	ok := frame.ToolReply{ID: 7, Result: json.RawMessage(`{"content":[{"type":"text","text":"D-1"}]}`)}
	payload, err := frame.EncodeToolReply(ok)
	if err != nil {
		t.Fatal(err)
	}
	if got, err := frame.DecodeToolReply(payload); err != nil || got.ID != 7 {
		t.Fatalf("decode = %+v, %v", got, err)
	}
	for _, r := range []frame.ToolReply{
		{ID: 1},
		{ID: 1, Result: json.RawMessage(`{}`), Error: &frame.ToolError{Code: 1, Message: "no"}},
		{ID: 1, Result: json.RawMessage(`{`)},
	} {
		if _, err := frame.EncodeToolReply(r); err == nil {
			t.Fatalf("invalid reply encoded: %+v", r)
		}
	}
	if _, err := frame.EncodeToolCall(frame.ToolCall{ID: 1, Name: "submit_work", Arguments: json.RawMessage(`{`)}); err == nil {
		t.Fatal("tool call with invalid arguments encoded")
	}
}
