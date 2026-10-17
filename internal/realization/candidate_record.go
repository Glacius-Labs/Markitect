package realization

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

var (
	ErrInvalidCandidateRecord = errors.New("invalid candidate record")
	ErrCandidateRecordLimit   = errors.New("candidate record limit exceeded or invalid")
)

// CandidateRecordLimits are explicit tool budgets, not language maxima. Every
// field must be positive. EncodedBytes cannot exceed the v1 64 MiB ceiling.
// FileBytes counts logical bytes in BOTH manifests, including unchanged files.
// ManifestEntries counts both manifests. MetadataRows counts JSON object rows
// and scalar array members (including manifests/data). No field selects defaults.
type CandidateRecordLimits struct {
	EncodedBytes    int64
	FileBytes       int64
	ManifestEntries int
	MetadataRows    int
}

func (l CandidateRecordLimits) valid() bool {
	return l.EncodedBytes > 0 && l.EncodedBytes <= 64<<20 && l.FileBytes > 0 && l.ManifestEntries > 0 && l.MetadataRows > 0
}

// CandidateObservation is a historical structural observation, not model.Result,
// compilation success, workspace proof, accepted Delivery or review success.
// Evidence handle existence/support and acquisition/authority remain obligations
// of the Host and independent reviewer. Zero is unavailable.
type CandidateObservation struct {
	candidate Candidate
	census    DefinitionCensus
}

func (o CandidateObservation) Candidate() Candidate     { return o.candidate }
func (o CandidateObservation) Census() DefinitionCensus { return o.census }

// CandidateRecord fixes a complete canonical v1 payload and its derived local
// observation. Hashes bind the whole subject, including registry reasons and
// ordered assessment proposals; they do not authenticate original acquisition.
// All exposed slices are copied through the existing immutable value APIs.
type CandidateRecord struct {
	encoded     string
	observation CandidateObservation
}

func (r CandidateRecord) Valid() bool                       { return r.encoded != "" }
func (r CandidateRecord) Bytes() []byte                     { return []byte(r.encoded) }
func (r CandidateRecord) Observation() CandidateObservation { return r.observation }
func (r CandidateRecord) Digest() string {
	if !r.Valid() {
		return ""
	}
	return fmt.Sprintf("sha256:%x", sha256.Sum256([]byte(r.encoded)))
}

// NewCandidateRecord records an actual supplied Passed compiler result matched
// to the exact CandidateBase. V1 deliberately supports EMPTY claims only; new
// unclaimed paths refuse on decode using the ordinary capture engine. Incomplete
// or invalid census proposals are retained with their recomputed status.
func NewCandidateRecord(compiled model.Result, candidate Candidate, classifications []Classification, rows []DefinitionAssessment, limits CandidateRecordLimits) (CandidateRecord, error) {
	if !limits.valid() {
		return CandidateRecord{}, ErrCandidateRecordLimit
	}
	if !candidate.Valid() || len(candidate.value.claims) != 0 {
		return CandidateRecord{}, ErrInvalidCandidateRecord
	}
	if err := boundRecordInputs(compiled, candidate, classifications, rows, limits); err != nil {
		return CandidateRecord{}, err
	}
	inventory := inventoryFromResult(compiled)
	matching, err := newCandidateBase(inventory, candidate.Base().value.facts)
	if err != nil || !candidate.Base().Equal(matching) {
		return CandidateRecord{}, ErrInvalidCandidateRecord
	}
	wire, err := recordWire(inventory, candidate, classifications, rows, limits)
	if err != nil {
		return CandidateRecord{}, err
	}
	encoded, err := json.Marshal(wire)
	if err != nil {
		return CandidateRecord{}, fmt.Errorf("%w: %v", ErrInvalidCandidateRecord, err)
	}
	// Use the same closed, bounded historical path even for live construction.
	return DecodeCandidateRecord(encoded, limits)
}

// DecodeCandidateRecord validates closed canonical JSON and recorded consistency
// with explicit finite budgets. It reconstructs through checked constructors,
// CaptureCandidate and the shared accounting/census core, never a fabricated
// model.Result. Failure returns no partial record or observation. A valid changed
// subject is a different record identity, not proof of authenticated acquisition.
func DecodeCandidateRecord(encoded []byte, limits CandidateRecordLimits) (CandidateRecord, error) {
	if err := preflightRecord(encoded, limits); err != nil {
		return CandidateRecord{}, err
	}
	var wire candidateRecordWire
	decoder := json.NewDecoder(bytes.NewReader(encoded))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&wire); err != nil {
		return CandidateRecord{}, fmt.Errorf("%w: %v", ErrInvalidCandidateRecord, err)
	}
	canonical, err := json.Marshal(wire)
	if err != nil || !bytes.Equal(encoded, canonical) {
		return CandidateRecord{}, ErrInvalidCandidateRecord
	}
	observation, inventory, err := wire.observation()
	if err != nil {
		return CandidateRecord{}, fmt.Errorf("%w: %w", ErrInvalidCandidateRecord, err)
	}
	// Re-encoding checked values also rejects noncanonical set orders, null lists,
	// duplicate entries, unused data and representations normalized by constructors.
	checked, err := recordWire(inventory, observation.candidate, observation.census.classifications, observation.census.assessments, limits)
	if err != nil {
		return CandidateRecord{}, err
	}
	canonical, err = json.Marshal(checked)
	if err != nil || !bytes.Equal(encoded, canonical) {
		return CandidateRecord{}, ErrInvalidCandidateRecord
	}
	return CandidateRecord{string(encoded), observation}, nil
}

