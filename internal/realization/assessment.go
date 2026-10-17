package realization

import (
	"slices"

	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

// Disposition is an assessment proposal, never an inferred semantic verdict.
type Disposition string

const (
	Realized   Disposition = "realized"
	Constrains Disposition = "constrains"
	Vocabulary Disposition = "vocabulary"
	Unresolved Disposition = "unresolved"
)

// EvidenceUse cites an opaque Host-owned handle and explains its relevance.
// This package neither issues nor resolves handles. Run must validate actual
// existence, result, subject, input and purpose bindings; reviewers judge support.
type EvidenceUse struct{ Ref, Relevance string }

// DefinitionAssessment is supplied proposal data. Census takes and returns deep
// copies. Uses names proposed concrete local uses, not a derived semantic graph.
// Vocabulary needs no standalone artifact; its reasons/evidence must explain uses
// or the claimed absence of uses. That claim never waives real use obligations.
type DefinitionAssessment struct {
	Definition  model.Identity
	Disposition Disposition
	Artifacts   []snapshot.Path
	Uses        []model.Identity
	Reason      string
	Evidence    []EvidenceUse
}

func copyAssessments(rows []DefinitionAssessment) []DefinitionAssessment {
	rows = slices.Clone(rows)
	for i := range rows {
		rows[i].Artifacts = slices.Clone(rows[i].Artifacts)
		rows[i].Uses = slices.Clone(rows[i].Uses)
		rows[i].Evidence = slices.Clone(rows[i].Evidence)
	}
	return rows
}

// CensusStatus concerns only the explicit inventory and proposal minimum form.
// StructurallyComplete does not mean Accounted, realized, reviewed or accepted.
type CensusStatus uint8

const (
	CensusIncomplete CensusStatus = iota
	CensusInvalid
	CensusStructurallyComplete
)

// DefinitionCensus binds one model basis, exact immutable subject, explicit scope
// and copied classification facts to its report. Zero is unavailable/incomplete.
// Rows preserve proposal order; inventory lists and findings are deterministic.
// Independent old accounting findings remain visible without a global Accounted
// gate. No caller-supplied Accounting report can substitute for the fixed inputs.
type DefinitionCensus struct {
	status                        CensusStatus
	basis                         CandidateBase
	candidate                     Candidate
	aggregate                     Aggregate
	owner                         Owner
	project                       bool
	classifications               []Classification
	assessments                   []DefinitionAssessment
	expected, missing, unresolved []model.Identity
	findings                      []Finding
	accounting                    Accounting
}

func (c DefinitionCensus) Status() CensusStatus         { return c.status }
func (c DefinitionCensus) Basis() CandidateBase         { return c.basis }
func (c DefinitionCensus) Candidate() (Candidate, bool) { return c.candidate, c.candidate.Valid() }
func (c DefinitionCensus) Aggregate() (Aggregate, bool) {
	return c.aggregate, c.aggregate.Valid()
}
func (c DefinitionCensus) ProjectScope() bool { return c.project && c.aggregate.Valid() }

// Owner returns the requested own-scope binding, not a live authority assertion.
func (c DefinitionCensus) Owner() (Owner, bool) {
	return c.owner, !c.project && (c.candidate.Valid() || c.aggregate.Valid())
}
func (c DefinitionCensus) Classifications() []Classification   { return slices.Clone(c.classifications) }
func (c DefinitionCensus) Assessments() []DefinitionAssessment { return copyAssessments(c.assessments) }
func (c DefinitionCensus) Expected() []model.Identity          { return slices.Clone(c.expected) }
func (c DefinitionCensus) Missing() []model.Identity           { return slices.Clone(c.missing) }
func (c DefinitionCensus) Unresolved() []model.Identity        { return slices.Clone(c.unresolved) }
func (c DefinitionCensus) Findings() []Finding                 { return slices.Clone(c.findings) }
func (c DefinitionCensus) Accounting() Accounting              { return c.accounting }

// CheckCandidateCensus checks exactly the candidate owner's own local Definitions,
// not descendants or imported vocabulary. Root-own is empty, never a project census.
func CheckCandidateCensus(compiled model.Result, candidate Candidate, classifications []Classification, rows []DefinitionAssessment) DefinitionCensus {
	return checkCensus(compiled, candidate.Base(), candidate.Tree(), candidate.Registry(), DefinitionCensus{candidate: candidate, owner: candidate.Owner()}, classifications, rows)
}

// CheckCompositionCensus checks one explicitly bound local owner on the accepted
// aggregate. The Host supplies the owner from its task; Agent claims grant no
// authority.
func CheckCompositionCensus(compiled model.Result, aggregate Aggregate, owner Owner, classifications []Classification, rows []DefinitionAssessment) DefinitionCensus {
	return checkCensus(compiled, aggregate.Base(), aggregate.Tree(), aggregate.Registry(), DefinitionCensus{aggregate: aggregate, owner: owner}, classifications, rows)
}

// CheckProjectCensus checks every Definition in the selected local model on the
// exact accepted aggregate, independently of which deliveries composed it.
func CheckProjectCensus(compiled model.Result, aggregate Aggregate, classifications []Classification, rows []DefinitionAssessment) DefinitionCensus {
	return checkCensus(compiled, aggregate.Base(), aggregate.Tree(), aggregate.Registry(), DefinitionCensus{aggregate: aggregate, project: true}, classifications, rows)
}
