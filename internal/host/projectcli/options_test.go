package projectcli

import (
	"strings"
	"testing"
)

func TestParseProjectActionsAndClosedFlags(t *testing.T) {
	cases := []struct {
		name string
		args []string
		want string
		fail string
	}{
		{"check", []string{"check", "--repo", "."}, "check", ""},
		{"context", []string{"project", "context", "--repo", ".", "--manager", "engineering/Manager/root"}, "context", ""},
		{"repeat manager", []string{"plan", "--repo", ".", "--goal", "Cancel order", "--manager", "sales/Manager/orders", "--manager", "sales/Manager/inventory"}, "plan", ""},
		{"unknown action", []string{"migrate"}, "", "unknown project action"},
		{"unknown flag", []string{"check", "--repo", ".", "--write"}, "", "flag provided but not defined"},
		{"wrong flag", []string{"impact", "--repo", ".", "--manager", "sales/Manager/orders"}, "", "flag provided but not defined"},
		{"positional", []string{"check", "--repo", ".", "extra"}, "", "unexpected positional"},
		{"missing revision base", []string{"impact", "--repo", ".", "--base", "abc"}, "", "requires --revision"},
		{"run needs write", []string{"run", "--repo", ".", "--plan", "p1"}, "", "requires --write"},
		{"repair read-only status", []string{"repair", "--repo", ".", "--run", "r1"}, "repair", ""},
		{"repair explicit write", []string{"repair", "--repo", ".", "--run", "r1", "--write"}, "repair", ""},
		{"repair requires run", []string{"repair", "--repo", "."}, "", "requires --run"},
		{"edit apply needs reviewed digest", []string{"edit", "--repo", ".", "--input", "proposal.json", "--write"}, "", "requires --expect"},
		{"apply requires verified digest", []string{"apply", "--repo", ".", "--plan", "p1", "--run", "r1", "--candidate", "c1", "--branch", "feature/x", "--head", "a", "--worktree", "b", "--write"}, "", "requires --expect"},
		{"apply preview is read only", []string{"apply", "--repo", ".", "--plan", "p1", "--run", "r1", "--candidate", "c1"}, "apply", ""},
		{"apply write accepts exact preflight", []string{"apply", "--repo", ".", "--plan", "p1", "--run", "r1", "--candidate", "c1", "--branch", "feature/x", "--head", "abc", "--worktree", "sha256:tree", "--expect", "sha256:verify", "--write"}, "apply", ""},
		{"apply write requires preflight", []string{"apply", "--repo", ".", "--plan", "p1", "--run", "r1", "--candidate", "c1", "--expect", "verify", "--write"}, "", "from the read-only preflight"},
		{"help is available", []string{"init", "--help"}, "", ""},
		{"setup preview", []string{"setup", "--repo", ".", "--tool-root", "C:\\src\\Markitect", "--provider", "codex", "--model", "m", "--input-micros-per-million", "5", "--output-micros-per-million", "10", "--max-cost-micros", "1000"}, "setup", ""},
		{"setup write requires exact digest", []string{"setup", "--repo", ".", "--tool-root", "src", "--provider", "codex", "--model", "m", "--input-micros-per-million", "5", "--output-micros-per-million", "10", "--max-cost-micros", "1000", "--write"}, "", "requires --expect"},
		{"distill generation requires write", []string{"distill", "--repo", ".", "--discovery", ".markitect/drafts/d.json", "--generate", "--output", ".markitect/drafts/report.json", "--input-micros-per-million", "5", "--output-micros-per-million", "10", "--max-cost-micros", "1000"}, "", "requires --write"},
		{"distill generation accepted", []string{"distill", "--repo", ".", "--discovery", ".markitect/drafts/d.json", "--generate", "--write", "--output", ".markitect/drafts/report.json", "--input-micros-per-million", "5", "--output-micros-per-million", "10", "--max-cost-micros", "1000"}, "distill", ""},
		{"resolve requires all explicit transports", []string{"resolve", "--repo", "target", "--source-repo", "source", "--revision", strings.Repeat("a", 40), "--discovery", ".markitect/drafts/d.json", "--report", ".markitect/drafts/r.json", "--input", ".markitect/drafts/choices.json"}, "resolve", ""},
		{"plan accepts since baseline", []string{"plan", "--repo", ".", "--goal", "Cancel", "--since", "0123456789012345678901234567890123456789"}, "plan", ""},
		{"adoption record output", []string{"adopt", "--repo", ".", "--source-repo", ".", "--revision", strings.Repeat("a", 40), "--discovery", ".markitect/drafts/d.json", "--report", ".markitect/drafts/r.json", "--resolution", ".markitect/drafts/resolution.json", "--output", ".markitect/drafts/plan.json"}, "adopt", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, help, err := parse(tc.args, nil)
			if tc.fail != "" {
				if err == nil || !strings.Contains(err.Error(), tc.fail) {
					t.Fatalf("parse error = %v, want containing %q", err, tc.fail)
				}
				return
			}
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			if tc.name == "help is available" {
				if !help || got.action != "init" {
					t.Fatalf("help result = %#v, help=%v", got, help)
				}
				return
			}
			if got.action != tc.want {
				t.Fatalf("action = %q, want %q", got.action, tc.want)
			}
		})
	}
}

