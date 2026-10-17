package run

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"

	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/organization"
)

// binding decodes the recorded versioned basis, not a new preparation. Original
// compiler/configuration results are historical facts. Recompiling with today's
// ProductBuild could change their meaning and does not belong in restoration.
// Only the closed recorded projection and its reducer inputs are reconstructed;
// no organization settings resolution or source acquisition is repeated.
func (r *restoreReader) binding(ref BlobRef) (Binding, error) {
	if ref.MediaType != bindingMediaType || ref.Encoding != "utf-8" {
		return Binding{}, ErrInvalid
	}
	p, err := r.blob(ref)
	if err != nil {
		return Binding{}, err
	}
	var w bindingWire
	if err := strictJSON(p.Bytes(), &w); err != nil || w.Version != 1 || !w.Source.valid() {
		return Binding{}, ErrInvalid
	}
	// Rebuild topology through its producer from the stored compiled inventory
	// and local identities. This validates ancestry/order and gives the same
	// structural review floor without interpreting the original model anew.
	compiled := model.Result{Status: model.Passed}
	for _, scope := range w.Compilation {
		if len(scope.Canonical) == 0 || scope.Digest != fmt.Sprintf("sha256:%x", sha256.Sum256(scope.Canonical)) {
			return Binding{}, ErrInvalid
		}
		m := model.ModelResult{ID: scope.ID, Digest: scope.Digest, Local: scope.Local, Status: model.Passed, Packages: scope.Packages}
		if scope.Local {
			for _, manager := range w.Managers {
				for _, id := range manager.Definitions {
					m.Definitions = append(m.Definitions, model.Definition{Identity: id, ModelID: scope.ID})
				}
			}
		}
		compiled.Models = append(compiled.Models, m)
	}
	topology, err := organization.BuildTopology(compiled)
	if err != nil {
		return Binding{}, ErrInvalid
	}
	actual, _ := json.Marshal(topology.Managers())
	recorded, _ := json.Marshal(w.Managers)
	if !bytes.Equal(actual, recorded) {
		return Binding{}, ErrInvalid
	}
	reviews := reviewFloor(topology.Managers())
	if err := w.Policy.validate(reviews); err != nil {
		return Binding{}, err
	}
	owner, err := r.blob(w.OwnerRequest)
	if err != nil {
		return Binding{}, err
	}
	return Binding{payload: p, owner: owner, source: w.Source, policy: w.Policy, reviews: reviews}, nil
}
