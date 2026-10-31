package desktop

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"path"
	"slices"
	"sort"
	"strings"
	"sync"

	"github.com/Glacius-Labs/Markitect/internal/analysis"
	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/project"
)

// The Model area's four views share one document per project: the working
// tree's model with what refs, graph, impact and lint say about it, and the
// change from a base. The window asks for it again every few seconds; the
// answer is computed again only when a file, the last commit or the last
// landing changed.

// The bases a change is measured from.
const (
	baseLanding = "landing" // the last commit a Run landed: the base apply takes
	baseCommit  = "commit"  // the last commit: what is not committed yet
)

// modelView is the Model area's document.
type modelView struct {
	Initialized bool  `json:"initialized"`
	Bases       bases `json:"bases"`
	// Base is the base the change is measured from; nil when nothing is
	// committed, so the whole model is the change.
	Base        *base            `json:"base"`
	Errors      []diagnostic     `json:"errors"`
	Lint        []diagnostic     `json:"lint"`
	Definitions []definitionView `json:"definitions"`
	Packages    []packageView    `json:"packages"`
	Edges       []edgeView       `json:"edges"`
	Cycles      [][]string       `json:"cycles"`
	// Impact is markitect impact from the base, with every working
	// Manager's checklist; ImpactNote says how it was taken, or why not.
	Impact     *analysis.Impact `json:"impact,omitempty"`
	ImpactNote string           `json:"impactNote,omitempty"`
	Diff       diffView         `json:"diff"`
}

// bases are the commits there are to compare with.
type bases struct {
	Commit  string `json:"commit,omitempty"`
	Landing string `json:"landing,omitempty"`
}

type base struct {
	Kind   string `json:"kind"` // baseLanding or baseCommit
	Commit string `json:"commit"`
}

// diagnostic is a compile error or a lint warning with where it belongs.
type diagnostic struct {
	model.Diagnostic
	Package    string `json:"package"`              // the Package it concerns; Root when no other
	Definition string `json:"definition,omitempty"` // the Definition in whose file it is
}

type definitionView struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Kind     string `json:"kind"`
	Library  string `json:"library,omitempty"` // a Library's Kind; empty for the model's own
	Package  string `json:"package"`
	File     string `json:"file,omitempty"`
	Internal bool   `json:"internal,omitempty"`
	Words    int    `json:"words"`
	Text     string `json:"text"`
	Links    []link `json:"links"`
	// ReferencedBy are the Definitions that link to it or are of it, as
	// markitect refs lists them; RealizedBy the files whose registry entries
	// name it.
	ReferencedBy []analysis.Ref `json:"referencedBy"`
	RealizedBy   []realizedBy   `json:"realizedBy"`
	SeenBy       *seenBy        `json:"seenBy,omitempty"`
	// Change is added or modified when the Definition differs from the base.
	Change string `json:"change,omitempty"`
}

type link struct {
	Raw     string `json:"raw"`
	Target  string `json:"target"` // empty when it resolves to nothing
	Package bool   `json:"package,omitempty"`
	Kind    bool   `json:"kind,omitempty"`
	Line    int    `json:"line"`
}

type realizedBy struct {
	File   string `json:"file"`
	Reason string `json:"reason"`
	Page   string `json:"page"`
	Line   int    `json:"line"`
}

// seenBy names the Managers that know a Definition when they start
// (concept 2.0 §5): in full, or by title and first sentence in the table of
// contents of a child.
type seenBy struct {
	Full     []string `json:"full"`
	Contents []string `json:"contents"`
}

type packageView struct {
	analysis.PackageNode
	Name     string   `json:"name"`
	Children []string `json:"children"`
	Own      []string `json:"own"` // its own Definitions
	Patterns []string `json:"patterns"`
	HasPage  bool     `json:"hasPage"`
	Files    int      `json:"files"` // the files it owns
	Change   string   `json:"change,omitempty"`
}

