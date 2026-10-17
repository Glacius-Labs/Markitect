package realization

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

// All fields are required, including explicit empty arrays. Wire structs are
// private, versioned structural facts, never compiler serialization formats.
type candidateRecordWire struct {
	Version         int                    `json:"version"`
	Models          []recordModel          `json:"models"`
	BaseTree        []recordFile           `json:"baseTree"`
	ResultTree      []recordFile           `json:"resultTree"`
	Data            []recordData           `json:"data"`
	BaseRegistry    []recordRegistration   `json:"baseRegistry"`
	ResultRegistry  []recordRegistration   `json:"resultRegistry"`
	Owner           recordOwner            `json:"owner"`
	Creation        []recordCreation       `json:"creation"`
	Protections     []recordProtection     `json:"protections"`
	Claims          []string               `json:"claims"`
	Classifications []recordClassification `json:"classifications"`
	Assessments     []recordAssessment     `json:"assessments"`
}
type recordIdentity struct {
	Package string `json:"package"`
	ID      string `json:"id"`
}
type recordDefinition struct {
	Identity recordIdentity `json:"identity"`
	ModelID  string         `json:"modelID"`
}
type recordModel struct {
	ID          string             `json:"id"`
	Digest      string             `json:"digest"`
	Local       bool               `json:"local"`
	Packages    []string           `json:"packages"`
	Definitions []recordDefinition `json:"definitions"`
}
type recordFile struct {
	Path   string        `json:"path"`
	Mode   snapshot.Mode `json:"mode"`
	Digest string        `json:"digest"`
	Length int64         `json:"length"`
}
type recordData struct {
	Digest string `json:"digest"`
	Bytes  []byte `json:"bytes"`
}
type recordOwner struct {
	Root    bool   `json:"root"`
	Package string `json:"package"`
}
type recordRegistration struct {
	Owner recordOwner      `json:"owner"`
	Path  string           `json:"path"`
	Form  Form             `json:"form"`
	Basis []recordIdentity `json:"basis"`
	Why   string           `json:"why"`
}
type recordCreation struct {
	Owner     recordOwner `json:"owner"`
	Directory string      `json:"directory"`
}
type recordProtection struct {
	File      string `json:"file"`
	Directory string `json:"directory"`
	Why       string `json:"why"`
}
type recordClassification struct {
	Path  string    `json:"path"`
	Class FileClass `json:"class"`
	Why   string    `json:"why"`
}
type recordEvidence struct {
	Ref       string `json:"ref"`
	Relevance string `json:"relevance"`
}
type recordAssessment struct {
	Definition  recordIdentity   `json:"definition"`
	Disposition Disposition      `json:"disposition"`
	Artifacts   []string         `json:"artifacts"`
	Uses        []recordIdentity `json:"uses"`
	Reason      string           `json:"reason"`
	Evidence    []recordEvidence `json:"evidence"`
}

