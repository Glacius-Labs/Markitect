package container

import (
	"archive/tar"
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"sync"
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
	e, calls := fakeCommandEngine(t, engineStep{Out: commandID}, engineStep{Out: imagePathHeader(t, "/opt", tar.TypeDir, "")}, engineStep{Out: string(archive)}, engineStep{})
	got, err := e.ObserveImageFile(context.Background(), commandImage, "/opt/check", int64(len(body)))
	if err != nil || got.Size != int64(len(body)) || got.SHA256 != sha256.Sum256([]byte(body)) {
		t.Fatalf("observation=%+v err=%v", got, err)
	}
	all := calls()
	if len(all) != 4 || len(all[0]) != 6 || all[0][1] != "--cidfile" || !filepath.IsAbs(all[0][2]) {
		t.Fatalf("engine calls=%q", all)
	}
	all[0][2] = "PRIVATE_CIDFILE"
	want := [][]string{{"create", "--cidfile", "PRIVATE_CIDFILE", "--entrypoint", "/markitect-never-started", commandImage}, {"cp", commandID + ":/opt", "-"}, {"cp", commandID + ":/opt/check", "-"}, {"rm", "--force", "--volumes", commandID}}
	if !reflect.DeepEqual(all, want) {
		t.Fatalf("engine calls=%q", all)
	}
}

func imagePathHeader(t *testing.T, file string, kind byte, link string) string {
	t.Helper()
	name := path.Base(file)
	if kind == tar.TypeDir {
		name += "/"
	}
	archive := imageArchive(t, tar.Header{Name: name, Typeflag: kind, Linkname: link, Mode: 0755}, "", false)
	return string(archive[:512])
}

func TestObserveImageFileAllowsSlowFirstPathObservation(t *testing.T) {
	// The first path in a cold image can exceed the former 15-second bound.
	// Exercise the real deadline without changing the production timeout.
	archive := imageArchive(t, tar.Header{Name: "check", Typeflag: tar.TypeReg, Size: 3}, "abc", false)
	e, calls := fakeCommandEngine(t, engineStep{Out: commandID},
		engineStep{Out: imagePathHeader(t, "/work", tar.TypeDir, ""), SleepMillis: 16000},
		engineStep{Out: string(archive)}, engineStep{})
	got, err := e.ObserveImageFileWithRequiredPaths(context.Background(), commandImage, "/check", 3, []string{"/work"}, nil)
	if err != nil || got.SHA256 != sha256.Sum256([]byte("abc")) || got.Size != 3 {
		t.Fatalf("observation=%+v err=%v", got, err)
	}
	all := calls()
	wantTail := [][]string{{"cp", commandID + ":/work", "-"}, {"cp", commandID + ":/check", "-"}, {"rm", "--force", "--volumes", commandID}}
	if len(all) != 4 || !reflect.DeepEqual(all[1:], wantTail) {
		t.Fatalf("engine calls=%q", all)
	}
}

func TestObserveImageFileColdPathHonorsParentDeadline(t *testing.T) {
	e, calls := fakeCommandEngine(t, engineStep{Out: commandID}, engineStep{SleepMillis: 16000}, engineStep{})
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	type result struct {
		observation ImageFileObservation
		err         error
	}
	done := make(chan result, 1)
	go func() {
		got, err := e.ObserveImageFileWithRequiredPaths(ctx, commandImage, "/check", 3, []string{"/work"}, nil)
		done <- result{got, err}
	}()
	select {
	case got := <-done:
		all := calls()
		wantTail := [][]string{{"cp", commandID + ":/work", "-"}, {"rm", "--force", "--volumes", commandID}}
		if !errors.Is(got.err, context.DeadlineExceeded) || got.observation != (ImageFileObservation{}) || len(all) != 3 || !reflect.DeepEqual(all[1:], wantTail) {
			t.Fatalf("observation=%+v err=%v calls=%q", got.observation, got.err, all)
		}
	case <-time.After(8 * time.Second):
		t.Fatal("cold path observation did not honor the shorter parent deadline and clean its container")
	}
}

