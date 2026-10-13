//go:build windows

package recordstore

import (
	"encoding/binary"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"testing"
)

func TestNativeRelativeDirectoryCreation(t *testing.T) {
	s, root := openFixture(t)
	f, err := s.root.Open(".")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	p, err := platformWritableParent(f)
	if err != nil {
		t.Fatal(err)
	}
	defer p.Close()
	if err := platformMkdir(s.root, p, "child"); err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("actual entries: %v", entries)
	if _, err := s.root.Lstat("child"); err != nil {
		t.Fatal(err)
	}
}
func TestExclusiveLeaseRemainsReadableByExistingProcessAudit(t *testing.T) {
	s, root := openFixture(t)
	data := []byte("original Runtime owner bytes")
	lease, err := s.Exclusive(Run, "", ".write.lock", data, 0, nil)
	if err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(root, filepath.FromSlash(RootPath), "run", ".write.lock")
	wire, err := os.ReadFile(p)
	if err != nil || string(wire) != string(data) {
		t.Fatalf("plain audit read blocked: %q %v", wire, err)
	}
	if err := lease.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(p); !os.IsNotExist(err) {
		t.Fatal("lease retained after release")
	}
}
func TestActualWindowsJunctionRefusesWithoutTargetMutation(t *testing.T) {
	s, root := openFixture(t)
	target := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, ".markitect"), 0700); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, ".markitect", "records")
	if output, err := exec.Command("cmd", "/c", "mklink", "/J", link, target).CombinedOutput(); err != nil {
		t.Skipf("actual junction creation unavailable: %v %s", err, output)
	}
	if err := s.Create(fixtureName(), []byte("forbidden")); err == nil {
		t.Fatal("junction accepted")
	}
	if entries, err := os.ReadDir(target); err != nil || len(entries) != 0 {
		t.Fatalf("junction target mutated: %v %v", entries, err)
	}
}
func TestWrongWindowsCapabilityRefusesBeforeControlCreation(t *testing.T) {
	for _, control := range []string{"abi", "version", "api", "filesystem", "remote", "device", "query", "query-length", "rights", "flush"} {
		t.Run(control, func(t *testing.T) {
			abi, version, volume, apis, parent, flush := writerABI, writerVersion, writerVolume, findWriterAPIs, openWriterParent, writerFlush
			defer func() {
				writerABI, writerVersion, writerVolume, findWriterAPIs, openWriterParent, writerFlush = abi, version, volume, apis, parent, flush
			}()
			s, root := openFixture(t)
			switch control {
			case "abi":
				writerABI = func() bool { return false }
			case "version":
				writerVersion = func(v *versionInfo) uint32 { v.Major = 10; v.Build = 26101; return 0 }
			case "api":
				findWriterAPIs = func() error { return errors.New("required API unavailable") }
				openWriterParent = func(*os.File) (*os.File, error) {
					t.Fatal("writer parent opened before required API admission")
					return nil, syscall.EIO
				}
			case "rights":
				openWriterParent = func(*os.File) (*os.File, error) { return nil, syscall.EACCES }
			case "flush":
				writerFlush = func(*os.File) (uintptr, error) { return 0, syscall.EIO }
			default:
				writerVolume = func(f *os.File, data []byte, class uint32) (uintptr, uint32) {
					length, status := volume(f, data, class)
					if control == "query" {
						return 0, 0xC0000022
					}
					if control == "query-length" {
						return 1, 0
					}
					if class == 5 && control == "filesystem" {
						binary.LittleEndian.PutUint16(data[12:], uint16('X'))
					}
					if class == 4 && control == "remote" {
						binary.LittleEndian.PutUint32(data[4:], 0x10)
					}
					if class == 4 && control == "device" {
						binary.LittleEndian.PutUint32(data[:4], 0)
					}
					return length, status
				}
			}
			if _, err := s.Exclusive(Run, "", ".write.lock", nil, 0, nil); err == nil {
				t.Fatal("unsupported writer acquired lease")
			}
			if err := s.Create(fixtureName(), []byte("record")); err == nil {
				t.Fatal("unsupported writer created record")
			}
			if _, err := os.Stat(filepath.Join(root, ".markitect")); !os.IsNotExist(err) {
				t.Fatalf("capability refusal created control/container: %v", err)
			}
			if entries, err := s.List(Run, "", ""); err != nil || len(entries) != 0 {
				t.Fatalf("read-only route changed: %v %v", entries, err)
			}
		})
	}
}