type edgeView struct {
	From  string `json:"from"`
	To    string `json:"to"`
	Uses  []use  `json:"uses"`
	Cycle bool   `json:"cycle,omitempty"` // both ends lie in one cycle
}

type use struct {
	From   string `json:"from"`
	Target string `json:"target"`
	Kind   bool   `json:"kind,omitempty"`
	File   string `json:"file"`
	Line   int    `json:"line"`
}

// diffView is the change from the base, for the views' diff toggle.
type diffView struct {
	Definitions []definitionChange `json:"definitions"`
	Packages    []analysis.Change  `json:"packages"`
	Edges       []edgeChange       `json:"edges"`
	// Lint holds the warnings that are new since the base (added) and those
	// that are gone (removed).
	Lint []lintChange `json:"lint"`
}

type definitionChange struct {
	ID      string `json:"id"`
	Change  string `json:"change"`
	Title   string `json:"title"`
	Package string `json:"package"`
	// BaseText is the Definition's text at the base, for one that was
	// modified or removed.
	BaseText string `json:"baseText,omitempty"`
}

type edgeChange struct {
	From   string `json:"from"`
	To     string `json:"to"`
	Change string `json:"change"`
}

type lintChange struct {
	diagnostic
	Change string `json:"change"`
}

// baseState is the project at a base commit with its lint warnings.
type baseState struct {
	project *project.Project
	lint    model.Diagnostics
}

// buildModelView derives the document from the working tree's project and
// the base, which is nil when nothing is committed; changed are the files
// that differ between the base and the working tree.
func buildModelView(cur *project.Project, b *baseState, bs bases, chosen *base, changed []analysis.Change) modelView {
	m := cur.Model
	v := modelView{Initialized: true, Bases: bs, Base: chosen, Errors: []diagnostic{}, Lint: []diagnostic{},
		Definitions: []definitionView{}, Packages: []packageView{}, Edges: []edgeView{}, Cycles: [][]string{}}
	var bm *model.Model
	var blint model.Diagnostics
	if b != nil {
		bm, blint = b.project.Model, b.lint
	}
	where := placer(cur)
	for _, d := range cur.Errors {
		v.Errors = append(v.Errors, where(d))
	}
	lint := analysis.Lint(cur)
	for _, d := range lint {
		v.Lint = append(v.Lint, where(d))
	}

	whereBase := where
	if b != nil {
		whereBase = placer(b.project)
	}
	v.Diff = diffModels(bm, m, blint, lint, where, whereBase)
	changes := map[string]string{}
	for _, c := range v.Diff.Definitions {
		changes[c.ID] = c.Change
	}
	seen := whoSees(m)
	var ids []string
	for id := range m.Definitions {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		d := m.Definitions[id]
		dv := definitionView{ID: id, Title: d.Title, Kind: d.Kind, Package: d.Package, File: d.File, Internal: d.Internal, Words: d.Words,
			Text: string(d.Data), Links: []link{}, ReferencedBy: []analysis.Ref{}, RealizedBy: []realizedBy{}, SeenBy: seen[id], Change: changes[id]}
		if !d.InModel() {
			dv.Library, dv.Package = d.Library, ""
		}
		for _, l := range d.Links {
			dv.Links = append(dv.Links, link{Raw: l.Raw, Target: l.Target, Package: l.Package, Kind: l.Kind, Line: l.Line})
		}
		refs, _ := analysis.Refs(cur, id)
		for _, r := range refs {
			if r.Via != "registry" {
				dv.ReferencedBy = append(dv.ReferencedBy, r)
				continue
			}
			rb := realizedBy{File: r.From, Page: r.File, Line: r.Line}
			if e := cur.Registry.Entries[r.From]; e != nil {
				rb.Reason = e.Reason
			}
			dv.RealizedBy = append(dv.RealizedBy, rb)
		}
		v.Definitions = append(v.Definitions, dv)
	}

	packageChanges := map[string]string{}
	for _, c := range v.Diff.Packages {
		packageChanges[c.Path] = c.Change
	}
	owned := map[string]int{}
	for _, pkg := range cur.Registry.Owner {
		owned[pkg]++
	}
	g := analysis.BuildGraph(cur)
	for _, n := range g.Packages {
		pkg := m.Packages[n.Path]
		pv := packageView{PackageNode: n, Name: model.PackageName(n.Path), Children: slices.Clone(pkg.Children), Own: slices.Clone(pkg.Definitions),
			Patterns: []string{}, Files: owned[n.Path], Change: packageChanges[n.Path]}
		if pv.Children == nil {
			pv.Children = []string{}
		}
		if pv.Own == nil {
			pv.Own = []string{}
		}
		if page := cur.Registry.Pages[n.Path]; page != nil {
			pv.HasPage = true
			for _, p := range page.Patterns {
				pv.Patterns = append(pv.Patterns, p.Text)
			}
		}
		v.Packages = append(v.Packages, pv)
	}
	inCycle := map[string]int{}
	for i, c := range m.Cycles() {
		v.Cycles = append(v.Cycles, c)
		for _, p := range c {
			inCycle[p] = i + 1
		}
	}
	for _, e := range g.Edges {
		ev := edgeView{From: e.From, To: e.To, Uses: []use{}, Cycle: inCycle[e.From] != 0 && inCycle[e.From] == inCycle[e.To]}
		for _, u := range e.Uses {
			ev.Uses = append(ev.Uses, use{From: u.From, Target: u.Target, Kind: u.Kind, File: u.File, Line: u.Line})
		}
		v.Edges = append(v.Edges, ev)
	}

	switch {
	case len(cur.Errors) > 0:
		v.ImpactNote = "markitect impact needs a model that compiles; the checklists show once the compile errors are fixed."
	case b == nil:
		im := analysis.ComputeImpact(analysis.ImpactInput{Base: cur, Current: cur, Full: true})
		v.Impact = &im
		v.ImpactNote = "Nothing is committed yet, so the whole model is the change, as with --full."
	default:
		im := analysis.ComputeImpact(analysis.ImpactInput{Base: b.project, Current: cur, Changed: changed})
		v.Impact = &im
	}
	return v
}