func TestObserveImageFileRequiredPathsAreOrderedUniqueAndBounded(t *testing.T) {
	// /opt and /opt/tools occur in both the executable and required-directory
	// chain; every resource is observed once, with parents before children.
	paths := []string{"/etc", "/home", "/opt", "/work", "/etc/hosts", "/home/agent", "/opt/tools"}
	steps := []engineStep{{Out: commandID}}
	for _, p := range paths {
		kind := byte(tar.TypeDir)
		if p == "/etc/hosts" {
			kind = tar.TypeReg
		}
		header := imagePathHeader(t, p, kind, "")
		if p == "/work" {
			header += strings.Repeat("unneeded directory contents", 200000)
		}
		steps = append(steps, engineStep{Out: header})
	}
	steps = append(steps, engineStep{Out: string(imageArchive(t, tar.Header{Name: "check", Typeflag: tar.TypeReg, Size: 3}, "abc", false))}, engineStep{})
	e, calls := fakeCommandEngine(t, steps...)
	got, err := e.ObserveImageFileWithRequiredPaths(context.Background(), commandImage, "/opt/tools/check", 3, []string{"/work", "/home/agent", "/opt/tools", "/work"}, []string{"/etc/hosts", "/etc/hosts"})
	if err != nil || got.SHA256 != sha256.Sum256([]byte("abc")) || got.Size != 3 {
		t.Fatalf("observation=%+v err=%v", got, err)
	}
	all := calls()
	if len(all) != len(paths)+3 {
		t.Fatalf("calls=%q", all)
	}
	for i, p := range append(paths, "/opt/tools/check") {
		if !reflect.DeepEqual(all[i+1], []string{"cp", commandID + ":" + p, "-"}) {
			t.Fatalf("call %d=%q", i+1, all[i+1])
		}
	}
	if !reflect.DeepEqual(all[len(all)-1], []string{"rm", "--force", "--volumes", commandID}) {
		t.Fatalf("cleanup=%q", all)
	}
}

func TestObserveImageFileRefusesAncestorAndMountAliases(t *testing.T) {
	for _, tc := range []struct {
		name, file, alias, target string
		dirs, files               []string
		prefix                    []engineStep
	}{
		{name: "executable ancestor", file: "/opt/tools/check", alias: "/opt/tools", target: "/work", prefix: []engineStep{{Out: imagePathHeader(t, "/opt", tar.TypeDir, "")}}},
		{name: "directory mount target", file: "/opt/tools/check", alias: "/work", target: "/opt/tools", dirs: []string{"/work"}, prefix: []engineStep{{Out: imagePathHeader(t, "/opt", tar.TypeDir, "")}}},
		{name: "daemon file mount target", file: "/check", alias: "/etc/hosts", target: "/check", files: []string{"/etc/hosts"}, prefix: []engineStep{{Out: imagePathHeader(t, "/etc", tar.TypeDir, "")}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			steps := append([]engineStep{{Out: commandID}}, tc.prefix...)
			steps = append(steps, engineStep{Out: imagePathHeader(t, tc.alias, tar.TypeSymlink, tc.target)}, engineStep{})
			e, calls := fakeCommandEngine(t, steps...)
			got, err := e.ObserveImageFileWithRequiredPaths(context.Background(), commandImage, tc.file, 100, tc.dirs, tc.files)
			if err == nil || got != (ImageFileObservation{}) {
				t.Fatalf("observation=%+v err=%v", got, err)
			}
			all := calls()
			if len(all) != len(steps) || !reflect.DeepEqual(all[len(all)-1], []string{"rm", "--force", "--volumes", commandID}) {
				t.Fatalf("calls=%q", all)
			}
		})
	}
}

