package checks

import (
	"context"
	"encoding/hex"
	"github.com/Glacius-Labs/Markitect/internal/adapters/container"
	"github.com/Glacius-Labs/Markitect/internal/run"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
	"os"
	"strings"
	"testing"
	"time"
)

// This explicit existing opt-in requires a separate bounded container grant.
func TestCheckWorkspaceRuntimeSmoke(t *testing.T) {
	if os.Getenv("MARKITECT_CHECK_WORKSPACE_SMOKE") != "run-check-workspace-smoke" {
		t.Skip("requires separately authorized check runtime smoke; see README.md")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	engine, err := container.NewEngine(os.Getenv("MARKITECT_CHECK_WORKSPACE_ENGINE"))
	if err != nil {
		t.Fatal(err)
	}
	image := os.Getenv("MARKITECT_CHECK_WORKSPACE_CHECK_IMAGE")
	if !strings.HasPrefix(image, "sha256:") || len(image) != 71 {
		t.Fatal("an exact preexisting check image is required")
	}
	if _, err = hex.DecodeString(image[7:]); err != nil {
		t.Fatal(err)
	}
	copies, err := NewLocalCopies(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	runner, err := NewRunner(engine, copies, nil)
	if err != nil {
		t.Fatal(err)
	}
	entry := fixtureEntry(t, "input.bin", []byte{0, 1, 255, 10})
	tree := fixtureTree(t, []snapshot.Entry{entry})
	for _, exit := range []int{0, 23} {
		code := `const fs=require('fs');if(process.cwd()!=='/work'||JSON.stringify(fs.readdirSync('/work'))!=='["input.bin"]'||fs.readdirSync('/home/agent').length)throw Error('copy isolation');if(!fs.readFileSync('input.bin').equals(Buffer.from([0,1,255,10])))throw Error('input bytes');fs.writeFileSync('/work/discarded','mutation');fs.writeFileSync('/home/agent/discarded','cache');process.stdout.write('x'.repeat(4096));process.exitCode=`
		if exit == 0 {
			code += "0"
		} else {
			code += "23"
		}
		req := run.CheckRequest{ID: run.CheckID("bind-check-" + time.Now().Format("150405.000000000")), Tree: tree, ImageID: image, Executable: "/usr/local/bin/node", Args: []string{"-e", code}, Directory: snapshot.RootDirectory(), Limits: run.CheckLimits{MilliCPUs: 1000, MemoryBytes: 256 << 20, Processes: 64}, Timeout: 30 * time.Second, MaxOutputBytes: 512}
		receipt, err := runner.Run(ctx, req)
		if err != nil || !receipt.CopiesClosed || !receipt.Exited || receipt.ExitCode != exit || receipt.Uncertain || receipt.TimedOut || !receipt.Truncated || receipt.Passed(req) != (exit == 0) {
			t.Fatal(receipt, err)
		}
		if _, err = os.Stat(copies.root); !os.IsNotExist(err) {
			t.Fatal("private check copies survived", err)
		}
		remaining, err := engine.Containers(ctx, "markitect.check", string(req.ID))
		if err != nil || len(remaining) != 0 {
			t.Fatal("check container closure unconfirmed", remaining, err)
		}
	}
}