// placer finds the Package, and the Definition if any, that a diagnostic's
// file belongs to: a Definition's file is its Package's, a registry page its
// Package's, a model file that did not compile its folder's, a realization
// file its owner's, and everything else Root's.
func placer(p *project.Project) func(model.Diagnostic) diagnostic {
	m := p.Model
	byFile := map[string]string{}
	for id, d := range m.Definitions {
		if d.InModel() {
			byFile[d.File] = id
		}
	}
	pages := map[string]string{}
	for pkg, page := range p.Registry.Pages {
		pages[page.File] = pkg
	}
	return func(d model.Diagnostic) diagnostic {
		out := diagnostic{Diagnostic: d, Package: model.RootPackage}
		if id, ok := byFile[d.File]; ok {
			out.Package, out.Definition = m.Definitions[id].Package, id
			return out
		}
		if pkg, ok := pages[d.File]; ok {
			out.Package = pkg
			return out
		}
		if rest, ok := strings.CutPrefix(d.File, project.RegistryDir+"/"); ok {
			if pkg, ok := strings.CutSuffix(rest, ".md"); ok && m.Packages[pkg] != nil {
				out.Package = pkg
			}
			return out
		}
		if rest, ok := strings.CutPrefix(d.File, project.ModelDir+"/"); ok {
			if dir := path.Dir(rest); m.Packages[dir] != nil {
				out.Package = dir
			}
			return out
		}
		if owner, ok := p.Registry.Owner[d.File]; ok {
			out.Package = owner
		}
		return out
	}
}