func identityWire(id model.Identity) recordIdentity { return recordIdentity{id.Package, id.ID} }
func (id recordIdentity) value() model.Identity {
	return model.Identity{Package: id.Package, ID: id.ID}
}
func ownerWire(o Owner) recordOwner { return recordOwner{o.root, o.packageName} }
func (o recordOwner) value() (Owner, error) {
	if o.Root {
		if o.Package != "" {
			return Owner{}, ErrInvalidOwner
		}
		return RootOwner(), nil
	}
	return PackageOwner(o.Package)
}
func recordWire(inventory modelInventory, candidate Candidate, classes []Classification, rows []DefinitionAssessment, limits CandidateRecordLimits) (candidateRecordWire, error) {
	w := candidateRecordWire{Version: 1, Models: []recordModel{}, BaseTree: []recordFile{}, ResultTree: []recordFile{}, Data: []recordData{}, BaseRegistry: []recordRegistration{}, ResultRegistry: []recordRegistration{}, Owner: ownerWire(candidate.Owner()), Creation: []recordCreation{}, Protections: []recordProtection{}, Claims: []string{}, Classifications: []recordClassification{}, Assessments: []recordAssessment{}}
	for _, scope := range inventory.models {
		m := recordModel{ID: scope.id, Digest: scope.digest, Local: scope.local, Packages: append([]string{}, scope.packages...), Definitions: []recordDefinition{}}
		for _, d := range scope.definitions {
			m.Definitions = append(m.Definitions, recordDefinition{identityWire(d.identity), d.modelID})
		}
		w.Models = append(w.Models, m)
	}
	// Count complete actual input manifests before building their data table.
	before, after := candidate.Base().Tree().Entries(), candidate.Tree().Entries()
	if len(before) > limits.ManifestEntries || len(after) > limits.ManifestEntries-len(before) {
		return candidateRecordWire{}, ErrCandidateRecordLimit
	}
	var total int64
	blobs := map[string]snapshot.Entry{}
	var dataEncoded int64
	for i, entries := range [][]snapshot.Entry{before, after} {
		for _, e := range entries {
			length := e.ByteLength()
			if length > limits.FileBytes-total {
				return candidateRecordWire{}, ErrCandidateRecordLimit
			}
			total += length
			digest := e.ContentID().String()
			if old, ok := blobs[digest]; ok {
				// Equal digest references must at least agree on their manifest length.
				if old.ByteLength() != length {
					return candidateRecordWire{}, ErrInvalidCandidateRecord
				}
			} else {
				// A base64 string needs four bytes per three source bytes. Reject before
				// any content copy or table serialization, even with permissive FileBytes.
				groups := length / 3
				if length%3 != 0 {
					groups++
				}
				if groups > (limits.EncodedBytes-dataEncoded)/4 {
					return candidateRecordWire{}, ErrCandidateRecordLimit
				}
				dataEncoded += groups * 4
				blobs[digest] = e
			}
			f := recordFile{e.Path().String(), e.Mode(), digest, length}
			if i == 0 {
				w.BaseTree = append(w.BaseTree, f)
			} else {
				w.ResultTree = append(w.ResultTree, f)
			}
		}
	}
	for digest := range blobs {
		w.Data = append(w.Data, recordData{digest, []byte{}})
	}
	slices.SortFunc(w.Data, func(a, b recordData) int { return strings.Compare(a.Digest, b.Digest) })
	registration := func(reg Registry) []recordRegistration {
		result := []recordRegistration{}
		for _, r := range reg.entries {
			basis := []recordIdentity{}
			for _, id := range r.artifact.basis {
				basis = append(basis, identityWire(id))
			}
			result = append(result, recordRegistration{ownerWire(r.owner), r.artifact.path.String(), r.artifact.form, basis, r.artifact.why})
		}
		return result
	}
	w.BaseRegistry, w.ResultRegistry = registration(candidate.Base().Registry()), registration(candidate.Registry())
	facts := candidate.Base().value.facts.basis
	for _, a := range facts.areas {
		w.Creation = append(w.Creation, recordCreation{ownerWire(a.owner), a.directory.String()})
	}
	for _, p := range facts.protections {
		w.Protections = append(w.Protections, recordProtection{p.file.String(), p.directory.String(), p.why})
	}
	for _, c := range classes {
		w.Classifications = append(w.Classifications, recordClassification{c.path.String(), c.class, c.why})
	}
	slices.SortFunc(w.Classifications, func(a, b recordClassification) int { return strings.Compare(a.Path, b.Path) })
	for _, row := range rows {
		r := recordAssessment{Definition: identityWire(row.Definition), Disposition: row.Disposition, Artifacts: []string{}, Uses: []recordIdentity{}, Reason: row.Reason, Evidence: []recordEvidence{}}
		for _, p := range row.Artifacts {
			r.Artifacts = append(r.Artifacts, p.String())
		}
		for _, id := range row.Uses {
			r.Uses = append(r.Uses, identityWire(id))
		}
		for _, e := range row.Evidence {
			r.Evidence = append(r.Evidence, recordEvidence{e.Ref, e.Relevance})
		}
		w.Assessments = append(w.Assessments, r)
	}
	// Check exact JSON size (with empty data placeholders) plus base64 content
	// before allocating any byte table or the serialized payload.
	if err := boundRecordJSON(w, limits.EncodedBytes-dataEncoded); err != nil {
		return candidateRecordWire{}, err
	}
	for i := range w.Data {
		w.Data[i].Bytes = blobs[w.Data[i].Digest].Bytes()
	}
	return w, nil
}