func TestObserveImageFilePathHeaderRefusalsAndCleanup(t *testing.T) {
	metadata := imageArchive(t, tar.Header{Name: "work", Typeflag: tar.TypeReg, PAXRecords: map[string]string{"comment": strings.Repeat("x", 4096)}, Format: tar.FormatPAX}, "", false)
	metadata = metadata[:len(metadata)-512-1024]
	for _, tc := range []struct {
		name   string
		header engineStep
	}{
		{"missing", engineStep{Code: 1, Err: "missing"}},
		{"wrong name", engineStep{Out: imagePathHeader(t, "/wrong", tar.TypeDir, "")}},
		{"wrong type", engineStep{Out: imagePathHeader(t, "/work", tar.TypeReg, "")}},
		{"partial header", engineStep{Out: imagePathHeader(t, "/work", tar.TypeDir, "")[:511]}},
		{"metadata budget", engineStep{Out: string(bytes.Repeat(metadata, 256))}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			e, calls := fakeCommandEngine(t, engineStep{Out: commandID}, tc.header, engineStep{})
			got, err := e.ObserveImageFileWithRequiredPaths(context.Background(), commandImage, "/check", 100, []string{"/work"}, nil)
			if err == nil || got != (ImageFileObservation{}) || len(calls()) != 3 {
				t.Fatalf("observation=%+v err=%v calls=%q", got, err, calls())
			}
		})
	}
}

func TestObserveImageFileRequiredPathCleanupFailureDiscardsHash(t *testing.T) {
	archive := imageArchive(t, tar.Header{Name: "check", Typeflag: tar.TypeReg, Size: 3}, "abc", false)
	e, calls := fakeCommandEngine(t, engineStep{Out: commandID}, engineStep{Out: imagePathHeader(t, "/work", tar.TypeDir, "")}, engineStep{Out: string(archive)}, engineStep{Code: 1, Err: "cleanup denied"})
	got, err := e.ObserveImageFileWithRequiredPaths(context.Background(), commandImage, "/check", 3, []string{"/work"}, nil)
	if err == nil || !strings.Contains(err.Error(), "cleanup") || got != (ImageFileObservation{}) || len(calls()) != 4 {
		t.Fatalf("observation=%+v err=%v calls=%q", got, err, calls())
	}
}

func TestObserveImageFileRequiredPathsRefuseInvalidInputsBeforeEngine(t *testing.T) {
	e := &Engine{path: "must-not-run"}
	for _, dirs := range [][]string{{"/tmp/../work"}, {"/"}, {"/check"}, make([]string, maxImagePathObservations+1), {"/" + strings.Repeat("part/", maxImagePathObservations) + "dir"}} {
		if _, err := e.ObserveImageFileWithRequiredPaths(context.Background(), commandImage, "/check", 100, dirs, nil); err == nil {
			t.Fatalf("accepted dirs=%q", dirs)
		}
	}
	if _, err := e.ObserveImageFileWithRequiredPaths(context.Background(), commandImage, "/check", 100, []string{"/work"}, []string{"/work"}); err == nil {
		t.Fatal("accepted conflicting path types")
	}
}

func TestImageHeaderStopRejectsUnrelatedWaitFailure(t *testing.T) {
	e, _ := fakeCommandEngine(t, engineStep{Code: 7})
	waitErr := exec.Command(e.path, "controlled-exit").Run()
	if waitErr == nil || confirmedImageHeaderStop(nil, waitErr) || confirmedImageHeaderStop(os.ErrProcessDone, waitErr) || confirmedImageHeaderStop(errors.New("kill failed"), waitErr) || !confirmedImageHeaderStop(errors.New("already exited"), nil) {
		t.Fatalf("accepted unconfirmed stop or unrelated exit: %v", waitErr)
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
			if len(all) != 3 || !reflect.DeepEqual(all[2], []string{"rm", "--force", "--volumes", commandID}) {
				t.Fatalf("cleanup calls=%q", all)
			}
		})
	}
}