// whoSees mirrors what each Manager knows when it starts (concept 2.0 §5, as
// internal/harness renders it): its own Package's Definitions, everything its
// parents define, the Definitions these link to, the Kinds of all of them,
// and the public Definitions below each child by title and first sentence.
// A Definition a Manager knows in full is listed once, in full, although the
// harness also names it in the child's table of contents.
func whoSees(m *model.Model) map[string]*seenBy {
	out := map[string]*seenBy{}
	entry := func(id string) *seenBy {
		if out[id] == nil {
			out[id] = &seenBy{Full: []string{}, Contents: []string{}}
		}
		return out[id]
	}
	for _, pkg := range m.SortedPackages() {
		full := map[string]bool{}
		var shown []string
		add := func(id string) {
			if !full[id] && m.Definitions[id] != nil {
				full[id] = true
				shown = append(shown, id)
			}
		}
		for _, id := range m.Packages[pkg].Definitions {
			add(id)
		}
		for _, a := range model.Ancestors(pkg) {
			if p := m.Packages[a]; p != nil {
				for _, id := range p.Definitions {
					add(id)
				}
			}
		}
		for _, id := range slices.Clone(shown) {
			for _, l := range m.Definitions[id].Links {
				if !l.Package && !l.Kind {
					add(l.Target)
				}
			}
		}
		for _, id := range slices.Clone(shown) {
			add(m.Definitions[id].Kind)
		}
		for _, id := range shown {
			if m.Definitions[id].InModel() {
				entry(id).Full = append(entry(id).Full, pkg)
			}
		}
		for _, c := range m.Packages[pkg].Children {
			for _, q := range m.Subtree(c) {
				for _, id := range m.Packages[q].Definitions {
					if !m.Definitions[id].Internal && !full[id] {
						entry(id).Contents = append(entry(id).Contents, pkg)
					}
				}
			}
		}
	}
	for _, s := range out {
		sort.Strings(s.Full)
		sort.Strings(s.Contents)
	}
	return out
}

// diffModels compares the model at the base, nil when there is none, with
// the working tree's.
func diffModels(base, cur *model.Model, baseLint, curLint model.Diagnostics, where, whereBase func(model.Diagnostic) diagnostic) diffView {
	v := diffView{Definitions: []definitionChange{}, Packages: []analysis.Change{}, Edges: []edgeChange{}, Lint: []lintChange{}}
	before := func(id string) *model.Definition {
		if base == nil {
			return nil
		}
		if d := base.Definitions[id]; d != nil && d.InModel() {
			return d
		}
		return nil
	}
	for _, id := range cur.ModelDefinitions() {
		d, old := cur.Definitions[id], before(id)
		switch {
		case old == nil:
			v.Definitions = append(v.Definitions, definitionChange{ID: id, Change: analysis.Added, Title: d.Title, Package: d.Package})
		case !bytes.Equal(old.Data, d.Data):
			v.Definitions = append(v.Definitions, definitionChange{ID: id, Change: analysis.Modified, Title: d.Title, Package: d.Package, BaseText: string(old.Data)})
		}
	}
	if base != nil {
		for _, id := range base.ModelDefinitions() {
			if d := cur.Definitions[id]; d == nil || !d.InModel() {
				old := base.Definitions[id]
				v.Definitions = append(v.Definitions, definitionChange{ID: id, Change: analysis.Removed, Title: old.Title, Package: old.Package, BaseText: string(old.Data)})
			}
		}
	}
	sort.Slice(v.Definitions, func(i, j int) bool { return v.Definitions[i].ID < v.Definitions[j].ID })

	packages := func(m *model.Model) map[string]bool {
		out := map[string]bool{}
		if m != nil {
			for p := range m.Packages {
				out[p] = p != model.RootPackage
			}
		}
		return out
	}
	was, is := packages(base), packages(cur)
	for _, p := range cur.SortedPackages() {
		if is[p] && !was[p] {
			v.Packages = append(v.Packages, analysis.Change{Path: p, Change: analysis.Added})
		}
	}
	for p, ok := range was {
		if ok && !is[p] {
			v.Packages = append(v.Packages, analysis.Change{Path: p, Change: analysis.Removed})
		}
	}
	sort.Slice(v.Packages, func(i, j int) bool { return v.Packages[i].Path < v.Packages[j].Path })

	edges := func(m *model.Model) map[[2]string]bool {
		out := map[[2]string]bool{}
		if m != nil {
			for from, tos := range m.Edges {
				for to := range tos {
					out[[2]string{from, to}] = true
				}
			}
		}
		return out
	}
	oldEdges, newEdges := edges(base), edges(cur)
	for e := range newEdges {
		if !oldEdges[e] {
			v.Edges = append(v.Edges, edgeChange{From: e[0], To: e[1], Change: analysis.Added})
		}
	}
	for e := range oldEdges {
		if !newEdges[e] {
			v.Edges = append(v.Edges, edgeChange{From: e[0], To: e[1], Change: analysis.Removed})
		}
	}
	sort.Slice(v.Edges, func(i, j int) bool {
		a, b := v.Edges[i], v.Edges[j]
		return a.From < b.From || a.From == b.From && a.To < b.To
	})

	// A warning is the same one at the base when its code and message are;
	// its line may have moved.
	key := func(d model.Diagnostic) string { return d.Code + "\x00" + d.Message }
	had, has := map[string]bool{}, map[string]bool{}
	for _, d := range baseLint {
		had[key(d)] = true
	}
	for _, d := range curLint {
		has[key(d)] = true
		if !had[key(d)] {
			v.Lint = append(v.Lint, lintChange{diagnostic: where(d), Change: analysis.Added})
		}
	}
	for _, d := range baseLint {
		if !has[key(d)] {
			// A warning that is gone is placed where it was, at the base.
			v.Lint = append(v.Lint, lintChange{diagnostic: whereBase(d), Change: analysis.Removed})
		}
	}
	return v
}

