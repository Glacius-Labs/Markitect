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
	p, err := r.blob(ref)
	if err != nil {
		return Binding{}, err
	}
	w, reviews, err := decodeBindingRecord(p)
	if err != nil {
		return Binding{}, err
	}
	owner, err := r.blob(w.OwnerRequest)
	if err != nil {
		return Binding{}, err
	}
	return Binding{payload: p, owner: owner, source: w.Source, policy: w.Policy, reviews: reviews}, nil
}

// BindingOwnerRef validates the original basis payload's byte identity and
// closed recorded format, then returns its required owner-input reference. It
// performs no I/O. A Journal adapter uses this alongside AppendBatch.BindingRefs
// and normal Payload identity checks, without decoding private binding fields
// or implementing another reducer. This does not prove the owner blob exists,
// is durable or grants authority; the store must verify and retain its bytes.
func BindingOwnerRef(payload Payload) (BlobRef, error) {
	w, _, err := decodeBindingRecord(payload)
	if err != nil {
		return BlobRef{}, err
	}
	return w.OwnerRequest, nil
}

func decodeBindingRecord(p Payload) (bindingWire, int64, error) {
	if !p.Valid() || p.Ref().MediaType != bindingMediaType || p.Ref().Encoding != "utf-8" {
		return bindingWire{}, 0, ErrInvalid
	}
	var w bindingWire
	if err := strictJSON(p.Bytes(), &w); err != nil || w.Version != 1 || !w.Source.valid() || !w.OwnerRequest.Valid() {
		return bindingWire{}, 0, ErrInvalid
	}
	// Rebuild topology through its producer from the stored compiled inventory
	// and local identities. This validates ancestry/order and gives the same
	// structural review floor without interpreting the original model anew.
	compiled := model.Result{Status: model.Passed}
	for _, scope := range w.Compilation {
		if len(scope.Canonical) == 0 || scope.Digest != fmt.Sprintf("sha256:%x", sha256.Sum256(scope.Canonical)) {
			return bindingWire{}, 0, ErrInvalid
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
		return bindingWire{}, 0, ErrInvalid
	}
	actual, _ := json.Marshal(topology.Managers())
	recorded, _ := json.Marshal(w.Managers)
	if !bytes.Equal(actual, recorded) {
		return bindingWire{}, 0, ErrInvalid
	}
	reviews := reviewFloor(topology.Managers())
	if err := w.Policy.validate(reviews); err != nil {
		return bindingWire{}, 0, err
	}
	return w, reviews, nil
}
