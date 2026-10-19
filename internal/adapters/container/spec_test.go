package container

import (
	"reflect"
	"strings"
	"testing"
)

const testImage = "sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

func validSpec() ContainerSpec {
	return ContainerSpec{
		Name:        "markitect-s1",
		Image:       testImage,
		Labels:      map[string]string{"io.markitect.store": "store-1", "io.markitect.session": "s1"},
		User:        "1000:1000",
		Mounts:      []Mount{{Volume: "mk-s1-work", Target: "/work"}, {Volume: "mk-s1-home", Target: "/home/agent"}, {Volume: "mk-c7", Target: "/candidate", ReadOnly: true}},
		Workdir:     "/work",
		CPUs:        1.5,
		MemoryBytes: 2 << 30,
		Pids:        512,
		TmpBytes:    256 << 20,
	}
}

func TestRunArgsHardenEveryContainer(t *testing.T) {
	got, err := validSpec().runArgs([]string{AgentdPath, "serve"})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		"run", "--interactive", "--rm", "--init",
		"--name", "markitect-s1",
		"--read-only",
		"--cap-drop", "ALL",
		"--security-opt", "no-new-privileges",
		"--user", "1000:1000",
		"--tmpfs", "/tmp:rw,exec,nosuid,nodev,mode=1777,size=268435456",
		"--workdir", "/work",
		"--cpus", "1.5",
		"--memory", "2147483648",
		"--pids-limit", "512",
		"--label", "io.markitect.session=s1",
		"--label", "io.markitect.store=store-1",
		"--mount", "type=volume,src=mk-s1-work,dst=/work",
		"--mount", "type=volume,src=mk-s1-home,dst=/home/agent",
		"--mount", "type=volume,src=mk-c7,dst=/candidate,readonly",
		"--entrypoint", AgentdPath, testImage, "serve",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("run args:\n got %q\nwant %q", got, want)
	}
}

func TestRunArgsRefuseUnsafeSpecs(t *testing.T) {
	cases := map[string]func(*ContainerSpec){
		"root user":           func(s *ContainerSpec) { s.User = "0:0" },
		"root group":          func(s *ContainerSpec) { s.User = "1000:0" },
		"host network":        func(s *ContainerSpec) { s.Network = "host" },
		"named user":          func(s *ContainerSpec) { s.User = "agent" },
		"image tag":           func(s *ContainerSpec) { s.Image = "markitect-runtime:latest" },
		"short digest":        func(s *ContainerSpec) { s.Image = "sha256:0123" },
		"option as name":      func(s *ContainerSpec) { s.Name = "--privileged" },
		"relative workdir":    func(s *ContainerSpec) { s.Workdir = "work" },
		"unclean workdir":     func(s *ContainerSpec) { s.Workdir = "/work/../etc" },
		"host path volume":    func(s *ContainerSpec) { s.Mounts[0].Volume = "C:/Users/owner/repo" },
		"mount option inject": func(s *ContainerSpec) { s.Mounts[0].Target = "/work,bind-propagation=shared" },
		"mount over tmp":      func(s *ContainerSpec) { s.Mounts[0].Target = "/tmp" },
		"duplicate target":    func(s *ContainerSpec) { s.Mounts[1].Target = "/work" },
		"subpath escapes":     func(s *ContainerSpec) { s.Mounts[0].Subpath = "../other" },
		"subpath parent":      func(s *ContainerSpec) { s.Mounts[0].Subpath = ".." },
		"subpath absolute":    func(s *ContainerSpec) { s.Mounts[0].Subpath = "/etc" },
		"subpath unclean":     func(s *ContainerSpec) { s.Mounts[0].Subpath = "a/../b" },
		"subpath trailing":    func(s *ContainerSpec) { s.Mounts[0].Subpath = "a/" },
		"subpath dot":         func(s *ContainerSpec) { s.Mounts[0].Subpath = "." },
		"subpath inject":      func(s *ContainerSpec) { s.Mounts[0].Subpath = "a,readonly=false" },
		"subpath backslash":   func(s *ContainerSpec) { s.Mounts[0].Subpath = `a\b` },
		"no limits":           func(s *ContainerSpec) { s.Pids = 0 },
		"no store label":      func(s *ContainerSpec) { s.Labels = nil },
		"label newline":       func(s *ContainerSpec) { s.Labels["io.markitect.store"] = "a\nb" },
		"odd network":         func(s *ContainerSpec) { s.Network = "container:other" },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			s := validSpec()
			s.Mounts = append([]Mount(nil), s.Mounts...)
			mutate(&s)
			if args, err := s.runArgs([]string{AgentdPath, "serve"}); err == nil {
				t.Fatalf("unsafe spec accepted: %s", strings.Join(args, " "))
			}
		})
	}
}