// Bound actual live inventories before projecting them into wire slices. The
// historical token pass applies the same row budget before decoding slices.
func boundRecordInputs(compiled model.Result, candidate Candidate, classes []Classification, assessments []DefinitionAssessment, limits CandidateRecordLimits) error {
	rows := int64(1) // root object
	textBytes := int64(0)
	charge := func(n int, texts ...string) bool {
		rows += int64(n)
		if rows > int64(limits.MetadataRows) || rows > limits.EncodedBytes/2 {
			return false
		}
		for _, s := range texts {
			if int64(len(s)) > limits.EncodedBytes-textBytes {
				return false
			}
			textBytes += int64(len(s))
		}
		return rows <= int64(limits.MetadataRows)
	}
	for _, scope := range compiled.Models {
		if !charge(1, scope.ID, scope.Digest) {
			return ErrCandidateRecordLimit
		}
		for _, p := range scope.Packages {
			if !charge(1, p.Name) {
				return ErrCandidateRecordLimit
			}
		}
		for _, d := range scope.Definitions {
			if !charge(2, d.Identity.Package, d.Identity.ID, d.ModelID) {
				return ErrCandidateRecordLimit
			}
		}
	}
	manifestEntries := 0
	var fileBytes int64
	digests := map[string]bool{}
	for _, tree := range []snapshot.Tree{candidate.Base().Tree(), candidate.Tree()} {
		count := tree.EntryCount()
		if count > limits.ManifestEntries-manifestEntries || int64(count) > int64(limits.MetadataRows)-rows || int64(count) > limits.EncodedBytes/2-rows {
			return ErrCandidateRecordLimit
		}
		manifestEntries += count
		entries := tree.Entries()
		for _, e := range entries {
			if e.ByteLength() > limits.FileBytes-fileBytes {
				return ErrCandidateRecordLimit
			}
			fileBytes += e.ByteLength()
			// Even one deduplicated occurrence needs its base64 content in the payload.
			// Check this before ContentID, which itself may copy snapshot bytes to hash.
			groups := e.ByteLength() / 3
			if e.ByteLength()%3 != 0 {
				groups++
			}
			if groups > limits.EncodedBytes/4 {
				return ErrCandidateRecordLimit
			}
			// Content digests have a fixed 64-byte wire spelling.
			if !charge(1, e.Path().String(), "0000000000000000000000000000000000000000000000000000000000000000") {
				return ErrCandidateRecordLimit
			}
			digest := e.ContentID().String()
			if !digests[digest] {
				if !charge(1, digest) {
					return ErrCandidateRecordLimit
				}
				digests[digest] = true
			}
		}
	}
	facts := candidate.Base().value.facts.basis
	for _, a := range facts.areas {
		if !charge(2, a.owner.packageName, a.directory.String()) {
			return ErrCandidateRecordLimit
		}
	}
	for _, p := range facts.protections {
		if !charge(1, p.file.String(), p.directory.String(), p.why) {
			return ErrCandidateRecordLimit
		}
	}
	for _, registry := range []Registry{candidate.Base().Registry(), candidate.Registry()} {
		for _, r := range registry.entries {
			if !charge(2, r.owner.packageName, r.artifact.path.String(), string(r.artifact.form), r.artifact.why) {
				return ErrCandidateRecordLimit
			}
			for _, id := range r.artifact.basis {
				if !charge(1, id.Package, id.ID) {
					return ErrCandidateRecordLimit
				}
			}
		}
	}
	if !charge(1, candidate.Owner().packageName) {
		return ErrCandidateRecordLimit
	}
	for _, c := range classes {
		if !charge(1, c.path.String(), string(c.class), c.why) {
			return ErrCandidateRecordLimit
		}
	}
	for _, a := range assessments {
		if !charge(2, a.Definition.Package, a.Definition.ID, string(a.Disposition), a.Reason) {
			return ErrCandidateRecordLimit
		}
		for _, p := range a.Artifacts {
			if !charge(1, p.String()) {
				return ErrCandidateRecordLimit
			}
		}
		for _, id := range a.Uses {
			if !charge(1, id.Package, id.ID) {
				return ErrCandidateRecordLimit
			}
		}
		for _, e := range a.Evidence {
			if !charge(1, e.Ref, e.Relevance) {
				return ErrCandidateRecordLimit
			}
		}
	}
	return nil
}