func validRecordDigest(s string, modelDigest bool) bool {
	if modelDigest {
		if !strings.HasPrefix(s, "sha256:") {
			return false
		}
		s = strings.TrimPrefix(s, "sha256:")
	}
	decoded, err := hex.DecodeString(s)
	return err == nil && len(decoded) == sha256.Size && s == strings.ToLower(s)
}

// This validates only recorded structural consistency, never compiler content,
// language validity, original compilation status or current model applicability.
func (w candidateRecordWire) inventory() (modelInventory, error) {
	inv := modelInventory{owners: map[string]bool{}, definitions: map[model.Identity]bool{}, local: map[model.Identity]bool{}, ready: true}
	ids := map[string]bool{}
	locals := 0
	for _, m := range w.Models {
		if strings.TrimSpace(m.ID) == "" || ids[m.ID] || !validRecordDigest(m.Digest, true) {
			return modelInventory{}, ErrInvalidCandidateRecord
		}
		ids[m.ID] = true
		if m.Local {
			locals++
		}
		scope := claimModelBasis{id: m.ID, digest: m.Digest, local: m.Local}
		packages := map[string]bool{}
		for _, p := range m.Packages {
			if _, err := snapshot.NewPath(p); err != nil || packages[p] {
				return modelInventory{}, ErrInvalidCandidateRecord
			}
			packages[p] = true
			scope.packages = append(scope.packages, p)
			if m.Local {
				inv.owners[p] = true
			}
		}
		for _, d := range m.Definitions {
			id := d.Identity.value()
			if !packages[id.Package] || strings.TrimSpace(id.ID) == "" || d.ModelID != m.ID || inv.definitions[id] {
				return modelInventory{}, ErrInvalidCandidateRecord
			}
			inv.definitions[id] = true
			scope.definitions = append(scope.definitions, inventoryDefinition{id, d.ModelID})
			if m.Local {
				inv.local[id] = true
			}
		}
		slices.Sort(scope.packages)
		slices.SortFunc(scope.definitions, func(a, b inventoryDefinition) int { return compareIdentity(a.identity, b.identity) })
		inv.models = append(inv.models, scope)
	}
	if locals != 1 {
		return modelInventory{}, ErrInvalidCandidateRecord
	}
	slices.SortFunc(inv.models, func(a, b claimModelBasis) int { return strings.Compare(a.id, b.id) })
	return inv, nil
}
func (w candidateRecordWire) observation() (CandidateObservation, modelInventory, error) {
	fail := func(err error) (CandidateObservation, modelInventory, error) {
		return CandidateObservation{}, modelInventory{}, err
	}
	if w.Version != 1 || len(w.Claims) != 0 {
		return fail(ErrInvalidCandidateRecord)
	}
	inv, err := w.inventory()
	if err != nil {
		return fail(err)
	}
	data := map[string][]byte{}
	for _, d := range w.Data {
		if !validRecordDigest(d.Digest, false) || fmt.Sprintf("%x", sha256.Sum256(d.Bytes)) != d.Digest {
			return fail(ErrInvalidCandidateRecord)
		}
		if _, ok := data[d.Digest]; ok {
			return fail(ErrInvalidCandidateRecord)
		}
		data[d.Digest] = d.Bytes
	}
	used := map[string]bool{}
	tree := func(files []recordFile) (snapshot.Tree, error) {
		entries := []snapshot.Entry{}
		for _, f := range files {
			content, ok := data[f.Digest]
			if !ok || int64(len(content)) != f.Length {
				return snapshot.Tree{}, ErrInvalidCandidateRecord
			}
			used[f.Digest] = true
			p, err := snapshot.NewPath(f.Path)
			if err != nil {
				return snapshot.Tree{}, err
			}
			e, err := snapshot.NewEntry(p, f.Mode, content)
			if err != nil {
				return snapshot.Tree{}, err
			}
			entries = append(entries, e)
		}
		return snapshot.NewTree(entries)
	}
	baseTree, err := tree(w.BaseTree)
	if err != nil {
		return fail(err)
	}
	resultTree, err := tree(w.ResultTree)
	if err != nil {
		return fail(err)
	}
	if len(used) != len(data) {
		return fail(ErrInvalidCandidateRecord)
	}
	registry := func(rows []recordRegistration) (Registry, error) {
		entries := []Registration{}
		for _, r := range rows {
			o, err := r.Owner.value()
			if err != nil {
				return Registry{}, err
			}
			p, err := snapshot.NewPath(r.Path)
			if err != nil {
				return Registry{}, err
			}
			basis := []model.Identity{}
			for _, id := range r.Basis {
				basis = append(basis, id.value())
			}
			a, err := NewArtifact(p, r.Form, basis, r.Why)
			if err != nil {
				return Registry{}, err
			}
			entry, err := NewRegistration(o, a)
			if err != nil {
				return Registry{}, err
			}
			entries = append(entries, entry)
		}
		return NewRegistry(entries)
	}
	baseRegistry, err := registry(w.BaseRegistry)
	if err != nil {
		return fail(err)
	}
	resultRegistry, err := registry(w.ResultRegistry)
	if err != nil {
		return fail(err)
	}
	areas := []CreationArea{}
	directory := func(s string) (snapshot.DirectoryBoundary, error) {
		if s == "." {
			return snapshot.RootDirectory(), nil
		}
		p, err := snapshot.NewPath(s)
		if err != nil {
			return snapshot.DirectoryBoundary{}, err
		}
		return snapshot.DirectoryAt(p)
	}
	for _, a := range w.Creation {
		o, err := a.Owner.value()
		if err != nil {
			return fail(err)
		}
		d, err := directory(a.Directory)
		if err != nil {
			return fail(err)
		}
		area, err := NewCreationArea(o, d)
		if err != nil {
			return fail(err)
		}
		areas = append(areas, area)
	}
	protections := []Protection{}
	protected := map[string]bool{}
	for _, p := range w.Protections {
		key := p.File + "\x00" + p.Directory
		if protected[key] {
			return fail(ErrInvalidClaimFacts)
		}
		protected[key] = true
		var protection Protection
		if p.File != "" && p.Directory == "" {
			file, e := snapshot.NewPath(p.File)
			if e != nil {
				return fail(e)
			}
			protection, err = ProtectFile(file, p.Why)
		} else if p.Directory != "" && p.File == "" {
			d, e := directory(p.Directory)
			if e != nil {
				return fail(e)
			}
			protection, err = ProtectDirectory(d, p.Why)
		} else {
			return fail(ErrInvalidClaimFacts)
		}
		if err != nil {
			return fail(err)
		}
		protections = append(protections, protection)
	}
	facts, err := newPathFacts(inv, baseRegistry, baseTree, areas, protections)
	if err != nil {
		return fail(err)
	}
	base, err := newCandidateBase(inv, facts)
	if err != nil {
		return fail(err)
	}
	owner, err := w.Owner.value()
	if err != nil {
		return fail(err)
	}
	capture := CaptureCandidate(base, owner, PathClaims{}, resultTree, resultRegistry)
	candidate, ok := capture.Candidate()
	if !ok {
		return fail(fmt.Errorf("candidate capture refused: %v", capture.Findings()))
	}
	classes := []Classification{}
	seenClasses := map[snapshot.Path]bool{}
	for _, c := range w.Classifications {
		p, err := snapshot.NewPath(c.Path)
		if err != nil {
			return fail(err)
		}
		if seenClasses[p] {
			return fail(ErrInvalidClassification)
		}
		seenClasses[p] = true
		class, err := NewClassification(p, c.Class, c.Why)
		if err != nil {
			return fail(err)
		}
		classes = append(classes, class)
	}
	rows := []DefinitionAssessment{}
	seenRows := map[model.Identity]bool{}
	for _, r := range w.Assessments {
		row := DefinitionAssessment{Definition: r.Definition.value(), Disposition: r.Disposition, Reason: r.Reason, Artifacts: []snapshot.Path{}, Uses: []model.Identity{}, Evidence: []EvidenceUse{}}
		if seenRows[row.Definition] {
			return fail(ErrInvalidCandidateRecord)
		}
		seenRows[row.Definition] = true
		artifacts := map[snapshot.Path]bool{}
		for _, s := range r.Artifacts {
			p, e := snapshot.NewPath(s)
			if e != nil {
				return fail(e)
			}
			if artifacts[p] {
				return fail(ErrInvalidCandidateRecord)
			}
			artifacts[p] = true
			row.Artifacts = append(row.Artifacts, p)
		}
		uses := map[model.Identity]bool{}
		for _, id := range r.Uses {
			if uses[id.value()] {
				return fail(ErrInvalidCandidateRecord)
			}
			uses[id.value()] = true
			row.Uses = append(row.Uses, id.value())
		}
		refs := map[string]bool{}
		for _, e := range r.Evidence {
			if refs[e.Ref] {
				return fail(ErrInvalidCandidateRecord)
			}
			refs[e.Ref] = true
			row.Evidence = append(row.Evidence, EvidenceUse{e.Ref, e.Relevance})
		}
		rows = append(rows, row)
	}
	census := checkInventoryCensus(inv, base, resultTree, resultRegistry, DefinitionCensus{candidate: candidate, owner: owner}, classes, rows)
	return CandidateObservation{candidate, census}, inv, nil
}

