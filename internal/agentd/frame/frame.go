// Package frame is the wire format between the Markitect Host and the
// session helper markitect-agentd. One stdio connection carries several
// channels as length-prefixed frames: control messages, the provider CLI's
// own protocol bytes, its diagnostic output and Host tool calls.
package frame

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
)

// ProtocolVersion identifies this frame format and its control messages.
// The Host and agentd refuse each other when their versions differ.
const ProtocolVersion = 1

// MaxPayload bounds one frame. Larger provider output is split by the writer.
const MaxPayload = 1 << 20

// Channel names the stream a frame belongs to.
type Channel byte

const (
	// Control carries one JSON control message per frame in both directions.
	Control Channel = 'c'
	// ProviderIn carries bytes from the Host to the provider's standard input.
	ProviderIn Channel = 'i'
	// ProviderOut carries bytes from the provider's standard output to the Host.
	ProviderOut Channel = 'o'
	// ProviderErr carries bytes from the provider's standard error to the Host.
	ProviderErr Channel = 'e'
	// Tool carries one JSON ToolCall (agentd to Host) or ToolReply (Host to
	// agentd) per frame.
	Tool Channel = 't'
)

func (c Channel) valid() bool {
	switch c {
	case Control, ProviderIn, ProviderOut, ProviderErr, Tool:
		return true
	}
	return false
}

func (c Channel) String() string {
	switch c {
	case Control:
		return "control"
	case ProviderIn:
		return "provider-in"
	case ProviderOut:
		return "provider-out"
	case ProviderErr:
		return "provider-err"
	case Tool:
		return "tool"
	}
	return fmt.Sprintf("channel(%#x)", byte(c))
}

// Frame is one unit on the connection.
type Frame struct {
	Channel Channel
	Payload []byte
}

const headerSize = 5

// ErrTooLarge reports a frame whose declared payload exceeds MaxPayload.
var ErrTooLarge = errors.New("frame payload exceeds the maximum size")

// Write encodes f as one channel byte, a big-endian payload length and the
// payload. It writes the whole frame with a single Write call so that
// concurrent writers serialized by a mutex never interleave partial frames.
func Write(w io.Writer, f Frame) error {
	if !f.Channel.valid() {
		return fmt.Errorf("write frame: unknown %s", f.Channel)
	}
	if len(f.Payload) > MaxPayload {
		return fmt.Errorf("write %s frame: %w", f.Channel, ErrTooLarge)
	}
	buf := make([]byte, headerSize+len(f.Payload))
	buf[0] = byte(f.Channel)
	binary.BigEndian.PutUint32(buf[1:headerSize], uint32(len(f.Payload)))
	copy(buf[headerSize:], f.Payload)
	_, err := w.Write(buf)
	return err
}

// Read decodes the next frame. It returns io.EOF only at a clean frame
// boundary; a frame cut short returns io.ErrUnexpectedEOF.
func Read(r io.Reader) (Frame, error) {
	var header [headerSize]byte
	if _, err := io.ReadFull(r, header[:1]); err != nil {
		return Frame{}, err
	}
	if _, err := io.ReadFull(r, header[1:]); err != nil {
		return Frame{}, unexpected(err)
	}
	c := Channel(header[0])
	if !c.valid() {
		return Frame{}, fmt.Errorf("read frame: unknown %s", c)
	}
	n := binary.BigEndian.Uint32(header[1:])
	if n > MaxPayload {
		return Frame{}, fmt.Errorf("read %s frame of %d bytes: %w", c, n, ErrTooLarge)
	}
	payload := make([]byte, n)
	if _, err := io.ReadFull(r, payload); err != nil {
		return Frame{}, unexpected(err)
	}
	return Frame{Channel: c, Payload: payload}, nil
}

func unexpected(err error) error {
	if errors.Is(err, io.EOF) {
		return io.ErrUnexpectedEOF
	}
	return err
}
