package gitrepo

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"slices"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/apply"
	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

type publicationPolicy struct{ autocrlf, eol, text, pathEOL, filemode string }

func publicationConfig(g *gitRepository, key, fallback string) (string, error) {
	out, err := g.run("config", "--get", key)
	if err != nil {
		var exit *exec.ExitError
		if errors.As(err, &exit) && exit.ExitCode() == 1 {
			return fallback, nil
		}
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func publicationPolicies(g *gitRepository, changes []snapshot.Change) (map[string]publicationPolicy, error) {
	result := make(map[string]publicationPolicy)
	auto, err := publicationConfig(g, "core.autocrlf", "false")
	if err != nil {
		return nil, err
	}
	eol, err := publicationConfig(g, "core.eol", "native")
	if err != nil {
		return nil, err
	}
	mode, err := publicationConfig(g, "core.filemode", "true")
	if err != nil {
		return nil, err
	}
	if (auto != "false" && auto != "true" && auto != "input") || (eol != "native" && eol != "lf" && eol != "crlf") || (mode != "true" && mode != "false") {
		return nil, fmt.Errorf("%w: unsupported Git checkout configuration", ErrUnrepresentable)
	}
	for _, c := range changes {
		p := c.Path().String()
		if p == "" || technical(p) {
			return nil, ErrUnsafe
		}
		out, err := g.run("check-attr", "-z", "--all", "--", p)
		if err != nil {
			return nil, err
		}
		fields := strings.Split(strings.TrimSuffix(string(out), "\x00"), "\x00")
		policy := publicationPolicy{autocrlf: auto, eol: eol, filemode: mode}
		if len(out) > 0 {
			if len(fields)%3 != 0 {
				return nil, ErrUnsafe
			}
			for i := 0; i < len(fields); i += 3 {
				key, value := fields[i+1], fields[i+2]
				switch key {
				case "filter", "working-tree-encoding", "ident":
					if value != "unset" && value != "unspecified" {
						return nil, fmt.Errorf("%w: unsupported checkout attribute %s at %q", ErrUnrepresentable, key, p)
					}
				case "text":
					if value != "set" && value != "unset" && value != "auto" && value != "unspecified" {
						return nil, ErrUnrepresentable
					}
					policy.text = value
				case "eol":
					if value != "lf" && value != "crlf" && value != "unset" && value != "unspecified" {
						return nil, ErrUnrepresentable
					}
					policy.pathEOL = value
				case "crlf":
					// Legacy crlf is equivalent to text but mixed declarations have
					// precedence subtleties. Refuse rather than approximate Git's choice.
					if value != "unspecified" {
						return nil, fmt.Errorf("%w: legacy crlf attribute", ErrUnrepresentable)
					}
				}
			}
		}
		result[p] = policy
	}
	return result, nil
}

func publicationRepresentable(a *acquisition, changes []snapshot.Change, policies map[string]publicationPolicy) error {
	entries := make(map[snapshot.Path]snapshot.Entry)
	for _, e := range a.raw.Census.Entries() {
		entries[e.Path()] = e
	}
	for _, c := range changes {
		attributeKey, _ := model.NameCollisionKey(path.Base(c.Path().String()))
		if attributeKey == ".gitattributes" {
			return fmt.Errorf("%w: changing checkout attributes in the same publication", ErrUnrepresentable)
		}
		if runtime.GOOS == "windows" {
			if !filepath.IsLocal(filepath.FromSlash(c.Path().String())) {
				return fmt.Errorf("%w: %w: native publication path %q", apply.ErrUnpublishable, ErrUnrepresentable, c.Path())
			}
			for _, part := range strings.Split(c.Path().String(), "/") {
				if strings.ContainsAny(part, `<>:"|?*`) || strings.HasSuffix(part, ".") || strings.HasSuffix(part, " ") {
					return fmt.Errorf("%w: %w: native publication name %q", apply.ErrUnpublishable, ErrUnrepresentable, part)
				}
			}
		}
		before, had := c.Before()
		after, has := c.After()
		policy := policies[c.Path().String()]
		if had && bytes.Contains(before.Bytes(), []byte("\r\n")) && policy.text != "set" && policy.text != "unset" && (policy.autocrlf != "false" || policy.text == "auto") {
			return fmt.Errorf("%w: automatic text conversion with a CRLF index entry", ErrUnrepresentable)
		}
		if has && runtime.GOOS == "windows" && policies[c.Path().String()].filemode == "true" && after.Mode() == snapshot.Executable {
			return fmt.Errorf("%w: %w: executable mode on this filesystem", apply.ErrUnpublishable, ErrUnrepresentable)
		}
		if has && policies[c.Path().String()].filemode == "false" && ((!had && after.Mode() == snapshot.Executable) || (had && before.Mode() != after.Mode())) {
			return fmt.Errorf("%w: %w: repository does not track requested mode at %q", apply.ErrUnpublishable, ErrUnrepresentable, c.Path())
		}
		if has {
			entries[c.Path()] = after
		} else {
			delete(entries, c.Path())
		}
	}
	d, err := openDisk(a.git.ctx, a.git.path, a.repository.limits, nil)
	if err != nil {
		return err
	}
	defer d.close()
	for _, c := range changes {
		root, name, err := publicationParent(d, c.Path().String(), false)
		if errors.Is(err, os.ErrNotExist) && c.Kind() == snapshot.Add {
			continue
		}
		if err != nil {
			return err
		}
		names, err := directoryNames(root, d.limits.Entries)
		if err != nil {
			return err
		}
		if c.Kind() == snapshot.Add {
			if _, err := root.Lstat(name); err == nil {
				return fmt.Errorf("%w: %w: native publication alias at %q", apply.ErrUnpublishable, ErrUnrepresentable, c.Path())
			} else if !errors.Is(err, os.ErrNotExist) {
				return err
			}
			want, _ := model.NameCollisionKey(name)
			for _, other := range names {
				key, ok := model.NameCollisionKey(other)
				if !ok {
					return ErrUnrepresentable
				}
				if key == want {
					return fmt.Errorf("%w: %w: publication target name exists or aliases at %q", apply.ErrUnpublishable, ErrUnrepresentable, c.Path())
				}
			}
		}
	}
	keys := make(map[string]string)
	for _, e := range entries {
		parts := strings.Split(e.Path().String(), "/")
		for i := range parts {
			p := strings.Join(parts[:i+1], "/")
			key, valid := model.NameCollisionKey(p)
			if !valid {
				return ErrUnrepresentable
			}
			if previous, found := keys[key]; found && previous != p {
				return fmt.Errorf("%w: %w: publication case or Unicode alias %q / %q", apply.ErrUnpublishable, ErrUnrepresentable, previous, p)
			}
			keys[key] = p
		}
	}
	// A case-only rename cannot safely be staged through the owner's tree on
	// every supported filesystem, even if its final tree has no alias.
	for _, c := range changes {
		key, _ := model.NameCollisionKey(c.Path().String())
		for _, old := range a.raw.Census.Entries() {
			oldKey, _ := model.NameCollisionKey(old.Path().String())
			if key == oldKey && c.Path() != old.Path() {
				return fmt.Errorf("%w: %w: case-only publication rename", apply.ErrUnpublishable, ErrUnrepresentable)
			}
		}
	}
	return nil
}

func (r *Repository) Inspect(ctx context.Context, request apply.PublicationRequest) (apply.PublicationReceipt, error) {
	request.Changes = slices.Clone(request.Changes)
	receipt := apply.PublicationReceipt{ID: request.ID, Outcome: apply.PublicationUnknown, Reason: "inspection classifies paths; it cannot prove publication or its final controller fence"}
	for _, c := range request.Changes {
		receipt.Paths = append(receipt.Paths, apply.PublishedPath{Path: c.Path(), State: apply.PathOther})
	}
	if r == nil {
		return receipt, ErrUnsafe
	}
	ctx, cancel := context.WithTimeout(ctx, r.limits.Timeout)
	defer cancel()
	g, err := identifyGit(ctx, request.Repository)
	if err != nil {
		return receipt, err
	}
	d, err := openDisk(ctx, g.path, r.limits, nil)
	if err != nil {
		return receipt, err
	}
	defer d.close()
	policies, policyErr := publicationPolicies(g, request.Changes)
	converter, err := newPublicationConverter(ctx, r.limits)
	if err != nil {
		return receipt, errors.Join(policyErr, err)
	}
	defer converter.close()
	var failures []error
	failures = append(failures, policyErr)
	indexed := &acquisition{ctx: ctx, git: g, repository: r}
	indexErr := indexed.loadIndexModes()
	failures = append(failures, indexErr)
	for i, c := range request.Changes {
		data, mode, found, readErr := publicationRead(d, c.Path())
		if readErr != nil || indexErr != nil {
			failures = append(failures, readErr)
			continue
		}
		matches := func(entry snapshot.Entry, present bool) bool {
			if !present {
				return !found
			}
			if !found || policyErr != nil {
				return false
			}
			policy := policies[c.Path().String()]
			if policy.filemode == "false" {
				// Git ignores physical executable bits; the original index mode remains
				// the only representable mode for an existing path.
				if current, ok := indexed.indexModes[c.Path().String()]; ok {
					mode = current
				}
			}
			if mode != entry.Mode() {
				return false
			}
			clean, err := converter.clean(data, policy)
			if err != nil {
				failures = append(failures, err)
				return false
			}
			return bytes.Equal(clean, entry.Bytes())
		}
		after, has := c.After()
		before, had := c.Before()
		if matches(after, has) {
			receipt.Paths[i].State = apply.PathAfter
		} else if (had && found && policyErr == nil && mode == before.Mode() && bytes.Equal(data, before.Bytes())) || matches(before, had) {
			receipt.Paths[i].State = apply.PathBefore
		}
	}
	if err := d.confirm(); err != nil {
		failures = append(failures, err)
	}
	// Stable order is the journaled request order, including on errors.
	if slices.ContainsFunc(receipt.Paths, func(p apply.PublishedPath) bool { return p.State == apply.PathOther }) {
		failures = append(failures, fmt.Errorf("publication path is neither before nor after"))
	}
	return receipt, errors.Join(failures...)
}

func publicationRead(d *diskBoundary, p snapshot.Path) ([]byte, snapshot.Mode, bool, error) {
	// A missing parent means a deleted/new path is absent. Existing parents
	// retain the ordinary exact-name and redirect checks.
	if p.String() == "" || technical(p.String()) {
		return nil, 0, false, ErrUnsafe
	}
	root, name, err := publicationParent(d, p.String(), false)
	if errors.Is(err, os.ErrNotExist) {
		return nil, 0, false, nil
	}
	if err != nil {
		return nil, 0, false, err
	}
	names, err := directoryNames(root, d.limits.Entries)
	if err != nil {
		return nil, 0, false, err
	}
	if !slices.Contains(names, name) {
		if _, err := root.Lstat(name); !errors.Is(err, os.ErrNotExist) {
			return nil, 0, false, ErrUnsafe
		}
	}
	return d.read(p.String(), true)
}
