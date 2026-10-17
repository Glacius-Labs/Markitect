package gitrepo

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"reflect"
	"slices"

	"github.com/Glacius-Labs/Markitect/internal/apply"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

var _ apply.Publisher = (*Repository)(nil)

type publicationPlan struct {
	request  apply.PublicationRequest
	capture  publicationCapture
	changed  map[string]bool
	policies map[string]publicationPolicy
	index    string
}

func (r *Repository) Preflight(ctx context.Context, request apply.PublicationRequest, control apply.ControlCheck) (apply.PublicationPermit, error) {
	request.Changes = slices.Clone(request.Changes)
	if r == nil || control == nil || request.ID == "" || request.Run == "" {
		return apply.PublicationPermit{}, fmt.Errorf("%w: publication identity and controller required", ErrUnsafe)
	}
	ctx, cancel := context.WithTimeout(ctx, r.limits.Timeout)
	defer cancel()
	if err := control.Check(ctx); err != nil {
		return apply.PublicationPermit{}, err
	}
	capture, a, err := r.publicationSource(ctx, request)
	if err != nil {
		return apply.PublicationPermit{}, err
	}
	changed, err := publicationChanges(request.Changes, capture.census)
	if err != nil {
		return apply.PublicationPermit{}, err
	}
	policies, err := publicationPolicies(a.git, request.Changes)
	if err != nil {
		return apply.PublicationPermit{}, err
	}
	if err := publicationRepresentable(a, request.Changes, policies); err != nil {
		return apply.PublicationPermit{}, err
	}
	converter, err := newPublicationConverter(ctx, r.limits)
	if err != nil {
		return apply.PublicationPermit{}, err
	}
	defer converter.close()
	if len(request.Changes) > r.limits.Entries {
		return apply.PublicationPermit{}, ErrLimit
	}
	var total int64
	for _, change := range request.Changes {
		if after, ok := change.After(); ok {
			total += after.ByteLength()
			if total > r.limits.TotalBytes {
				return apply.PublicationPermit{}, ErrLimit
			}
			if _, err := converter.checkout(after, policies[change.Path().String()]); err != nil {
				return apply.PublicationPermit{}, err
			}
		}
	}
	index, err := publicationIndex(a.git, r.limits)
	if err != nil {
		return apply.PublicationPermit{}, publicationCaptureError(err)
	}
	if err := control.Check(ctx); err != nil {
		return apply.PublicationPermit{}, err
	}
	var nonce [32]byte
	if _, err := rand.Read(nonce[:]); err != nil {
		return apply.PublicationPermit{}, err
	}
	token := hex.EncodeToString(nonce[:])
	plan := publicationPlan{request, capture, changed, policies, index}
	r.publicationMu.Lock()
	if r.permits == nil {
		r.permits = make(map[string]publicationPlan)
	}
	r.permits[token] = plan
	r.publicationMu.Unlock()
	request.Changes = slices.Clone(request.Changes)
	return apply.PublicationPermit{Request: request, Token: token}, nil
}

func publicationChanges(changes []snapshot.Change, census snapshot.Tree) (map[string]bool, error) {
	changed := make(map[string]bool)
	entries := make(map[snapshot.Path]snapshot.Entry)
	for _, e := range census.Entries() {
		entries[e.Path()] = e
	}
	for _, c := range changes {
		p := c.Path()
		if p.String() == "" || technical(p.String()) || p.String() == ".markitect/state" || changed[p.String()] {
			return nil, fmt.Errorf("%w: duplicate or technical publication path", ErrUnsafe)
		}
		changed[p.String()] = true
		before, had := c.Before()
		after, has := c.After()
		actual, found := entries[p]
		if (!had && !has) || found != had || (had && actual != before) || (has && after.Path() != p) {
			return nil, fmt.Errorf("%w: publication before at %q", ErrChanged, p)
		}
		if has {
			entries[p] = after
		} else {
			delete(entries, p)
		}
	}
	var result []snapshot.Entry
	for _, e := range entries {
		result = append(result, e)
	}
	if _, err := snapshot.NewTree(result); err != nil {
		return nil, err
	}
	return changed, nil
}

