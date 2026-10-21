package run

import (
	"fmt"

	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

// modelBasisPending identifies an outstanding model-only precommitment.
func (m *member) modelBasisPending() bool {
	return m.p.spec.ModelBasis && m.basis == nil
}

// candidateHidden also keeps a running model-only turn blind if the last
// required basis is accepted during that turn. Candidate introduction happens
// only when a fresh turn starts after the pass-wide barrier opens.
func (m *member) candidateHidden() bool {
	return m.p.spec.ModelBasis && (!m.p.modelBasisReady.Load() || !m.candidateIntroduced)
}

// openModelCandidateLocked opens once every required reviewer has an accepted
// basis. Until then even accepted parents only coordinate model-derived work.
func (e *engine) openModelCandidateLocked(p *pass) {
	if !p.spec.ModelBasis || p.modelBasisReady.Load() {
		return
	}
	for _, m := range p.members {
		if m.basis == nil {
			return
		}
	}
	p.modelBasisReady.Store(true)
	for _, m := range p.members {
		e.deliverLocked(m, item{kind: itemAssignment})
	}
}

func (e *engine) prepareModelBasis(p *pass) bool {
	if !p.spec.ModelBasis {
		return true
	}
	tree, err := snapshot.NewTree(nil)
	if err != nil {
		e.fail(err)
		return false
	}
	id := WorkspaceID(fmt.Sprintf("%s-%s-model-basis", e.cfg.RunID, p.id))
	volume, ok := e.materialize(id, tree, nil)
	if !ok {
		e.closeVolume(id)
		return false
	}
	p.modelBasisID, p.modelBasisVolume = id, volume
	return true
}

// modelBasisOf keeps the existing basis envelope as a precommitment rather
// than inventing another review role, phase engine or evidence declaration.
func modelBasisOf(m *member, in reportIn) error {
	if !m.modelBasisPending() || in.Kind != kindBasis {
		return nil
	}
	if len(in.Artifacts)+len(in.Evidence)+len(in.Findings)+len(in.Closed)+len(in.Disputed)+len(in.Dispositions) > 0 {
		return fmt.Errorf("the model-first review basis records conditions and intended cases, not candidate evidence or findings; assess the candidate after basis acceptance")
	}
	for _, c := range in.Obligations {
		if c.Status != coverOpen || len(c.Evidence) > 0 {
			return fmt.Errorf("the model-first basis keeps %s open without candidate evidence; state source-derived conditions, expected outcomes and cases in conditions, and the pending assessment in note", c.Definition)
		}
	}
	if c := in.Integration; c != nil && (c.Status != coverOpen || len(c.Evidence) > 0) {
		return fmt.Errorf("the model-first basis keeps integration open without candidate evidence; state intended composed cases and model authority")
	}
	return nil
}
