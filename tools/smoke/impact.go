package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"slices"
	"strings"
)

// version checks that the built CLI names its version and build platform.
func (s *smoke) version() error {
	out, err := s.run(s.repo, nil, s.tool("markitect"), "version")
	if err != nil {
		return err
	}
	pattern := regexp.MustCompile(`^Markitect \S+ \(` + regexp.QuoteMeta(runtime.GOOS+"/"+runtime.GOARCH) + `\)$`)
	if !pattern.MatchString(strings.TrimSpace(out)) {
		return fmt.Errorf("unexpected version output %q", out)
	}
	return nil
}

const (
	policyFile = ".markitect/model/commerce/sales/orders/cancel-before-shipped.yaml"
	policyOld  = "a shipped order cannot be cancelled."
	policyNew  = "a shipped or delivered order cannot be cancelled."
)

var (
	editedStatement     = `["project.markitect.example.org/v1alpha1","Statement","commerce.sales.orders","cancel-before-shipped"]`
	ordersManager       = `["project.markitect.example.org/v1alpha1","Manager","commerce.sales.orders","orders"]`
	cancellationCheck   = `["project.markitect.example.org/v1alpha1","Check","engineering","cancellation-tests"]`
	policyAffectedFiles = []string{"docs/cancellation.md", "src/shop/orders/"}
)

type impactReport struct {
	Digest             string
	ChangedDefinitions []string
	Managers           []string
	Checks             []string
	Files              []string
}

// policyEditImpact replays a fixed policy edit on a committed copy of the
// project-world example: the edited model still checks, and impact between
// the two full commit IDs names exactly the edited Statement, routes its
// Manager, Check and files, repeats byte for byte, and is empty for an
// unchanged pair.
func (s *smoke) policyEditImpact() error {
	repo, err := s.gitRepo("policy-edit", "feature/policy-edit")
	if err != nil {
		return err
	}
	if err := os.CopyFS(repo, os.DirFS(filepath.Join(s.repo, "examples", "project-world"))); err != nil {
		return err
	}
	base, err := s.commit(repo, "Freeze the project-world fixture")
	if err != nil {
		return err
	}
	path := filepath.Join(repo, filepath.FromSlash(policyFile))
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if !strings.Contains(string(data), policyOld) {
		return fmt.Errorf("the fixed policy wording changed in %s; review this replay", policyFile)
	}
	if err := os.WriteFile(path, []byte(strings.Replace(string(data), policyOld, policyNew, 1)), 0o644); err != nil {
		return err
	}
	cli := func(args ...string) (string, error) { return s.run(repo, nil, s.tool("markitect"), args...) }
	if err := s.checkSucceeds(func() (string, error) { return cli("check", "--repo", repo) }); err != nil {
		return err
	}
	candidate, err := s.commit(repo, "Tighten the cancellation policy")
	if err != nil {
		return err
	}
	impact := func(since, revision string) (impactReport, error) {
		var report impactReport
		out, err := cli("impact", "--repo", repo, "--since", since, "--revision", revision)
		if err == nil {
			err = decode(out, &report)
		}
		return report, err
	}
	first, err := impact(base, candidate)
	if err != nil {
		return err
	}
	if !slices.Equal(first.ChangedDefinitions, []string{editedStatement}) {
		return fmt.Errorf("changed definitions %v, want only the edited Statement", first.ChangedDefinitions)
	}
	if !slices.Contains(first.Managers, ordersManager) || !slices.Contains(first.Checks, cancellationCheck) {
		return fmt.Errorf("impact does not route the orders Manager and the cancellation check: managers %v, checks %v", first.Managers, first.Checks)
	}
	for _, file := range policyAffectedFiles {
		if !slices.Contains(first.Files, file) {
			return fmt.Errorf("impact files %v lack %s", first.Files, file)
		}
	}
	again, err := impact(base, candidate)
	if err != nil {
		return err
	}
	if again.Digest != first.Digest || !isDigest(strings.TrimPrefix(first.Digest, "sha256:")) {
		return fmt.Errorf("impact is not deterministic: %s then %s", first.Digest, again.Digest)
	}
	unchanged, err := impact(candidate, candidate)
	if err != nil {
		return err
	}
	if len(unchanged.ChangedDefinitions) != 0 {
		return fmt.Errorf("an unchanged revision pair reports changed definitions %v", unchanged.ChangedDefinitions)
	}
	return nil
}

// commit stages everything in repo, commits it and returns the full commit ID.
func (s *smoke) commit(repo, message string) (string, error) {
	for _, args := range [][]string{{"add", "--all"}, {"commit", "--quiet", "-m", message}} {
		if _, err := s.run(repo, nil, "git", args...); err != nil {
			return "", err
		}
	}
	out, err := s.run(repo, nil, "git", "rev-parse", "HEAD")
	return strings.TrimSpace(out), err
}