// The token pass bounds rows, nesting, logical file lengths and decoded data
// before unmarshalling allocates manifest/metadata slices or decoded byte tables.
// It retains only a finite depth of parser state, not a second object graph.
func preflightRecord(encoded []byte, limits CandidateRecordLimits) error {
	if !limits.valid() || int64(len(encoded)) > limits.EncodedBytes {
		return ErrCandidateRecordLimit
	}
	if !utf8.Valid(encoded) {
		return ErrInvalidCandidateRecord
	}
	decoder := json.NewDecoder(bytes.NewReader(encoded))
	decoder.UseNumber()
	var rows, entries int
	var fileBytes, dataBytes int64
	var scan func(int, string, string) error
	scan = func(depth int, section, key string) error {
		if depth > 16 {
			return ErrInvalidCandidateRecord
		}
		token, err := decoder.Token()
		if err != nil {
			return ErrInvalidCandidateRecord
		}
		if delim, ok := token.(json.Delim); ok {
			switch delim {
			case '{':
				rows++
				if rows > limits.MetadataRows {
					return ErrCandidateRecordLimit
				}
				if section == "baseTree" || section == "resultTree" {
					entries++
					if entries > limits.ManifestEntries {
						return ErrCandidateRecordLimit
					}
				}
				for decoder.More() {
					field, e := decoder.Token()
					if e != nil {
						return ErrInvalidCandidateRecord
					}
					name, ok := field.(string)
					if !ok {
						return ErrInvalidCandidateRecord
					}
					next := section
					if depth == 0 {
						next = name
					}
					if err := scan(depth+1, next, name); err != nil {
						return err
					}
				}
				close, e := decoder.Token()
				if e != nil || close != json.Delim('}') {
					return ErrInvalidCandidateRecord
				}
			case '[':
				for decoder.More() {
					if err := scan(depth+1, section, ""); err != nil {
						return err
					}
				}
				close, e := decoder.Token()
				if e != nil || close != json.Delim(']') {
					return ErrInvalidCandidateRecord
				}
			default:
				return ErrInvalidCandidateRecord
			}
			return nil
		}
		if key == "" {
			rows++
			if rows > limits.MetadataRows {
				return ErrCandidateRecordLimit
			}
		}
		if (section == "baseTree" || section == "resultTree") && key == "length" {
			n, ok := token.(json.Number)
			if !ok {
				return ErrInvalidCandidateRecord
			}
			length, e := n.Int64()
			if e != nil || length < 0 {
				return ErrInvalidCandidateRecord
			}
			if length > limits.FileBytes-fileBytes {
				return ErrCandidateRecordLimit
			}
			fileBytes += length
		}
		if section == "data" && key == "bytes" {
			s, ok := token.(string)
			if !ok {
				return ErrInvalidCandidateRecord
			}
			length := base64.StdEncoding.DecodedLen(len(s))
			if strings.HasSuffix(s, "==") {
				length -= 2
			} else if strings.HasSuffix(s, "=") {
				length--
			}
			if length < 0 || int64(length) > limits.FileBytes-dataBytes {
				return ErrCandidateRecordLimit
			}
			dataBytes += int64(length)
		}
		return nil
	}
	if err := scan(0, "", ""); err != nil {
		return err
	}
	if _, err := decoder.Token(); err != io.EOF {
		return ErrInvalidCandidateRecord
	}
	return nil
}

