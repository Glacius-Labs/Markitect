package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/host/government"
	"go.yaml.in/yaml/v3"
)

func TestGovernmentCLIRealFixtureIsReadonlyAndBounded(t *testing.T) {
	repo, err := filepath.Abs("../../../examples/government")
	if err != nil {
		t.Fatal(err)
	}
	before := map[string]string{}
	if err := filepath.WalkDir(repo, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			b, e := os.ReadFile(p)
			if e != nil {
				return e
			}
			before[p] = government.BytesDigest(b)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name   string
		args   []string
		code   int
		status string
	}{
		{"inspect", []string{"--action", "inspect"}, 1, ""},
		{"positive", []string{"--action", "plan", "--order", "order.yaml"}, 0, "planned-scoped"},
		{"negative", []string{"--action", "plan", "--order", "negative-order.yaml"}, 1, "blocked"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			args := []string{"government", "--repo", repo, "--config", "government.yaml"}
			args = append(args, tc.args...)
			var out, errout bytes.Buffer
			code := Run(args, &out, &errout)
			if code != tc.code {
				t.Fatalf("code %d: %s\n%s", code, errout.String(), out.String())
			}
			if tc.status != "" {
				var p government.Plan
				if err := yaml.Unmarshal(out.Bytes(), &p); err != nil {
					t.Fatal(err)
				}
				if p.Status != tc.status || !p.Provisional || p.ActiveConstitution == "" || len(p.Affected) != 2 {
					t.Fatalf("unbound plan: %+v", p)
				}
				if !reflect.DeepEqual(p.Survey.Unknown, []string{"misc/unassigned.md", "negative-order.yaml", "order.yaml"}) {
					t.Fatalf("unknowns: %+v", p.Survey.Unknown)
				}
				if tc.name == "negative" && !strings.Contains(out.String(), "authority.missing") {
					t.Fatal("negative authority case missing")
				}
			} else if !strings.Contains(out.String(), "misc/unassigned.md") {
				t.Fatal("native unknown hidden")
			}
		})
	}
	for p, digest := range before {
		b, err := os.ReadFile(p)
		if err != nil || government.BytesDigest(b) != digest {
			t.Fatalf("read-only command modified %s", p)
		}
	}
}

func TestGovernmentCLIRejectsMutationAndMalformedRequests(t *testing.T) {
	for _, args := range [][]string{{"--write"}, {"--write=false"}, {"--action", "schema", "--runtime="}, {"--runtime", "runtime.json"}, {"--revision", "HEAD"}, {"--action", "run"}, {"--action", "run", "--write", "--config", "government.yaml", "--order", "order.yaml", "--runtime", "relative.json"}, {"--action", "promote"}, {"--action", "plan"}, {"--action", "inspect", "--order", "order.yaml"}, {"--action", "schema", "--config", "government.yaml"}, {"--action", "schema", "positional"}} {
		var out, errout bytes.Buffer
		if Run(append([]string{"government"}, args...), &out, &errout) != 2 {
			t.Fatalf("accepted %v", args)
		}
	}
	var out, errout bytes.Buffer
	if Run([]string{"government", "--action", "schema"}, &out, &errout) != 0 || !strings.Contains(out.String(), government.APIVersion) {
		t.Fatalf("schema: %s %s", out.String(), errout.String())
	}
}

func TestGovernmentQueueCLIRejectsUnboundAndMixedRequests(t *testing.T) {
	backlog := filepath.Join(t.TempDir(), "backlog.json")
	queue := filepath.Join(t.TempDir(), "queue")
	for _, args := range [][]string{
		{"--action", "queue", "--backlog", backlog},
		{"--action", "queue", "--backlog", "relative.json", "--write"},
		{"--action", "queue", "--backlog", backlog, "--queue", queue, "--write"},
		{"--action", "resume", "--backlog", backlog, "--write"},
		{"--action", "resume", "--backlog", backlog, "--queue", "relative", "--write"},
		{"--action", "queue", "--backlog", backlog, "--runtime", backlog, "--write"},
		{"--action", "resume", "--backlog", backlog, "--queue", queue, "--config", "government.yaml", "--write"},
		{"--action", "run", "--backlog="},
		{"--action", "schema", "--queue="},
	} {
		var out, errout bytes.Buffer
		if Run(append([]string{"government"}, args...), &out, &errout) != 2 {
			t.Fatalf("accepted unbound queue invocation %v: %s", args, out.String())
		}
	}
}