// models caches the Model area's documents per project, and the projects at
// their bases, which are slow to read from Git and rarely change.
type models struct {
	mu       sync.Mutex
	views    map[string]cachedView // by project ID and base choice
	bases    map[string]*baseState // by repository root and commit
	order    []string              // the cached bases, oldest first
	landings map[string]string     // the last landing by repository root and tip
}

type cachedView struct {
	etag string
	body []byte
}

// maxBases bounds the cached base projects, across all projects; maxViews
// the cached documents and landings, which start over when full.
const (
	maxBases = 8
	maxViews = 64
)

func newModels() *models {
	return &models{views: map[string]cachedView{}, bases: map[string]*baseState{}, landings: map[string]string{}}
}

// listed is a working tree whose file list was read once.
type listed struct {
	Workspace
	files []string
}

func (l listed) Files() ([]string, error) { return slices.Clone(l.files), nil }

// view returns the document for a project and base choice, with an ETag that
// changes whenever the document does.
func (c *models) view(ctx context.Context, repos Repos, p Project, choice string) ([]byte, string, error) {
	ws, err := repos.Open(ctx, p.Path)
	if err != nil {
		return nil, "", err
	}
	files, err := ws.Files()
	if err != nil {
		return nil, "", err
	}
	h := sha256.New()
	for _, f := range files {
		fmt.Fprintf(h, "%s\x00%s\x00", f, ws.Stamp(f))
	}
	root := ws.Root()
	var bs bases
	// An unborn HEAD, in a repository with no commit yet, is no error.
	head, err := repos.Commit(ctx, root, "HEAD")
	if ctx.Err() != nil {
		return nil, "", ctx.Err()
	}
	if err == nil {
		bs.Commit = head.ID()
		if bs.Landing, err = c.landing(ctx, repos, root, bs.Commit); err != nil {
			return nil, "", err
		}
	}
	chosen := chooseBase(bs, choice)
	fmt.Fprintf(h, "%s\x00%s\x00%s\x00%v", root, bs.Commit, bs.Landing, chosen)
	etag := `"` + hex.EncodeToString(h.Sum(nil))[:32] + `"`
	key := p.ID + "\x00" + choice
	c.mu.Lock()
	cached, ok := c.views[key]
	c.mu.Unlock()
	if ok && cached.etag == etag {
		return cached.body, etag, nil
	}

	cur, err := project.Load(listed{ws, files})
	if err != nil {
		return nil, "", err
	}
	var v any = struct {
		Initialized bool  `json:"initialized"`
		Bases       bases `json:"bases"`
	}{false, bs}
	if slices.Contains(cur.Files, project.ManifestFile) {
		var b *baseState
		var changed []analysis.Change
		if chosen != nil {
			commit := head
			if chosen.Commit != head.ID() {
				if commit, err = repos.Commit(ctx, root, chosen.Commit); err != nil {
					return nil, "", err
				}
			}
			if b, err = c.base(ctx, root, commit); err != nil {
				return nil, "", err
			}
			if changed, err = commit.Changed(); err != nil {
				return nil, "", err
			}
		}
		v = buildModelView(cur, b, bs, chosen, changed)
	}
	// A request that ended meanwhile may have read files only in part: the
	// loader records a failed read as a diagnostic. Its document is not kept.
	if ctx.Err() != nil {
		return nil, "", ctx.Err()
	}
	body, err := json.Marshal(v)
	if err != nil {
		return nil, "", err
	}
	c.mu.Lock()
	if len(c.views) >= maxViews {
		clear(c.views)
	}
	c.views[key] = cachedView{etag: etag, body: body}
	c.mu.Unlock()
	return body, etag, nil
}

