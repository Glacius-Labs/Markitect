package container

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"flag"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/agentd/frame"
)

var bindNative = struct {
	enabled                         *bool
	engine, image, agentd, evidence *string
}{
	flag.Bool("markitect-bind-native", false, "separate finite dispatch for one credential-free bind mapping probe"),
	flag.String("markitect-bind-engine", "", "absolute local engine executable"),
	flag.String("markitect-bind-image", "", "immutable runtime image"),
	flag.String("markitect-bind-agentd", "", "source-bound agentd version"),
	flag.String("markitect-bind-evidence", "", "new absolute evidence directory"),
}

// TestLocalBindNativeProbe is a technical engine/agentd mapping observation,
// not a provider, LocalGit implementation, native Manager or product Run.
func TestLocalBindNativeProbe(t *testing.T) {
	if !*bindNative.enabled {
		t.Skip("UNEXECUTED: concrete finite bind dispatch required")
	}
	if !filepath.IsAbs(*bindNative.engine) || !filepath.IsAbs(*bindNative.evidence) || !imageIDPattern.MatchString(*bindNative.image) || *bindNative.agentd == "" {
		t.Fatal("explicit engine/image/version/evidence bindings required")
	}
	if err := os.Mkdir(*bindNative.evidence, 0700); err != nil {
		t.Fatal("evidence directory must be new")
	}
	var nonceBytes [16]byte
	if _, err := rand.Read(nonceBytes[:]); err != nil {
		t.Fatal("private binding nonce unavailable")
	}
	nonce := hex.EncodeToString(nonceBytes[:])
	name := "bind-native-" + nonce
	copyPath, home := filepath.Join(*bindNative.evidence, "private-fixture"), filepath.Join(*bindNative.evidence, "private-home")
	control := filepath.Join(copyPath, ".git", ".markitect-bind")
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	engine, err := NewEngine(*bindNative.engine)
	if err != nil {
		t.Fatal("explicit engine unavailable")
	}
	var session *Session
	passed, closed, absent, removed := false, false, false, false
	mappingObserved, modesPreserved := false, false
	var programExit *int
	started := time.Now().UTC()
	defer func() {
		cleanup, done := context.WithTimeout(context.Background(), 45*time.Second)
		defer done()
		if session != nil {
			closed = session.Close() == nil
		}
		if session != nil {
			if err := os.WriteFile(filepath.Join(*bindNative.evidence, "program.stderr.log"), session.Diagnostics(), 0600); err != nil {
				t.Error("mapping diagnostics write failed")
			}
		}
		containers, err := engine.Containers(cleanup, "markitect.store", name)
		absent = err == nil && len(containers) == 0
		// Remove only the two exact fixture children, after container absence.
		if absent {
			removed = true
			for _, path := range []string{copyPath, home} {
				rel, err := filepath.Rel(*bindNative.evidence, path)
				if err != nil || (rel != "private-fixture" && rel != "private-home") || filepath.Dir(path) != *bindNative.evidence {
					removed = false
					continue
				}
				if err := os.RemoveAll(path); err != nil {
					removed = false
					continue
				}
				if _, err := os.Lstat(path); !os.IsNotExist(err) {
					removed = false
				}
			}
		}
		status := "incomplete"
		if passed && closed && absent && removed && !t.Failed() {
			status = "passed"
		}
		if !closed || !absent || !removed {
			t.Error("bind fixture closure incomplete")
		}
		record, _ := json.MarshalIndent(struct {
			Status, ProbeID, Image, Agentd, Started, Ended            string
			Protocol                                                  int
			PrivateFixture, PrivateHome                               string
			SessionClosed, ContainerAbsent, FixtureDirectoriesRemoved bool
			MappingObserved, ModesPreserved                           bool
			ProgramExitCode                                           *int
			ProviderSessions, CredentialReads                         int
		}{status, name, *bindNative.image, *bindNative.agentd, started.Format(time.RFC3339Nano), time.Now().UTC().Format(time.RFC3339Nano), frame.ProtocolVersion,
			copyPath, home, closed, absent, removed, mappingObserved, modesPreserved, programExit, 0, 0}, "", "  ")
		if err := os.WriteFile(filepath.Join(*bindNative.evidence, "probe.json"), record, 0600); err != nil {
			t.Error("bind evidence write failed")
		}
	}()
	if err := os.MkdirAll(control, 0700); err != nil {
		t.Fatal("private fixture unavailable")
	}
	if err := os.Mkdir(home, 0700); err != nil {
		t.Fatal("private home unavailable")
	}
	for path, content := range map[string]string{filepath.Join(control, "host"): nonce, filepath.Join(copyPath, "candidate.txt"): "unchanged fixture candidate\n"} {
		if err := os.WriteFile(path, []byte(content), 0600); err != nil {
			t.Fatal("fixture write failed")
		}
	}
	for _, file := range []struct {
		name, text string
		mode       os.FileMode
	}{
		{"plain.txt", "plain mode fixture\n", 0644}, {"executable.sh", "#!/bin/sh\nexit 0\n", 0755},
	} {
		path := filepath.Join(copyPath, file.name)
		if err := os.WriteFile(path, []byte(file.text), file.mode); err != nil {
			t.Fatal("mode fixture creation failed")
		}
		if err := os.Chmod(path, file.mode); err != nil {
			t.Fatal("mode fixture chmod failed")
		}
	}
	spec := ContainerSpec{Name: name, Image: *bindNative.image, Labels: map[string]string{"markitect.store": name}, User: "1000:1000", Workdir: "/work",
		CPUs: 1, MemoryBytes: 512 << 20, Pids: 64, TmpBytes: 16 << 20, Network: "none",
		Mounts: []Mount{{BindDirectory: copyPath, Target: "/work"}, {BindDirectory: copyPath, Target: "/readonly", ReadOnly: true}, {BindDirectory: home, Target: "/home/agent"}}}
	session, err = engine.Start(ctx, spec)
	if err != nil {
		t.Fatal("bind session failed before any credential or provider")
	}
	if session.Hello().Version != *bindNative.agentd {
		t.Fatal("agentd source binding differs")
	}
	// One fixed non-provider program, no shell, auth, model, network or writes
	// to candidate files. The caller never supplies source paths as argv code.
	script := `const fs=require('node:fs'); const a=fs.readFileSync('/work/.git/.markitect-bind/host','utf8'); const b=fs.readFileSync('/readonly/.git/.markitect-bind/host','utf8'); if(a!==b)throw Error('mapping mismatch'); fs.writeFileSync('/work/.git/.markitect-bind/container',a+'-container',{flag:'wx',mode:0o600}); fs.writeFileSync('/home/agent/mapping',a,{flag:'wx',mode:0o600}); let ro=false; try{fs.writeFileSync('/readonly/.git/.markitect-bind/forbidden','unexpected',{flag:'wx'})}catch(e){if(e.code==='EROFS')ro=true;else throw e} const plain=fs.statSync('/work/plain.txt').mode&0o777; const executable=fs.statSync('/work/executable.sh').mode&0o777; const result=require('node:child_process').spawnSync('/work/executable.sh'); process.stdout.write(JSON.stringify({nonce:a,readOnly:ro,uid:process.getuid(),gid:process.getgid(),plainMode:plain,executableMode:executable,executableExit:result.status,plainText:fs.readFileSync('/work/plain.txt','utf8')}));`
	if _, err := session.StartProvider(ctx, frame.Start{Argv: []string{"/usr/local/bin/node", "-e", script}, Dir: "/work"}); err != nil {
		t.Fatal("fixed non-provider mapping program did not start")
	}
	select {
	case exit := <-session.Exited():
		code := exit.Code
		programExit = &code
	case <-ctx.Done():
		t.Fatal("mapping program deadline exceeded")
	}
	if err := session.Close(); err != nil {
		t.Fatal("mapping session closure failed")
	}
	output, err := io.ReadAll(io.LimitReader(session.Stdout(), 4097))
	if writeErr := os.WriteFile(filepath.Join(*bindNative.evidence, "mapping.json"), output, 0600); writeErr != nil {
		t.Fatal("mapping observation write failed")
	}
	var observed struct {
		Nonce                               string
		ReadOnly                            bool
		UID, GID, PlainMode, ExecutableMode int
		ExecutableExit                      *int
		PlainText                           string
	}
	if err != nil || len(output) > 4096 || programExit == nil || *programExit != 0 || json.Unmarshal(output, &observed) != nil || observed.Nonce != nonce || !observed.ReadOnly || observed.UID != 1000 || observed.GID != 1000 || observed.PlainText != "plain mode fixture\n" {
		t.Fatal("actual mapping response mismatch")
	}
	for path, expected := range map[string]string{filepath.Join(control, "container"): nonce + "-container", filepath.Join(home, "mapping"): nonce, filepath.Join(copyPath, "candidate.txt"): "unchanged fixture candidate\n"} {
		actual, err := os.ReadFile(path)
		if err != nil || string(actual) != expected {
			t.Fatal("exact host readback mismatch")
		}
	}
	if _, err := os.Lstat(filepath.Join(control, "forbidden")); !os.IsNotExist(err) {
		t.Fatal("read-only boundary changed host bytes")
	}
	mappingObserved = true
	modesPreserved = observed.PlainMode == 0644 && observed.ExecutableMode == 0755 && observed.ExecutableExit != nil && *observed.ExecutableExit == 0
	if !modesPreserved {
		t.Fatal("bind maps bytes but does not preserve required 0644/0755 executable semantics; retain actual mapping.json")
	}
	passed = true
}
