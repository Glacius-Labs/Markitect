package container

import (
	"archive/tar"
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func imageArchive(t *testing.T, h tar.Header, body string, extra bool) []byte {
	t.Helper()
	var out bytes.Buffer
	tw := tar.NewWriter(&out)
	if err := tw.WriteHeader(&h); err != nil {
		t.Fatal(err)
	}
	if _, err := io.WriteString(tw, body); err != nil {
		t.Fatal(err)
	}
	if extra {
		if err := tw.WriteHeader(&tar.Header{Name: "another", Typeflag: tar.TypeReg, Mode: 0755}); err != nil {
			t.Fatal(err)
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	return out.Bytes()
}

func TestObserveImageFileStreamsBeyondMetadataLimits(t *testing.T) {
	body := strings.Repeat("whole executable\x00", (5<<20)/17+1)
	archive := imageArchive(t, tar.Header{Name: "check", Typeflag: tar.TypeReg, Mode: 0755, Size: int64(len(body))}, body, false)
	e, calls := fakeCommandEngine(t, engineStep{Out: commandID}, engineStep{Out: string(archive)}, engineStep{})
	got, err := e.ObserveImageFile(context.Background(), commandImage, "/opt/check", int64(len(body)))
	if err != nil || got.Size != int64(len(body)) || got.SHA256 != sha256.Sum256([]byte(body)) {
		t.Fatalf("observation=%+v err=%v", got, err)
	}
	want := [][]string{{"create", "--entrypoint", "/markitect-never-started", commandImage}, {"cp", commandID + ":/opt/check", "-"}, {"rm", "--force", commandID}}
	if !reflect.DeepEqual(calls(), want) {
		t.Fatalf("engine calls=%q", calls())
	}
}

func TestObserveImageFileRefusesInvalidInputsBeforeEngine(t *testing.T) {
	e := &Engine{path: "must-not-run"}
	for _, file := range []string{"", "/", "check", "/opt/../check", "/opt//check", "/opt/check/", "/opt/check\x00", "/opt\\check", "/opt/check\n"} {
		if _, err := e.ObserveImageFile(context.Background(), commandImage, file, 100); err == nil {
			t.Fatalf("accepted file %q", file)
		}
	}
	for _, limit := range []int64{-1, 0, MaxImageFileObservationBytes + 1} {
		if _, err := e.ObserveImageFile(context.Background(), commandImage, "/check", limit); err == nil {
			t.Fatalf("accepted limit %d", limit)
		}
	}
	if _, err := e.ObserveImageFile(context.Background(), "image:tag", "/check", 100); err == nil {
		t.Fatal("accepted mutable image")
	}
}

func TestObserveImageFileRejectsIncompleteObservationsAndCleans(t *testing.T) {
	regular := tar.Header{Name: "check", Typeflag: tar.TypeReg, Mode: 0755, Size: 3}
	valid := imageArchive(t, regular, "abc", false)
	wrongName := regular
	wrongName.Name = "wrong"
	for _, tc := range []struct {
		name          string
		archive       []byte
		limit         int64
		copy, cleanup engineStep
	}{
		{name: "oversize", archive: valid, limit: 2},
		{name: "symlink", archive: imageArchive(t, tar.Header{Name: "check", Typeflag: tar.TypeSymlink, Linkname: "target"}, "", false)},
		{name: "directory", archive: imageArchive(t, tar.Header{Name: "check", Typeflag: tar.TypeDir}, "", false)},
		{name: "wrong name", archive: imageArchive(t, wrongName, "abc", false)},
		{name: "extra entry", archive: imageArchive(t, regular, "abc", true)},
		{name: "truncated payload", archive: valid[:514]},
		{name: "missing terminator", archive: valid[:1024]},
		{name: "single block terminator", archive: valid[:1536]},
		{name: "malformed", archive: []byte("not tar")},
		{name: "trailing garbage", archive: append(bytes.Clone(valid), 'x')},
		{name: "trailing partial block", archive: append(bytes.Clone(valid), 0)},
		{name: "unbounded trailer", archive: append(bytes.Clone(valid), make([]byte, (1<<20)+512)...)},
		{name: "copy failed after complete archive", archive: valid, copy: engineStep{Code: 1, Err: "copy refused"}},
		{name: "cleanup failed", archive: valid, cleanup: engineStep{Code: 1, Err: "remove refused"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			limit := tc.limit
			if limit == 0 {
				limit = 100
			}
			tc.copy.Out = string(tc.archive)
			e, calls := fakeCommandEngine(t, engineStep{Out: commandID}, tc.copy, tc.cleanup)
			got, err := e.ObserveImageFile(context.Background(), commandImage, "/check", limit)
			if err == nil || got != (ImageFileObservation{}) {
				t.Fatalf("observation=%+v err=%v", got, err)
			}
			all := calls()
			if len(all) != 3 || !reflect.DeepEqual(all[2], []string{"rm", "--force", commandID}) {
				t.Fatalf("cleanup calls=%q", all)
			}
		})
	}
}

func TestObserveImageFileCancellationCleansWithoutCertifyingBytes(t *testing.T) {
	e, calls := fakeCommandEngine(t, engineStep{Out: commandID}, engineStep{SleepMillis: 10000}, engineStep{})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	type result struct {
		observation ImageFileObservation
		err         error
	}
	done := make(chan result, 1)
	go func() {
		observation, err := e.ObserveImageFile(ctx, commandImage, "/check", 100)
		done <- result{observation, err}
	}()
	deadline := time.NewTimer(10 * time.Second)
	defer deadline.Stop()
	tick := time.NewTicker(5 * time.Millisecond)
	defer tick.Stop()
	for {
		select {
		case <-deadline.C:
			t.Fatal("observation did not reconcile cancellation")
		case got := <-done:
			if !errors.Is(got.err, context.Canceled) || got.observation != (ImageFileObservation{}) || len(calls()) != 3 {
				t.Fatalf("result=%+v calls=%q", got, calls())
			}
			return
		case <-tick.C:
			index, _ := os.ReadFile(filepath.Join(os.Getenv("MARKITECT_FAKE_COMMAND_ENGINE"), "index"))
			if string(index) == "2" {
				cancel()
			}
		}
	}
}

type failedImageReader struct{ err error }

func (r failedImageReader) Read([]byte) (int, error) { return 0, r.err }

func TestImageArchivePreservesReadFailure(t *testing.T) {
	archive := imageArchive(t, tar.Header{Name: "check", Typeflag: tar.TypeReg, Size: 3}, "abc", false)
	failure := errors.New("read failed")
	for _, prefix := range [][]byte{archive[:514], archive} {
		got, err := observeImageArchive(io.MultiReader(bytes.NewReader(prefix), failedImageReader{failure}), "check", 100)
		if !errors.Is(err, failure) || got != (ImageFileObservation{}) {
			t.Fatalf("observation=%+v err=%v", got, err)
		}
	}
}
