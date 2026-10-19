package main

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

func (s *smoke) version() error {
	out, err := s.run(s.repo, nil, s.tool("markitect"), "version")
	if err != nil {
		return err
	}
	var report struct{ Version string }
	if err := decode(out, &report); err != nil {
		return err
	}
	if report.Version == "" {
		return fmt.Errorf("empty build version")
	}
	return nil
}

// libraryPreparation consumes the unchanged normative Library sources through
// the packaged product. It creates no implementation, provider or Apply record.
// The only M2 change is the specified five-to-three model edit in a temp repo.
func (s *smoke) libraryPreparation() error {
	repo, err := s.gitRepo("library-preparation", "codex/source-smoke")
	if err != nil {
		return err
	}
	base := filepath.Join(s.repo, "docs", "design", "canonical-model")
	for _, m := range []struct{ source, target string }{
		{filepath.Join(base, "library-example", "m1"), filepath.Join(repo, ".markitect")},
		{filepath.Join(base, "library-example", "vocabulary"), filepath.Join(repo, ".markitect", "dependencies", "software")},
		{filepath.Join(base, "minimal-language", "exchange-example", "dependencies", "core"), filepath.Join(repo, ".markitect", "dependencies", "core")},
	} {
		if err := copyModel(m.source, m.target); err != nil {
			return err
		}
	}
	document, err := os.ReadFile(filepath.Join(base, "manager-apply-configuration.md"))
	if err != nil {
		return err
	}
	_, text, found := strings.Cut(string(document), "```yaml\n")
	if !found { // A Windows checkout may retain CRLF; preserve YAML bytes.
		_, text, found = strings.Cut(string(document), "```yaml\r\n")
	}
	if !found {
		return fmt.Errorf("documented execution fixture missing")
	}
	execution, _, found := strings.Cut(text, "```")
	if !found {
		return fmt.Errorf("execution fixture is unclosed")
	}
	if err := os.WriteFile(filepath.Join(repo, ".markitect", "execution.yaml"), []byte(execution), 0644); err != nil {
		return err
	}
	if _, err := s.run(repo, nil, "git", "add", "."); err != nil {
		return err
	}
	if _, err := s.run(repo, nil, "git", "commit", "--quiet", "--no-verify", "-m", "Unchanged normative Library M1 inputs"); err != nil {
		return err
	}
	common := []string{"--repo", repo, "--dependency", ".markitect/dependencies/core/model.yaml", "--dependency", ".markitect/dependencies/software/model.yaml"}
	invoke := func(verb string, extra ...string) (string, error) {
		args := append([]string{verb}, common...)
		args = append(args, extra...)
		return s.run(repo, nil, s.tool("markitect"), args...)
	}
	check := func(extra ...string) (string, error) {
		out, err := invoke("check", extra...)
		if err != nil {
			return "", err
		}
		var r struct {
			Status string
			Stages struct {
				Structural struct {
					Model struct {
						Models []struct {
							Local                bool
							CanonicalModelDigest string
						}
					}
				}
			}
		}
		if err := decode(out, &r); err != nil {
			return "", err
		}
		if r.Status != "passed" {
			return "", fmt.Errorf("Library preparation: %s", tail(out))
		}
		for _, m := range r.Stages.Structural.Model.Models {
			if m.Local && m.CanonicalModelDigest != "" {
				return m.CanonicalModelDigest, nil
			}
		}
		return "", fmt.Errorf("missing actual local canonical digest")
	}
	m1, err := check()
	if err != nil {
		return err
	}
	fixed, err := check("--revision", "HEAD")
	if err != nil {
		return err
	}
	if fixed != m1 {
		return fmt.Errorf("fixed M1 differs from working M1")
	}
	policy := filepath.Join(repo, ".markitect", "model", "lending", "loan-policy.yaml")
	bytes, err := os.ReadFile(policy)
	if err != nil {
		return err
	}
	if strings.Count(string(bytes), "maxOpenLoans: 5") != 1 {
		return fmt.Errorf("normative M1 loan policy changed")
	}
	if err := os.WriteFile(policy, []byte(strings.Replace(string(bytes), "maxOpenLoans: 5", "maxOpenLoans: 3", 1)), 0644); err != nil {
		return err
	}
	m2, err := check()
	if err != nil {
		return err
	}
	if m2 == m1 {
		return fmt.Errorf("M2 change did not alter canonical digest")
	}
	fixed, err = check("--revision", "HEAD")
	if err != nil {
		return err
	}
	if fixed != m1 {
		return fmt.Errorf("fixed M1 fell back to changed working M2")
	}
	contextArgs := append([]string{"context", "lending"}, common...)
	out, err := s.run(repo, nil, s.tool("markitect"), contextArgs...)
	if err != nil {
		return err
	}
	var context struct {
		Status   string
		Contexts []struct {
			Definitions []struct {
				Definition struct {
					Identity struct{ ID string }
					Spec     map[string]any
				}
				Reasons []any
			}
		}
	}
	if err := decode(out, &context); err != nil {
		return err
	}
	policyFound := false
	for _, c := range context.Contexts {
		for _, d := range c.Definitions {
			if d.Definition.Identity.ID == "loan-policy" {
				v, ok := d.Definition.Spec["maxOpenLoans"].(map[string]any)
				policyFound = ok && v["integer"] == "3" && len(d.Reasons) > 0
			}
		}
	}
	if context.Status != "passed" || !policyFound {
		return fmt.Errorf("actual M2 context lacks its policy or inclusion reasons")
	}
	out, err = invoke("docs")
	if err != nil {
		return err
	}
	var docs struct {
		View struct{ Status, ContentDigest, PreviewDigest, CanonicalModelDigest string }
	}
	if err := decode(out, &docs); err != nil {
		return err
	}
	if docs.View.Status != "passed" || docs.View.CanonicalModelDigest != m2 || docs.View.PreviewDigest == docs.View.ContentDigest || docs.View.ContentDigest == m2 {
		return fmt.Errorf("documentation conflates source, output or model identities")
	}
	host := filepath.Join(s.work, "explicit-host.yaml")
	if err := os.WriteFile(host, []byte("apiVersion: markitect.host/v1\nagentProfiles: {}\ncheckProfiles: {}\n"), 0644); err != nil {
		return err
	}
	out, err = invoke("check", "--metadata", "--revision", "HEAD", "--host-profile", host)
	if err == nil || !strings.Contains(out, "no revision binding for external boundary") {
		return fmt.Errorf("fixed preview accepted current external Host bytes: %s", tail(out))
	}
	out, err = s.run(repo, nil, s.tool("markitect"), "apply", "--repo", filepath.Join(s.work, "never-created"))
	if err == nil || !strings.Contains(out, "not_completed") || !strings.Contains(out, "unavailable") {
		return fmt.Errorf("Apply did not fail explicitly before acquisition: %s", tail(out))
	}
	if err := requireAbsent(s.work, "never-created"); err != nil {
		return err
	}
	// A configuration of the removed sequential Apply (formatVersion 1) is refused
	// as failed before the repository is acquired, the lock is taken or any
	// container engine or provider is started.
	oldConfig := filepath.Join(s.work, "old-run.json")
	state := filepath.Join(s.work, "never-created-state")
	if err := os.WriteFile(oldConfig, []byte(fmt.Sprintf(`{"formatVersion":1,"stateDirectory":%q,"allocation":{"runs":1}}`, state)), 0644); err != nil {
		return err
	}
	out, err = s.run(repo, nil, s.tool("markitect"), "apply", "--repo", filepath.Join(s.work, "never-created"), "--host-profile", host, "--run-config", oldConfig)
	if err == nil || !strings.Contains(out, `"status": "failed"`) || !strings.Contains(out, "formatVersion 2") {
		return fmt.Errorf("Apply accepted an invalid run configuration: %s", tail(out))
	}
	for _, name := range []string{"never-created", "never-created-state"} {
		if err := requireAbsent(s.work, name); err != nil {
			return err
		}
	}
	return nil
}

func copyModel(source, target string) error {
	return filepath.WalkDir(source, func(name string, e fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		relative, err := filepath.Rel(source, name)
		if err != nil {
			return err
		}
		if e.IsDir() {
			return nil
		}
		if e.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("redirected normative fixture: %s", name)
		}
		relative = filepath.ToSlash(relative)
		if relative != "model.yaml" && relative != "model-lock.yaml" && !strings.HasPrefix(relative, "model/") {
			return nil
		}
		data, err := os.ReadFile(name)
		if err != nil {
			return err
		}
		to := filepath.Join(target, filepath.FromSlash(relative))
		if err := os.MkdirAll(filepath.Dir(to), 0755); err != nil {
			return err
		}
		return os.WriteFile(to, data, 0644)
	})
}