func TestRunArgsNeedAnAbsoluteEntry(t *testing.T) {
	for _, entry := range [][]string{nil, {}, {"git"}, {"./tool"}} {
		if args, err := validSpec().runArgs(entry); err == nil {
			t.Fatalf("entry %q accepted: %q", entry, args)
		}
	}
	got, err := validSpec().runArgs([]string{"/usr/bin/git", "status"})
	if err != nil || got[len(got)-3] != "/usr/bin/git" || got[len(got)-1] != "status" {
		t.Fatalf("helper args = %q, %v", got, err)
	}
}

func TestRunArgsPreventImageCopyUpForExactVolumes(t *testing.T) {
	for _, readOnly := range []bool{false, true} {
		spec := validSpec()
		spec.Mounts = []Mount{{Volume: "verified-copy", Target: "/work", NoCopy: true, ReadOnly: readOnly}}
		args, err := spec.runArgs([]string{"/usr/bin/check"})
		if err != nil {
			t.Fatal(err)
		}
		want := "type=volume,src=verified-copy,dst=/work,volume-nocopy"
		if readOnly {
			want += ",readonly"
		}
		if !containsArg(args, want) {
			t.Fatalf("missing exact volume mount %q in %q", want, args)
		}
	}
}

func TestRunArgsMountVolumeSubpathsOverTheBase(t *testing.T) {
	for name, tc := range map[string]struct {
		baseReadOnly bool
		overlays     []Mount
		want         []string
	}{
		"root style: writable base, read-only overlays": {false,
			[]Mount{{Volume: "ws", Target: "/work/lending", Subpath: "lending", ReadOnly: true, NoCopy: true}, {Volume: "ws", Target: "/work/library", Subpath: "library", ReadOnly: true, NoCopy: true}},
			[]string{"type=volume,src=ws,dst=/work,volume-nocopy",
				"type=volume,src=ws,dst=/work/lending,volume-subpath=lending,volume-nocopy,readonly",
				"type=volume,src=ws,dst=/work/library,volume-subpath=library,volume-nocopy,readonly"}},
		"child style: read-only base, writable overlays": {true,
			[]Mount{{Volume: "ws", Target: "/work/lending/reminders", Subpath: "lending/reminders", NoCopy: true}},
			[]string{"type=volume,src=ws,dst=/work,volume-nocopy,readonly",
				"type=volume,src=ws,dst=/work/lending/reminders,volume-subpath=lending/reminders,volume-nocopy"}},
	} {
		t.Run(name, func(t *testing.T) {
			spec := validSpec()
			spec.Mounts = append([]Mount{{Volume: "ws", Target: "/work", NoCopy: true, ReadOnly: tc.baseReadOnly}}, tc.overlays...)
			args, err := spec.runArgs([]string{"/usr/bin/check"})
			if err != nil {
				t.Fatal(err)
			}
			var got []string
			for i, a := range args {
				if a == "--mount" {
					got = append(got, args[i+1])
				}
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("mounts %q, want %q", got, tc.want)
			}
		})
	}
}