func (r *Repository) Publish(ctx context.Context, permit apply.PublicationPermit, control apply.ControlCheck) (apply.PublicationReceipt, error) {
	request := permit.Request
	request.Changes = slices.Clone(request.Changes)
	fail := func(outcome apply.PublicationOutcome, err error) (apply.PublicationReceipt, error) {
		receipt, inspectErr := r.Inspect(ctx, request)
		receipt.Outcome, receipt.Reason = outcome, err.Error()
		return receipt, errors.Join(err, inspectErr)
	}
	if r == nil || control == nil {
		return fail(apply.PublicationFailed, fmt.Errorf("%w: repository and controller required", ErrUnsafe))
	}
	ctx, cancel := context.WithTimeout(ctx, r.limits.Timeout)
	defer cancel()
	r.publicationMu.Lock()
	plan, ok := r.permits[permit.Token]
	r.publicationMu.Unlock()
	if !ok || permit.Token == "" || !reflect.DeepEqual(plan.request, request) {
		return fail(apply.PublicationFailed, fmt.Errorf("%w: publication permit binding", ErrUnsafe))
	}
	request = plan.request
	if err := control.Check(ctx); err != nil {
		return fail(apply.PublicationStale, err)
	}
	_, a, err := r.publicationSource(ctx, request)
	if err != nil {
		return fail(apply.PublicationStale, err)
	}
	current, err := publicationPolicies(a.git, request.Changes)
	if err != nil {
		return fail(apply.PublicationStale, err)
	}
	if !reflect.DeepEqual(plan.policies, current) {
		return fail(apply.PublicationStale, publicationCaptureError(ErrChanged))
	}
	converter, err := newPublicationConverter(ctx, r.limits)
	if err != nil {
		return fail(apply.PublicationFailed, err)
	}
	defer converter.close()
	// All conversion and representability checks finish before the controller
	// fence immediately preceding the first possible owner-tree write.
	converted := make(map[string][]byte)
	for _, change := range request.Changes {
		if after, ok := change.After(); ok {
			data, err := converter.checkout(after, plan.policies[change.Path().String()])
			if err != nil {
				return fail(apply.PublicationFailed, err)
			}
			converted[change.Path().String()] = data
		}
	}
	if err := control.Check(ctx); err != nil {
		return fail(apply.PublicationStale, err)
	}
	if _, _, err := r.publicationSource(ctx, request); err != nil {
		return fail(apply.PublicationStale, err)
	}
	index, indexErr := publicationIndex(a.git, r.limits)
	if indexErr != nil {
		return fail(apply.PublicationStale, publicationCaptureError(indexErr))
	}
	if index != plan.index {
		return fail(apply.PublicationStale, publicationCaptureError(ErrChanged))
	}
	if err := control.Check(ctx); err != nil {
		return fail(apply.PublicationStale, err)
	}
	writes := 0
	for _, change := range request.Changes {
		if r.observe != nil {
			r.observe("publication-before-write", change.Path().String())
		}
		wrote, err := r.publicationWrite(ctx, a.git, change, converted[change.Path().String()])
		if wrote {
			writes++
		}
		if err != nil {
			receipt, inspectErr := r.Inspect(ctx, request)
			receipt.Outcome = apply.PublicationFailed
			if writes > 0 {
				receipt.Outcome = apply.PublicationUnknown
				before, after := 0, 0
				for _, p := range receipt.Paths {
					if p.State == apply.PathBefore {
						before++
					}
					if p.State == apply.PathAfter {
						after++
					}
				}
				if before > 0 && after > 0 {
					receipt.Outcome = apply.PublicationPartial
				}
				finalErr := errors.Join(control.Check(ctx), r.publicationFinal(ctx, plan))
				if finalErr != nil {
					receipt.Outcome = apply.PublicationUnknown
					err = errors.Join(err, finalErr)
				}
			}
			if inspectErr != nil {
				receipt.Outcome = apply.PublicationUnknown
			}
			for _, p := range receipt.Paths {
				if p.State == apply.PathOther {
					receipt.Outcome = apply.PublicationUnknown
				}
			}
			receipt.Reason = err.Error()
			return receipt, errors.Join(err, inspectErr)
		}
	}
	if r.observe != nil {
		r.observe("publication-after-write", request.Repository)
	}
	// Both final checks run even if one fails. Any loss of this final fence is
	// unknown, including a publication whose requested paths all look correct.
	finalErr := errors.Join(control.Check(ctx), r.publicationFinal(ctx, plan))
	receipt, inspectErr := r.Inspect(ctx, request)
	if finalErr != nil || inspectErr != nil {
		receipt.Outcome = apply.PublicationUnknown
		receipt.Reason = errors.Join(finalErr, inspectErr).Error()
		return receipt, errors.Join(finalErr, inspectErr)
	}
	for _, p := range receipt.Paths {
		if p.State != apply.PathAfter {
			receipt.Outcome = apply.PublicationUnknown
			receipt.Reason = "publication readback differs"
			return receipt, ErrChanged
		}
	}
	receipt.Outcome = apply.PublicationApplied
	receipt.Reason = ""
	if len(request.Changes) == 0 {
		receipt.Outcome = apply.PublicationNoChange
	}
	return receipt, nil
}
