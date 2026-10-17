package container

import (
	"reflect"
	"strings"
	"testing"
)

const testImage = "sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

func validSpec() SessionSpec {
	return SessionSpec{
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
	got, err := validSpec().runArgs()
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
		"--tmpfs", "/tmp:rw,nosuid,nodev,mode=1777,size=268435456",
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
	cases := map[string]func(*SessionSpec){
		"root user":           func(s *SessionSpec) { s.User = "0:0" },
		"root group":          func(s *SessionSpec) { s.User = "1000:0" },
		"host network":        func(s *SessionSpec) { s.Network = "host" },
		"named user":          func(s *SessionSpec) { s.User = "agent" },
		"image tag":           func(s *SessionSpec) { s.Image = "markitect-runtime:latest" },
		"short digest":        func(s *SessionSpec) { s.Image = "sha256:0123" },
		"option as name":      func(s *SessionSpec) { s.Name = "--privileged" },
		"relative workdir":    func(s *SessionSpec) { s.Workdir = "work" },
		"unclean workdir":     func(s *SessionSpec) { s.Workdir = "/work/../etc" },
		"host path volume":    func(s *SessionSpec) { s.Mounts[0].Volume = "C:/Users/owner/repo" },
		"mount option inject": func(s *SessionSpec) { s.Mounts[0].Target = "/work,bind-propagation=shared" },
		"mount over tmp":      func(s *SessionSpec) { s.Mounts[0].Target = "/tmp" },
		"duplicate target":    func(s *SessionSpec) { s.Mounts[1].Target = "/work" },
		"no limits":           func(s *SessionSpec) { s.Pids = 0 },
		"no store label":      func(s *SessionSpec) { s.Labels = nil },
		"label newline":       func(s *SessionSpec) { s.Labels["io.markitect.store"] = "a\nb" },
		"odd network":         func(s *SessionSpec) { s.Network = "container:other" },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			s := validSpec()
			s.Mounts = append([]Mount(nil), s.Mounts...)
			mutate(&s)
			if args, err := s.runArgs(); err == nil {
				t.Fatalf("unsafe spec accepted: %s", strings.Join(args, " "))
			}
		})
	}
}