func TestParseWriteIsAnExplicitOperationFlag(t *testing.T) {
	for _, args := range [][]string{{"check", "--repo", ".", "--write"}} {
		if _, _, err := parse(args, nil); err == nil {
			t.Fatalf("accepted invalid write args %v", args)
		}
	}
	if opts, _, err := parse([]string{"init", "--repo", ".", "--name", "shop", "--write"}, nil); err != nil || !opts.write {
		t.Fatalf("explicit init write was rejected: %#v, %v", opts, err)
	}
	if opts, _, err := parse([]string{"adopt", "--repo", ".", "--source-repo", ".", "--revision", strings.Repeat("a", 40), "--discovery", ".markitect/drafts/d.json", "--report", ".markitect/drafts/r.json", "--resolution", ".markitect/drafts/resolution.json", "--plan", ".markitect/drafts/plan.json", "--expect", "sha256:plan", "--write"}, nil); err != nil || !opts.write || opts.plan != ".markitect/drafts/plan.json" {
		t.Fatalf("exact adoption apply was rejected: %#v, %v", opts, err)
	}
}

func TestParseProjectHelp(t *testing.T) {
	_, help, err := parse([]string{"project", "--help"}, nil)
	if err != nil || !help {
		t.Fatalf("project help = %v, err=%v", help, err)
	}
	_, help, err = parse([]string{"--help"}, nil)
	if err != nil || !help {
		t.Fatalf("top-level project help = %v, err=%v", help, err)
	}
	out := new(strings.Builder)
	if code := Run([]string{"--help"}, out, new(strings.Builder)); code != 0 || !strings.Contains(out.String(), "markitect project") {
		t.Fatalf("top-level project help code=%d output=%q", code, out.String())
	}
}

func TestParsePositiveRatesUsesBoundedCallerSuppliedEstimate(t *testing.T) {
	rates, err := parsePositiveRates("0", "125000", "2500000")
	if err != nil || rates.input != 0 || rates.output != 125000 || rates.maxCost != 2500000 {
		t.Fatalf("parsed rates = %#v, err=%v", rates, err)
	}
	for _, values := range [][]string{{"0", "0", "10"}, {"-1", "2", "10"}, {"1", "1", "0"}, {"1000000000001", "0", "10"}, {"1", "1", "1000000000001"}} {
		if _, err := parsePositiveRates(values[0], values[1], values[2]); err == nil {
			t.Errorf("accepted out-of-contract caller values %v", values)
		}
	}
}