// Count the required-field wire representation before json.Marshal allocates its
// output. The only supported shapes are these private structs, slices and scalar
// values. String sizing matches encoding/json's default HTML escaping. This is a
// budget calculation, not an alternative encoder or validation engine.
func boundRecordJSON(w candidateRecordWire, remaining int64) error {
	take := func(n int64) bool {
		if n > remaining {
			return false
		}
		remaining -= n
		return true
	}
	stringSize := func(s string) error {
		if !utf8.ValidString(s) {
			return ErrInvalidCandidateRecord
		}
		if !take(2) {
			return ErrCandidateRecordLimit
		}
		for _, r := range s {
			n := int64(utf8.RuneLen(r))
			switch r {
			case '"', '\\', '\b', '\f', '\n', '\r', '\t':
				n = 2
			case '<', '>', '&', 0x2028, 0x2029:
				n = 6
			default:
				if r < 0x20 {
					n = 6
				}
			}
			if !take(n) {
				return ErrCandidateRecordLimit
			}
		}
		return nil
	}
	var count func(reflect.Value) error
	count = func(v reflect.Value) error {
		switch v.Kind() {
		case reflect.Struct:
			if !take(2) {
				return ErrCandidateRecordLimit
			}
			for i := 0; i < v.NumField(); i++ {
				if i > 0 && !take(1) {
					return ErrCandidateRecordLimit
				}
				if err := stringSize(v.Type().Field(i).Tag.Get("json")); err != nil {
					return err
				}
				if !take(1) {
					return ErrCandidateRecordLimit
				}
				if err := count(v.Field(i)); err != nil {
					return err
				}
			}
		case reflect.Slice:
			if v.Type().Elem().Kind() == reflect.Uint8 {
				if !take(2 + 4*((int64(v.Len())+2)/3)) {
					return ErrCandidateRecordLimit
				}
				return nil
			}
			if !take(2) {
				return ErrCandidateRecordLimit
			}
			for i := 0; i < v.Len(); i++ {
				if i > 0 && !take(1) {
					return ErrCandidateRecordLimit
				}
				if err := count(v.Index(i)); err != nil {
					return err
				}
			}
		case reflect.String:
			return stringSize(v.String())
		case reflect.Bool:
			n := int64(5)
			if v.Bool() {
				n = 4
			}
			if !take(n) {
				return ErrCandidateRecordLimit
			}
		case reflect.Int, reflect.Int64:
			if !take(int64(len(strconv.FormatInt(v.Int(), 10)))) {
				return ErrCandidateRecordLimit
			}
		case reflect.Uint8:
			if !take(int64(len(strconv.FormatUint(v.Uint(), 10)))) {
				return ErrCandidateRecordLimit
			}
		default:
			return ErrInvalidCandidateRecord
		}
		return nil
	}
	return count(reflect.ValueOf(w))
}