// chooseBase takes the last landing unless the owner asked for the last
// commit or no Run has landed.
func chooseBase(bs bases, choice string) *base {
	switch {
	case bs.Commit == "":
		return nil
	case choice != baseCommit && bs.Landing != "":
		return &base{Kind: baseLanding, Commit: bs.Landing}
	default:
		return &base{Kind: baseCommit, Commit: bs.Commit}
	}
}

func (c *models) landing(ctx context.Context, repos Repos, root, tip string) (string, error) {
	key := root + "\x00" + tip
	c.mu.Lock()
	id, ok := c.landings[key]
	c.mu.Unlock()
	if ok {
		return id, nil
	}
	id, _, err := repos.LastLanded(ctx, root, tip)
	if err != nil {
		return "", err
	}
	c.mu.Lock()
	if len(c.landings) >= maxViews {
		clear(c.landings)
	}
	c.landings[key] = id
	c.mu.Unlock()
	return id, nil
}

func (c *models) base(ctx context.Context, root string, commit Commit) (*baseState, error) {
	key := root + "\x00" + commit.ID()
	c.mu.Lock()
	b := c.bases[key]
	c.mu.Unlock()
	if b != nil {
		return b, nil
	}
	p, err := project.Load(commit)
	if err != nil {
		return nil, err
	}
	// A commit never changes, so a base read in part would stay wrong.
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	b = &baseState{project: p, lint: analysis.Lint(p)}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.bases[key] == nil {
		c.bases[key] = b
		c.order = append(c.order, key)
		if len(c.order) > maxBases {
			delete(c.bases, c.order[0])
			c.order = c.order[1:]
		}
	}
	return c.bases[key], nil
}

// modelRoute answers GET /api/projects/{id}/model?base=landing|commit.
func (s *server) modelRoute(w http.ResponseWriter, r *http.Request) {
	p, ok := s.find(w, r)
	if !ok {
		return
	}
	choice := r.URL.Query().Get("base")
	if choice != "" && choice != baseLanding && choice != baseCommit {
		fail(w, http.StatusBadRequest, fmt.Errorf("unknown base %q: landing or commit", choice))
		return
	}
	body, etag, err := s.models.view(r.Context(), s.repos, p, choice)
	if err != nil {
		fail(w, http.StatusUnprocessableEntity, err)
		return
	}
	// The window sends the last ETag itself; nothing is kept on disk.
	w.Header().Set("ETag", etag)
	if r.Header.Get("If-None-Match") == etag {
		w.WriteHeader(http.StatusNotModified)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(body)
}
