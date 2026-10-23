package container_test

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/adapters/container"
)

// TestProjectImageRunsAGoCheck runs a project check in the example Go
// runtime image under the session hardening: the check reads the workspace,
// writes only Go's caches in the provider home and passes.
//
//	MARKITECT_TEST_ENGINE    absolute container engine path
//	MARKITECT_TEST_GO_IMAGE  example Go image, for example markitect-runtime-go
func TestProjectImageRunsAGoCheck(t *testing.T) {
	enginePath, ref := os.Getenv("MARKITECT_TEST_ENGINE"), os.Getenv("MARKITECT_TEST_GO_IMAGE")
	if enginePath == "" || ref == "" {
		t.Skip("set MARKITECT_TEST_ENGINE and MARKITECT_TEST_GO_IMAGE")
	}
	engine, err := container.NewEngine(enginePath)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	image, err := engine.ImageID(ctx, ref)
	if err != nil {
		t.Fatal(err)
	}
	r := &runtimeSmoke{t: t, ctx: ctx, engine: engine, image: image, store: fmt.Sprintf("check-%d", time.Now().UnixNano())}
	mounts := r.volumes("check")
	r.writeFile("check-mod", mounts, "/work/go.mod", "module example.test/limit\n\ngo 1.27\n")
	r.writeFile("check-test", mounts, "/work/limit_test.go", "package limit\n\nimport \"testing\"\n\nfunc TestLoanLimit(t *testing.T) {\n\tif limit := 3; limit != 3 {\n\t\tt.Fatal(limit)\n\t}\n}\n")
	spec := r.spec("check-run", mounts)
	spec.Network = "none"
	var out strings.Builder
	if err := engine.RunHelper(ctx, spec, []string{"/usr/local/go/bin/go", "test", "./..."}, nil, &out); err != nil {
		t.Fatalf("go test in the project image: %v\n%s", err, out.String())
	}
	if !strings.Contains(out.String(), "ok") {
		t.Fatalf("go test output %q", out.String())
	}
}
