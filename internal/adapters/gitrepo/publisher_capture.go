package gitrepo

import (
	"context"
	"fmt"
	"os"
	"reflect"
	"slices"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/apply"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

// Capture state belongs to this adapter instance. No caller-owned mutable
// selection or raw model bytes are retained.
type publicationCapture struct {
	selection                     apply.InputSelection
	census                        snapshot.Tree
	head                          string
	external                      string
	rootInfo, gitInfo, commonInfo os.FileInfo
}

func (r *Repository) rememberPublicationCapture(selection apply.InputSelection, a *acquisition) {
	selection.Dependencies = slices.Clone(selection.Dependencies)
	selection.Execution = slices.Clone(selection.Execution)
	selection.Registry = slices.Clone(selection.Registry)
	selection.Ignore = slices.Clone(selection.Ignore)
	selection.HostProfiles = slices.Clone(selection.HostProfiles)
	selection.Instructions = slices.Clone(selection.Instructions)
	r.publicationMu.Lock()
	defer r.publicationMu.Unlock()
	if r.captures == nil {
		r.captures = make(map[apply.PublicationGuard]publicationCapture)
	}
	r.captures[a.guard()] = publicationCapture{selection: selection, census: a.raw.Census, head: a.head, external: publicationExternal(a), rootInfo: a.git.rootInfo, gitInfo: a.git.gitInfo, commonInfo: a.git.commonInfo}
}

func publicationExternal(a *acquisition) string {
	// Reuse the capture encoding, retaining only external sources. In-repository
	// inputs are covered by the census and its exact outside-change comparison.
	copy := *a
	copy.raw = a.raw
	copy.raw.Census = snapshot.Tree{}
	copy.models = nil
	for _, m := range a.models {
		if m.target.RepositoryArea.String() == "" {
			copy.models = append(copy.models, m)
		}
	}
	external := func(files []apply.SourceFile) []apply.SourceFile {
		var result []apply.SourceFile
		for _, f := range files {
			if f.RepositoryPath.String() == "" {
				result = append(result, f)
			}
		}
		return result
	}
	copy.raw.Execution = external(a.raw.Execution)
	copy.raw.Registry = external(a.raw.Registry)
	copy.raw.Ignore = external(a.raw.Ignore)
	copy.raw.HostProfiles = external(a.raw.HostProfiles)
	copy.raw.Instructions = external(a.raw.Instructions)
	return string(copy.guard())
}

func (r *Repository) publicationSource(ctx context.Context, request apply.PublicationRequest) (publicationCapture, *acquisition, error) {
	r.publicationMu.Lock()
	capture, ok := r.captures[request.Guard]
	r.publicationMu.Unlock()
	if !ok || request.Guard == "" || request.Repository != capture.selection.Repository {
		return capture, nil, fmt.Errorf("%w: publication requires this adapter's original capture and repository", ErrUnsafe)
	}
	a, err := r.acquire(ctx, capture.selection, "", true)
	if err != nil {
		return capture, a, publicationCaptureError(err)
	}
	if a.guard() != request.Guard {
		return capture, a, publicationCaptureError(ErrChanged)
	}
	if !os.SameFile(capture.rootInfo, a.git.rootInfo) || !os.SameFile(capture.gitInfo, a.git.gitInfo) || !os.SameFile(capture.commonInfo, a.git.commonInfo) {
		return capture, a, publicationCaptureError(ErrChanged)
	}
	a.git.ctx = ctx
	return capture, a, nil
}

func (r *Repository) publicationFinal(ctx context.Context, plan publicationPlan) error {
	a, err := r.acquire(ctx, plan.capture.selection, "", false)
	if err != nil {
		return publicationCaptureError(err)
	}
	a.git.ctx = ctx
	if !os.SameFile(plan.capture.rootInfo, a.git.rootInfo) || !os.SameFile(plan.capture.gitInfo, a.git.gitInfo) || !os.SameFile(plan.capture.commonInfo, a.git.commonInfo) {
		return publicationCaptureError(ErrChanged)
	}
	if a.head != plan.capture.head || publicationExternal(a) != plan.capture.external {
		return publicationCaptureError(ErrChanged)
	}
	outside := func(tree snapshot.Tree) []snapshot.Entry {
		var result []snapshot.Entry
		for _, e := range tree.Entries() {
			if !plan.changed[e.Path().String()] {
				result = append(result, e)
			}
		}
		return result
	}
	if !slices.Equal(outside(a.raw.Census), outside(plan.capture.census)) {
		return publicationCaptureError(fmt.Errorf("%w: outside publication paths", ErrChanged))
	}
	index, err := publicationIndex(a.git, r.limits)
	if err != nil {
		return publicationCaptureError(err)
	}
	if index != plan.index {
		return publicationCaptureError(fmt.Errorf("%w: publication index", ErrChanged))
	}
	policies, err := publicationPolicies(a.git, plan.request.Changes)
	if err != nil {
		return err
	}
	if !reflect.DeepEqual(policies, plan.policies) {
		return publicationCaptureError(fmt.Errorf("%w: Git checkout policy", ErrChanged))
	}
	return nil
}

// Only positively observed drift gets the consumer's capture classification.
// Acquisition can join drift with an unsafe path or an unknown I/O failure;
// those incomplete observations must retain their original error meaning.
func publicationCaptureError(err error) error {
	if !publicationOnlyDrift(err) {
		return err
	}
	return fmt.Errorf("%w: %w", apply.ErrCaptureChanged, err)
}

func publicationOnlyDrift(err error) bool {
	if err == ErrChanged || err == ErrDirty {
		return true
	}
	switch err := err.(type) {
	case interface{ Unwrap() error }:
		return publicationOnlyDrift(err.Unwrap())
	case interface{ Unwrap() []error }:
		children := err.Unwrap()
		if len(children) == 0 {
			return false
		}
		for _, child := range children {
			if !publicationOnlyDrift(child) {
				return false
			}
		}
		return true
	default:
		return false
	}
}

func publicationIndex(g *gitRepository, limits Limits) (string, error) {
	shared, err := g.run("rev-parse", "--shared-index-path")
	if err != nil {
		return "", err
	}
	if len(strings.TrimSpace(string(shared))) != 0 {
		return "", fmt.Errorf("%w: split Git index", ErrUnrepresentable)
	}
	root, err := openDisk(g.ctx, g.gitDir, limits, nil)
	if err != nil {
		return "", err
	}
	defer root.close()
	data, _, found, err := root.read("index", true)
	if err != nil {
		return "", err
	}
	if !found {
		return "", ErrChanged
	}
	if err := root.confirm(); err != nil {
		return "", err
	}
	return string(data), nil
}