func TestObserveImageFileCleansObservedCIDOnCreateFailure(t *testing.T) {
	e, calls := fakeCommandEngine(t, engineStep{Out: commandID, Code: 1, Err: "create transport failed"}, engineStep{})
	got, err := e.ObserveImageFile(context.Background(), commandImage, "/check", 100)
	if err == nil || got != (ImageFileObservation{}) {
		t.Fatalf("observation=%+v err=%v", got, err)
	}
	all := calls()
	if len(all) != 2 || !reflect.DeepEqual(all[1], []string{"rm", "--force", "--volumes", commandID}) {
		t.Fatalf("calls=%q", all)
	}
}

func TestObserveImageFileCleansObservedCIDOnCreateCancellation(t *testing.T) {
	e, calls := fakeCommandEngine(t, engineStep{Out: commandID, SleepMillis: 10000}, engineStep{})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() { _, err := e.ObserveImageFile(ctx, commandImage, "/check", 100); done <- err }()
	deadline := time.Now().Add(5 * time.Second)
	observed := false
	for time.Now().Before(deadline) {
		if raw, err := os.ReadFile(filepath.Join(os.Getenv("MARKITECT_FAKE_COMMAND_ENGINE"), "create-ready")); err == nil && string(raw) == commandID {
			observed = true
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	if !observed {
		t.Fatal("fake create did not report CID before cancellation")
	}
	cancel()
	select {
	case err := <-done:
		all := calls()
		if !errors.Is(err, context.Canceled) || len(all) != 2 || !reflect.DeepEqual(all[1], []string{"rm", "--force", "--volumes", commandID}) {
			t.Fatalf("err=%v calls=%q", err, all)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("create cancellation did not clean observed CID")
	}
}

func TestObserveImageFileRefusesInvalidCreatedCID(t *testing.T) {
	e, calls := fakeCommandEngine(t, engineStep{Out: "foreign-name"})
	got, err := e.ObserveImageFile(context.Background(), commandImage, "/check", 100)
	if err == nil || got != (ImageFileObservation{}) || len(calls()) != 1 {
		t.Fatalf("observation=%+v err=%v calls=%q", got, err, calls())
	}
}

func TestObserveImageFileCancelsInheritedStdout(t *testing.T) {
	e, calls := fakeCommandEngine(t, engineStep{Out: commandID}, engineStep{HoldStdoutMillis: 10000}, engineStep{})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() { _, err := e.ObserveImageFile(ctx, commandImage, "/check", 100); done <- err }()
	pidPath := filepath.Join(os.Getenv("MARKITECT_FAKE_COMMAND_ENGINE"), "stdout-holder")
	deadline := time.Now().Add(5 * time.Second)
	var holder *os.Process
	for time.Now().Before(deadline) {
		if raw, err := os.ReadFile(pidPath); err == nil {
			pid, _ := strconv.Atoi(string(raw))
			holder, _ = os.FindProcess(pid)
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	if holder == nil {
		t.Fatal("fake engine did not start stdout holder")
	}
	defer holder.Kill()
	// The cp CLI exits with no inherited stderr handle; only its child keeps
	// stdout open. Cancellation must unblock the reader before cmd.Wait.
	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) || len(calls()) != 3 {
			t.Fatalf("err=%v calls=%q", err, calls())
		}
	case <-time.After(2 * time.Second):
		t.Fatal("inherited stdout blocked cancellation")
	}
}

type countedImageReader struct {
	io.Reader
	n int
}

func (r *countedImageReader) Read(p []byte) (int, error) {
	n, err := r.Reader.Read(p)
	r.n += n
	return n, err
}

func TestImageArchiveBoundsMetadataBeforeFileBudget(t *testing.T) {
	withMetadata := imageArchive(t, tar.Header{Name: "check", Typeflag: tar.TypeReg, PAXRecords: map[string]string{"comment": strings.Repeat("x", 4096)}, Format: tar.FormatPAX}, "", false)
	// Remove the empty regular header and terminator, leaving a PAX metadata
	// record generated by archive/tar. Repeat it before one actual file.
	metadata := withMetadata[:len(withMetadata)-512-1024]
	archive := append(bytes.Repeat(metadata, 256), imageArchive(t, tar.Header{Name: "check", Typeflag: tar.TypeReg}, "", false)...)
	reader := &countedImageReader{Reader: bytes.NewReader(archive)}
	got, err := observeImageArchive(reader, "check", MaxImageFileObservationBytes)
	if err == nil || got != (ImageFileObservation{}) || reader.n > 1<<20 {
		t.Fatalf("observation=%+v err=%v bytes=%d", got, err, reader.n)
	}
}

func TestObserveImageFileCancellationCleansWithoutCertifyingBytes(t *testing.T) {
	for _, requiredDirectory := range []string{"", "/work"} {
		t.Run("requiredDirectory="+requiredDirectory, func(t *testing.T) {
			e, calls := fakeCommandEngine(t, engineStep{Out: commandID}, engineStep{SleepMillis: 10000}, engineStep{})
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			type result struct {
				observation ImageFileObservation
				err         error
			}
			done := make(chan result, 1)
			go func() {
				var observation ImageFileObservation
				var err error
				if requiredDirectory == "" {
					observation, err = e.ObserveImageFile(ctx, commandImage, "/check", 100)
				} else {
					observation, err = e.ObserveImageFileWithRequiredPaths(ctx, commandImage, "/check", 100, []string{requiredDirectory}, nil)
				}
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
					all := calls()
					copied := requiredDirectory
					if copied == "" {
						copied = "/check"
					}
					if !errors.Is(got.err, context.Canceled) || got.observation != (ImageFileObservation{}) || len(all) != 3 ||
						!reflect.DeepEqual(all[1], []string{"cp", commandID + ":" + copied, "-"}) || !reflect.DeepEqual(all[2], []string{"rm", "--force", "--volumes", commandID}) {
						t.Fatalf("result=%+v calls=%q", got, all)
					}
					return
				case <-tick.C:
					index, _ := os.ReadFile(filepath.Join(os.Getenv("MARKITECT_FAKE_COMMAND_ENGINE"), "index"))
					if string(index) == "2" {
						cancel()
					}
				}
			}
		})
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

func observationSteps(t *testing.T, dir string, kind byte, body string) []engineStep {
	t.Helper()
	return []engineStep{{Out: commandID}, {Out: imagePathHeader(t, dir, kind, "")},
		{Out: string(imageArchive(t, tar.Header{Name: "check", Typeflag: tar.TypeReg, Size: int64(len(body))}, body, false))}, {}}
}

func removeCall() []string { return []string{"rm", "--force", "--volumes", commandID} }

func TestObserveImageFileMemoizesIdenticalObservations(t *testing.T) {
	e, calls := fakeCommandEngine(t, observationSteps(t, "/work", tar.TypeDir, "abc")...)
	want := ImageFileObservation{SHA256: sha256.Sum256([]byte("abc")), Size: 3}
	// Order and repetition of the required paths do not change the request.
	for _, dirs := range [][]string{{"/work"}, {"/work", "/work"}} {
		got, err := e.ObserveImageFileWithRequiredPaths(context.Background(), commandImage, "/check", 3, dirs, nil)
		if err != nil || got != want {
			t.Fatalf("observation=%+v err=%v", got, err)
		}
	}
	all := calls()
	if len(all) != 4 || all[0][0] != "create" || !reflect.DeepEqual(all[1], []string{"cp", commandID + ":/work", "-"}) ||
		!reflect.DeepEqual(all[2], []string{"cp", commandID + ":/check", "-"}) || !reflect.DeepEqual(all[3], removeCall()) {
		t.Fatalf("two identical observations must run one engine sequence: %q", all)
	}
}

func TestObserveImageFileMemoIsKeyedByTheExactRequest(t *testing.T) {
	steps := observationSteps(t, "/work", tar.TypeDir, "abc")
	steps = append(steps, observationSteps(t, "/work", tar.TypeDir, "abc")...)
	steps = append(steps, observationSteps(t, "/work", tar.TypeDir, "abc")...)
	steps = append(steps, observationSteps(t, "/work", tar.TypeReg, "abc")...)
	e, calls := fakeCommandEngine(t, steps...)
	ctx := context.Background()
	if _, err := e.ObserveImageFileWithRequiredPaths(ctx, commandImage, "/check", 3, []string{"/work"}, nil); err != nil {
		t.Fatal(err)
	}
	// Another size bound, another image and another path set each observe again.
	if _, err := e.ObserveImageFileWithRequiredPaths(ctx, commandImage, "/check", 4, []string{"/work"}, nil); err != nil {
		t.Fatal(err)
	}
	other := "sha256:" + strings.Repeat("c", 64)
	if _, err := e.ObserveImageFileWithRequiredPaths(ctx, other, "/check", 3, []string{"/work"}, nil); err != nil {
		t.Fatal(err)
	}
	// The same path as a file instead of a directory is a different requirement
	// and must not be answered from the directory observation.
	if _, err := e.ObserveImageFileWithRequiredPaths(ctx, commandImage, "/check", 3, nil, []string{"/work"}); err != nil {
		t.Fatal(err)
	}
	if got := len(calls()); got != 16 {
		t.Fatalf("engine calls=%d, want one full sequence per distinct request", got)
	}
}

func TestObserveImageFileDoesNotRememberFailures(t *testing.T) {
	steps := []engineStep{{Out: commandID}, {Code: 1, Err: "transient"}, {}}
	steps = append(steps, observationSteps(t, "/work", tar.TypeDir, "abc")...)
	e, calls := fakeCommandEngine(t, steps...)
	if got, err := e.ObserveImageFileWithRequiredPaths(context.Background(), commandImage, "/check", 3, []string{"/work"}, nil); err == nil || got != (ImageFileObservation{}) {
		t.Fatalf("observation=%+v err=%v", got, err)
	}
	got, err := e.ObserveImageFileWithRequiredPaths(context.Background(), commandImage, "/check", 3, []string{"/work"}, nil)
	if err != nil || got.SHA256 != sha256.Sum256([]byte("abc")) || len(calls()) != 7 {
		t.Fatalf("a failed observation must be retried: observation=%+v err=%v calls=%q", got, err, calls())
	}
	// And the success is remembered.
	if _, err := e.ObserveImageFileWithRequiredPaths(context.Background(), commandImage, "/check", 3, []string{"/work"}, nil); err != nil || len(calls()) != 7 {
		t.Fatalf("err=%v calls=%q", err, calls())
	}
}

func TestObserveImageFileDoesNotRememberCleanupFailure(t *testing.T) {
	steps := observationSteps(t, "/work", tar.TypeDir, "abc")
	steps[3] = engineStep{Code: 1, Err: "remove refused"}
	steps = append(steps, observationSteps(t, "/work", tar.TypeDir, "abc")...)
	e, calls := fakeCommandEngine(t, steps...)
	if got, err := e.ObserveImageFileWithRequiredPaths(context.Background(), commandImage, "/check", 3, []string{"/work"}, nil); err == nil || got != (ImageFileObservation{}) {
		t.Fatalf("observation=%+v err=%v", got, err)
	}
	if got, err := e.ObserveImageFileWithRequiredPaths(context.Background(), commandImage, "/check", 3, []string{"/work"}, nil); err != nil || got.Size != 3 || len(calls()) != 8 {
		t.Fatalf("observation=%+v err=%v calls=%d", got, err, len(calls()))
	}
}

func TestObserveImageFileConcurrentIdenticalObservationsShareOne(t *testing.T) {
	steps := observationSteps(t, "/work", tar.TypeDir, "abc")
	steps[0].SleepMillis = 300
	e, calls := fakeCommandEngine(t, steps...)
	var wg sync.WaitGroup
	results := make([]error, 4)
	for i := range results {
		wg.Add(1)
		go func() {
			defer wg.Done()
			got, err := e.ObserveImageFileWithRequiredPaths(context.Background(), commandImage, "/check", 3, []string{"/work"}, nil)
			if err == nil && got.Size != 3 {
				err = errors.New("wrong observation")
			}
			results[i] = err
		}()
	}
	wg.Wait()
	for _, err := range results {
		if err != nil {
			t.Fatal(err)
		}
	}
	if got := len(calls()); got != 4 {
		t.Fatalf("engine calls=%d, want one sequence for four identical concurrent observations", got)
	}
}

func TestImageObservationMemoWaiterDoesNotInheritLeaderFailure(t *testing.T) {
	var memo imageObservationMemo
	release, started := make(chan struct{}), make(chan struct{})
	runs := 0
	var mu sync.Mutex
	run := func(err error) func() (ImageFileObservation, error) {
		return func() (ImageFileObservation, error) {
			mu.Lock()
			runs++
			mu.Unlock()
			if err != nil {
				close(started)
				<-release
				return ImageFileObservation{}, err
			}
			return ImageFileObservation{Size: 7}, nil
		}
	}
	leader := make(chan error, 1)
	go func() {
		_, err := memo.observe(context.Background(), "key", run(context.Canceled))
		leader <- err
	}()
	<-started
	waiter := make(chan ImageFileObservation, 1)
	go func() {
		got, _ := memo.observe(context.Background(), "key", run(nil))
		waiter <- got
	}()
	// A waiter whose own context ends stops waiting without a result.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if got, err := memo.observe(ctx, "key", run(nil)); !errors.Is(err, context.Canceled) || got != (ImageFileObservation{}) {
		t.Fatalf("observation=%+v err=%v", got, err)
	}
	close(release)
	if err := <-leader; !errors.Is(err, context.Canceled) {
		t.Fatalf("leader err=%v", err)
	}
	if got := <-waiter; got.Size != 7 {
		t.Fatalf("waiter must run its own observation after the leader failed: %+v", got)
	}
	if got, err := memo.observe(context.Background(), "key", run(nil)); err != nil || got.Size != 7 {
		t.Fatalf("observation=%+v err=%v", got, err)
	}
	mu.Lock()
	defer mu.Unlock()
	if runs != 2 {
		t.Fatalf("runs=%d, want the failed leader and one successful waiter", runs)
	}
}

func TestObserveImageFileDoesNotReportItsOwnKillAsEngineFailure(t *testing.T) {
	// The engine still has more to send when the observation refuses the first
	// header and stops the copy on purpose.
	wrong := imageArchive(t, tar.Header{Name: "check", Typeflag: tar.TypeSymlink, Linkname: "elsewhere"}, "", false)
	e, calls := fakeCommandEngine(t, engineStep{Out: commandID}, engineStep{Out: string(wrong), LingerMillis: 20000}, engineStep{})
	start := time.Now()
	got, err := e.ObserveImageFile(context.Background(), commandImage, "/check", 100)
	if err == nil || got != (ImageFileObservation{}) || !strings.Contains(err.Error(), "needs exactly regular file") {
		t.Fatalf("observation=%+v err=%v", got, err)
	}
	if strings.Contains(err.Error(), "killed") || strings.Contains(err.Error(), "container engine cp") {
		t.Fatalf("our own stop leaked into the error: %v", err)
	}
	if time.Since(start) > 10*time.Second || len(calls()) != 3 {
		t.Fatalf("copy was not stopped: elapsed=%s calls=%q", time.Since(start), calls())
	}
}

func TestObserveImageFileStillReportsAnEngineCopyFailure(t *testing.T) {
	// The engine failed by itself; nothing of ours stopped it.
	e, _ := fakeCommandEngine(t, engineStep{Out: commandID}, engineStep{Code: 1, Err: "no such file"}, engineStep{})
	_, err := e.ObserveImageFile(context.Background(), commandImage, "/check", 100)
	if err == nil || !strings.Contains(err.Error(), "container engine cp") || !strings.Contains(err.Error(), "no such file") {
		t.Fatalf("err=%v", err)
	}
}

func TestObserveImageFileKeepsTheEnginesOwnReasonWhenItStopsTheCopy(t *testing.T) {
	wrong := imageArchive(t, tar.Header{Name: "check", Typeflag: tar.TypeSymlink, Linkname: "elsewhere"}, "", false)
	// Keep stdout larger than a pipe buffer: rejecting its first header must
	// interrupt the producer, while its already-emitted diagnostic survives.
	wrong = append(wrong, make([]byte, 1<<20)...)
	e, _ := fakeCommandEngine(t, engineStep{Out: commandID}, engineStep{Out: string(wrong), Err: "no such file", LingerMillis: 20000}, engineStep{})
	_, err := e.ObserveImageFile(context.Background(), commandImage, "/check", 100)
	if err == nil || !strings.Contains(err.Error(), "needs exactly regular file") || !strings.Contains(err.Error(), "no such file") || strings.Contains(err.Error(), "killed") {
		t.Fatalf("err=%v", err)
	}
}

func TestObserveImageFileDoesNotReportCancellationKillAsEngineFailure(t *testing.T) {
	// The archive is complete and valid; the engine process lingers, and the
	// caller cancels while the observation waits for it to end.
	valid := imageArchive(t, tar.Header{Name: "check", Typeflag: tar.TypeReg, Size: 3}, "abc", false)
	e, calls := fakeCommandEngine(t, engineStep{Out: commandID}, engineStep{Out: string(valid), LingerMillis: 20000}, engineStep{})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	time.AfterFunc(1500*time.Millisecond, cancel)
	start := time.Now()
	got, err := e.ObserveImageFile(ctx, commandImage, "/check", 100)
	if !errors.Is(err, context.Canceled) || got != (ImageFileObservation{}) {
		t.Fatalf("observation=%+v err=%v", got, err)
	}
	if strings.Contains(err.Error(), "killed") || strings.Contains(err.Error(), "container engine cp") || time.Since(start) > 10*time.Second || len(calls()) != 3 {
		t.Fatalf("cancellation kill leaked: err=%v elapsed=%s calls=%q", err, time.Since(start), calls())
	}
}

func TestImageObservationMemoPanicReleasesWaitersAndRemembersNothing(t *testing.T) {
	var memo imageObservationMemo
	release, started := make(chan struct{}), make(chan struct{})
	leaderDone := make(chan any, 1)
	go func() {
		defer func() { leaderDone <- recover() }()
		_, _ = memo.observe(context.Background(), "key", func() (ImageFileObservation, error) {
			close(started)
			<-release
			panic("observation panicked")
		})
	}()
	<-started
	waiter := make(chan ImageFileObservation, 1)
	go func() {
		got, _ := memo.observe(context.Background(), "key", func() (ImageFileObservation, error) { return ImageFileObservation{Size: 9}, nil })
		waiter <- got
	}()
	close(release)
	if v := <-leaderDone; v == nil {
		t.Fatal("panic was swallowed")
	}
	select {
	case got := <-waiter:
		if got.Size != 9 {
			t.Fatalf("waiter observation=%+v", got)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("waiter stayed blocked after the leader panicked")
	}
	memo.mu.Lock()
	defer memo.mu.Unlock()
	if len(memo.running) != 0 || len(memo.done) != 1 {
		t.Fatalf("running=%d done=%d", len(memo.running), len(memo.done))
	}
}
