package runner

import (
	"reflect"
	"strings"
	"testing"
)

func TestHostPathSpellsMountSourcesPerPlatform(t *testing.T) {
	for _, tc := range []struct {
		goos, in, want string
	}{
		{"linux", "/home/me/repo", "/home/me/repo"},
		{"linux", "/mnt/c/Users/me/repo", "/mnt/c/Users/me/repo"},
		{"linux", "/home/me/my repo", "/home/me/my repo"},
		{"darwin", "/Users/me/repo", "/Users/me/repo"},
		{"windows", `C:\Users\me\repo`, "C:/Users/me/repo"},
		{"windows", `c:\Users\me\My Repo`, "C:/Users/me/My Repo"},
		{"windows", `D:/work/repo`, "D:/work/repo"},
		{"windows", `C:\Users\me\.codex-runtime\auth.json`, "C:/Users/me/.codex-runtime/auth.json"},
	} {
		got, err := HostPath(tc.goos, tc.in)
		if err != nil || got != tc.want {
			t.Errorf("HostPath(%s, %q) = %q, %v; want %q", tc.goos, tc.in, got, err, tc.want)
		}
	}
	for _, tc := range []struct{ goos, in string }{
		{"linux", ""},
		{"linux", "relative/repo"},
		{"linux", "/"},
		{"linux", "/home/me/../repo"},
		{"linux", "/home/me/repo/"},
		{"linux", "/home/me,readonly=false/repo"},
		{"linux", `/home/me/"repo"`},
		{"linux", "/home/me/repo\nx"},
		{"windows", `C:\`},
		{"windows", `C:repo`},
		{"windows", `repo`},
		{"windows", `\Users\me\repo`},
		{"windows", `\\server\share\repo`},
		{"windows", `\\wsl$\Ubuntu\home\me\repo`},
		{"windows", `\\?\C:\Users\me\repo`},
		{"windows", `C:\Users\me\..\repo`},
		{"windows", `C:\Users\me\repo\`},
		{"windows", `C:\Users\me\re,po`},
		{"windows", `C:\Users\me\a:b`},
		{"windows", `/home/me/repo`},
	} {
		if got, err := HostPath(tc.goos, tc.in); err == nil {
			t.Errorf("HostPath(%s, %q) = %q, want refusal", tc.goos, tc.in, got)
		}
	}
}

func TestMountsHoldTheCompleteRunnerSet(t *testing.T) {
	spec := MountSpec{
		Checkout:    `C:\Users\me\repo`,
		Invocation:  `C:\state\proj\runner\abc\invocation`,
		Credentials: []string{`C:\Users\me\login-a.json`, `C:\Users\me\login-b.json`},
		Output:      `C:\state\proj\runner\abc\result`,
		StateVolume: "markitect-state-proj",
	}
	got, err := Mounts("windows", spec)
	if err != nil {
		t.Fatal(err)
	}
	want := []Mount{
		{Type: "bind", Source: "C:/Users/me/repo", Target: "/markitect/checkout", ReadOnly: true},
		{Type: "bind", Source: "C:/state/proj/runner/abc/invocation", Target: "/markitect/invocation", ReadOnly: true},
		{Type: "bind", Source: "C:/Users/me/login-a.json", Target: "/markitect/credentials/1.json", ReadOnly: true},
		{Type: "bind", Source: "C:/Users/me/login-b.json", Target: "/markitect/credentials/2.json", ReadOnly: true},
		{Type: "bind", Source: "/var/run/docker.sock", Target: "/var/run/docker.sock"},
		{Type: "bind", Source: "C:/state/proj/runner/abc/result", Target: "/markitect/output"},
		{Type: "volume", Source: "markitect-state-proj", Target: "/markitect/state"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("mounts:\n got %#v\nwant %#v", got, want)
	}
	// The same set from WSL keeps POSIX paths as given.
	linux := MountSpec{Checkout: "/mnt/c/Users/me/repo", Invocation: "/home/me/state/inv", Credentials: []string{"/home/me/login.json"}, Output: "/home/me/state/out", StateVolume: "markitect-state-proj"}
	got, err = Mounts("linux", linux)
	if err != nil || got[0].Source != "/mnt/c/Users/me/repo" || got[2].Source != "/home/me/login.json" || got[3].Source != EngineSocket {
		t.Fatalf("linux mounts: %#v %v", got, err)
	}
}

func TestMountsRefuseIncompleteOrUnsafeSets(t *testing.T) {
	good := MountSpec{Checkout: "/repo", Invocation: "/inv", Credentials: []string{"/login.json"}, Output: "/out", StateVolume: "markitect-state-proj"}
	for name, mutate := range map[string]func(*MountSpec){
		"no login":          func(s *MountSpec) { s.Credentials = nil },
		"relative checkout": func(s *MountSpec) { s.Checkout = "repo" },
		"volume name":       func(s *MountSpec) { s.StateVolume = "../state" },
		"output":            func(s *MountSpec) { s.Output = "/out,readonly" },
	} {
		spec := good
		mutate(&spec)
		if _, err := Mounts("linux", spec); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
	secret := "/home/me/secret,login.json"
	spec := good
	spec.Credentials = []string{secret}
	if _, err := Mounts("linux", spec); err == nil || strings.Contains(err.Error(), "secret") {
		t.Fatalf("unsafe login path refused without repeating it: %v", err)
	}
}

func TestRunArgsStartTheRunnerWithoutNetwork(t *testing.T) {
	base := strings.Repeat("a", 40)
	image := "sha256:" + strings.Repeat("b", 64)
	mounts := []Mount{
		{Type: "bind", Source: "/repo", Target: CheckoutDir, ReadOnly: true},
		{Type: "volume", Source: "markitect-state-proj", Target: StateDir},
	}
	got, err := RunArgs(RunSpec{Name: "markitect-runner-1", Label: "1", Image: image, Base: base, Model: ".markitect/model.yaml", Dependencies: []string{"libs/core/model.yaml"}, Mounts: mounts})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"run", "--rm", "--init", "--pull", "never", "--name", "markitect-runner-1", "--label", "markitect.runner=1",
		"--network", "none", "--security-opt", "no-new-privileges",
		"--mount", "type=bind,source=/repo,target=/markitect/checkout,readonly",
		"--mount", "type=volume,source=markitect-state-proj,target=/markitect/state",
		image, "--base", base, "--model", ".markitect/model.yaml", "--dependency", "libs/core/model.yaml"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("args:\n got %q\nwant %q", got, want)
	}
	for name, spec := range map[string]RunSpec{
		"image tag":      {Name: "n", Label: "l", Image: "markitect-runner:latest", Base: base},
		"short base":     {Name: "n", Label: "l", Image: image, Base: "abc"},
		"absolute model": {Name: "n", Label: "l", Image: image, Base: base, Model: "/repo/.markitect/model.yaml"},
		"escaping dep":   {Name: "n", Label: "l", Image: image, Base: base, Dependencies: []string{"../other/model.yaml"}},
		"repeated mount": {Name: "n", Label: "l", Image: image, Base: base, Mounts: []Mount{mounts[0], mounts[0]}},
		"bad name":       {Name: "-n", Label: "l", Image: image, Base: base},
	} {
		if _, err := RunArgs(spec); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
	if got := StopArgs("markitect-runner-1", 240e9); !reflect.DeepEqual(got, []string{"stop", "--time", "240", "markitect-runner-1"}) {
		t.Fatalf("stop args %q", got)
	}
}

func TestBoundedBufferDrainsBeyondItsLimit(t *testing.T) {
	b := &boundedBuffer{limit: 4}
	for _, chunk := range []string{"ab", "cde", "f"} {
		if n, err := b.Write([]byte(chunk)); err != nil || n != len(chunk) {
			t.Fatalf("write %q: %d %v", chunk, n, err)
		}
	}
	if !b.exceeded || b.String() != "ab" {
		t.Fatalf("exceeded=%v kept %q", b.exceeded, b.String())
	}
}

func TestGitErrorsNameTheSubcommand(t *testing.T) {
	if got := subcommand([]string{"-c", "user.name=x", "-c", "user.email=y", "commit", "-m", "m"}); got != "commit" {
		t.Fatalf("subcommand %q", got)
	}
	if got := subcommand([]string{"status"}); got != "status" {
		t.Fatalf("subcommand %q", got)
	}
}

func TestRepositoryRelative(t *testing.T) {
	for _, p := range []string{".markitect/model.yaml", "libs/core/model.yaml", "a"} {
		if !RepositoryRelative(p, false) {
			t.Errorf("refused %q", p)
		}
	}
	if !RepositoryRelative(".", true) || RepositoryRelative(".", false) {
		t.Error("dot handling")
	}
	for _, p := range []string{"", "/abs", "../x", "a/../b", "a//b", "a/", `a\b`, "C:/x", "./a", "a/./b"} {
		if RepositoryRelative(p, true) {
			t.Errorf("accepted %q", p)
		}
	}
}

func TestResultValidateRequiresAConsistentRecord(t *testing.T) {
	base, commit := strings.Repeat("1", 40), strings.Repeat("2", 40)
	id := strings.Repeat("3", 32)
	ok := Result{FormatVersion: 1, ApplyStatus: StatusPassed, ApplyID: id, Base: base, Commit: commit, Ref: ResultRef, Paths: []string{"a.txt"}}
	if err := ok.validate(); err != nil {
		t.Fatal(err)
	}
	for name, mutate := range map[string]func(*Result){
		"format":              func(r *Result) { r.FormatVersion = 2 },
		"status":              func(r *Result) { r.ApplyStatus = "ok" },
		"base":                func(r *Result) { r.Base = "x" },
		"no commit":           func(r *Result) { r.Commit = "" },
		"ref":                 func(r *Result) { r.Ref = "refs/heads/main" },
		"no paths":            func(r *Result) { r.Paths = nil },
		"escaping path":       func(r *Result) { r.Paths = []string{"../x"} },
		"repeated path":       func(r *Result) { r.Paths = []string{"a", "a"} },
		"commit with error":   func(r *Result) { r.Error = "boom" },
		"commit on failure":   func(r *Result) { r.ApplyStatus = StatusFailed },
		"commit on no change": func(r *Result) { r.NoChange = true },
		"apply id":            func(r *Result) { r.ApplyID = "short" },
	} {
		r := ok
		mutate(&r)
		if err := r.validate(); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
	failed := Result{FormatVersion: 1, ApplyStatus: StatusFailed, Base: base}
	if err := failed.validate(); err != nil {
		t.Fatal(err)
	}
}
