package cli

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/buildinfo"
)

func TestCockpitPassesItsSelectionAndReportsWhereItServes(t *testing.T) {
	var got CockpitRequest
	c := New(nil, nil, nil, nil, nil, nil, nil).WithCockpit(func(_ context.Context, r CockpitRequest, ready func(string)) error {
		got = r
		ready("http://127.0.0.1:7341/")
		return nil
	})
	var out, errout bytes.Buffer
	config := filepath.Join(t.TempDir(), "run.json")
	args := []string{"cockpit", "--state", "a", "--state", "b", "--run-config", config, "--listen", "127.0.0.1:9000"}
	if code := c.Run(context.Background(), args, &out, &errout); code != 0 {
		t.Fatalf("%d: %s %s", code, out.String(), errout.String())
	}
	if !slices.Equal(got.States, []string{"a", "b"}) || got.RunConfig != config || got.Listen != "127.0.0.1:9000" || got.Build != buildinfo.Build() {
		t.Fatalf("request = %+v", got)
	}
	if !strings.Contains(out.String(), `"url": "http://127.0.0.1:7341/"`) || !strings.Contains(out.String(), `"serving"`) {
		t.Fatal(out.String())
	}
}

func TestCockpitRefusesIncompleteSelectionsAndReportsFailures(t *testing.T) {
	calls := 0
	c := New(nil, nil, nil, nil, nil, nil, nil).WithCockpit(func(context.Context, CockpitRequest, func(string)) error {
		calls++
		return errors.New("listen address is not a loopback address")
	})
	for _, args := range [][]string{{"cockpit"}, {"cockpit", "--run-config", "relative.json"}, {"cockpit", "--state", "a", "extra"}} {
		var out, errout bytes.Buffer
		if code := c.Run(context.Background(), args, &out, &errout); code != 2 {
			t.Fatalf("%v = %d", args, code)
		}
	}
	if calls != 0 {
		t.Fatal("an incomplete selection reached the cockpit")
	}
	var out, errout bytes.Buffer
	if code := c.Run(context.Background(), []string{"cockpit", "--state", "a"}, &out, &errout); code != 3 || !strings.Contains(out.String(), "loopback") {
		t.Fatalf("%d %s", code, out.String())
	}
	if code := New(nil, nil, nil, nil, nil, nil, nil).Run(context.Background(), []string{"cockpit", "--state", "a"}, &out, &errout); code != 3 {
		t.Fatalf("an unconfigured cockpit = %d", code)
	}
}

func TestCockpitObservesTheCurrentRepositoryByDefault(t *testing.T) {
	var got CockpitRequest
	c := New(nil, nil, nil, nil, nil, nil, nil).WithCockpit(func(_ context.Context, r CockpitRequest, _ func(string)) error {
		got = r
		return nil
	})
	var out, errout bytes.Buffer
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if code := c.Run(context.Background(), []string{"cockpit", "--state", "a"}, &out, &errout); code != 0 || got.Selection.Repository != wd {
		t.Fatalf("%d: repository %q", code, got.Selection.Repository)
	}
	if code := c.Run(context.Background(), []string{"cockpit", "--state", "a", "--repo", "other"}, &out, &errout); code != 0 || got.Selection.Repository != filepath.Join(wd, "other") {
		t.Fatalf("%d: repository %q", code, got.Selection.Repository)
	}
}

func TestCockpitSelectsTheModelAndDependenciesAsCheckDoes(t *testing.T) {
	var got CockpitRequest
	c := New(nil, nil, nil, nil, nil, nil, nil).WithCockpit(func(_ context.Context, r CockpitRequest, _ func(string)) error {
		got = r
		return nil
	})
	var out, errout bytes.Buffer
	args := []string{"cockpit", "--state", "a", "--repo", "project", "--dependency", ".markitect/dependencies/core/model.yaml", "--dependency", ".markitect/dependencies/software/model.yaml"}
	if code := c.Run(context.Background(), args, &out, &errout); code != 0 {
		t.Fatalf("%d: %s", code, errout.String())
	}
	if !filepath.IsAbs(got.Selection.Repository) || filepath.Base(got.Selection.Repository) != "project" || got.Selection.Local.ManifestPath.String() != ".markitect/model.yaml" ||
		len(got.Selection.Dependencies) != 2 || got.Selection.Dependencies[1].ManifestPath.String() != ".markitect/dependencies/software/model.yaml" {
		t.Fatalf("selection = %+v", got.Selection)
	}
	if code := c.Run(context.Background(), []string{"cockpit", "--state", "a", "--dependency", "../outside.yaml"}, &out, &errout); code != 2 {
		t.Fatalf("an invalid dependency = %d", code)
	}
}

func TestCockpitOpensItsWindowUnlessToldNotTo(t *testing.T) {
	for _, tt := range []struct {
		name   string
		args   []string
		opener func(string) error
		opened []string
		stderr string
	}{
		{"by default", []string{"cockpit", "--state", "a"}, nil, []string{"http://127.0.0.1:7341/"}, ""},
		{"not with --no-open", []string{"cockpit", "--state", "a", "--no-open"}, nil, nil, ""},
		{"a failure is reported and not fatal", []string{"cockpit", "--state", "a"}, func(string) error { return errors.New("no browser") }, []string{"http://127.0.0.1:7341/"}, "no browser"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var opened []string
			c := New(nil, nil, nil, nil, nil, nil, nil).WithCockpit(func(_ context.Context, _ CockpitRequest, ready func(string)) error {
				ready("http://127.0.0.1:7341/")
				return nil
			}).WithWindowOpener(func(url string) error {
				opened = append(opened, url)
				if tt.opener != nil {
					return tt.opener(url)
				}
				return nil
			})
			var out, errout bytes.Buffer
			if code := c.Run(context.Background(), tt.args, &out, &errout); code != 0 {
				t.Fatalf("%d: %s %s", code, out.String(), errout.String())
			}
			if !slices.Equal(opened, tt.opened) || !strings.Contains(errout.String(), tt.stderr) {
				t.Fatalf("opened %v, stderr %q", opened, errout.String())
			}
			if !strings.Contains(out.String(), `"url": "http://127.0.0.1:7341/"`) {
				t.Fatal("the address is printed either way: " + out.String())
			}
		})
	}
}
