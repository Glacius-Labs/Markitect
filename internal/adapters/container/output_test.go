package container

import (
	"errors"
	"io"
	"testing"
)

func TestOutputBufferNeverWaitsForTheReader(t *testing.T) {
	b := newOutputBuffer()
	if _, err := b.Write([]byte("first ")); err != nil {
		t.Fatal(err)
	}
	if _, err := b.Write([]byte("second")); err != nil {
		t.Fatal(err)
	}
	b.close()
	got, err := io.ReadAll(b)
	if err != nil || string(got) != "first second" {
		t.Fatalf("read %q, %v", got, err)
	}
	if _, err := b.Write([]byte("late")); !errors.Is(err, io.ErrClosedPipe) {
		t.Fatalf("write after close = %v", err)
	}
}

func TestOutputBufferBoundsAnAbsentReader(t *testing.T) {
	b := newOutputBuffer()
	if _, err := b.Write(make([]byte, maxBufferedOutput)); err != nil {
		t.Fatal(err)
	}
	if _, err := b.Write([]byte{1}); !errors.Is(err, errOutputOverflow) {
		t.Fatalf("overflow = %v", err)
	}
	if _, err := b.Read(make([]byte, maxBufferedOutput)); err != nil {
		t.Fatalf("draining after the overflow = %v", err)
	}
	if _, err := b.Write([]byte{2}); !errors.Is(err, errOutputOverflow) {
		t.Fatalf("a write after the overflow was accepted: %v", err)
	}
	if _, err := b.Read(make([]byte, 1)); !errors.Is(err, errOutputOverflow) {
		t.Fatalf("read after draining = %v, want the overflow", err)
	}
	b.detach()
	if n, err := b.Write([]byte("discarded")); err != nil || n != 9 {
		t.Fatalf("write after detach = %d, %v", n, err)
	}
	if n, err := b.Read(make([]byte, 4)); n != 0 || err != io.EOF {
		t.Fatalf("read after detach = %d, %v", n, err)
	}
}
