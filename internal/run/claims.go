package run

import (
	"fmt"
	"slices"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/organization"
	"github.com/Glacius-Labs/Markitect/internal/realization"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

// Claim is a path reservation or a registry proposal that a task's concluded
// writing step made with reserve_path or register (core rule S4). The run
// records them, so that a restarted Host forms the same claims; a failed step's
// claims are dropped with it (core rule T6). A reservation of a path that the
// aggregate registers constrains nothing more, and a proposal equal to the
// aggregate's record proposes nothing.
type Claim struct {
	Tool     string
	Path     string
	Form     string
	Realizes []string `json:",omitempty"`
	Why      string   `json:",omitempty"`
}

type claimInput struct {
	Path     string   `json:"path"`
	Form     string   `json:"form"`
	Realizes []string `json:"realizes"`
	Why      string   `json:"why"`
}

// decodeClaim reads a recorded reserve_path or register call's input.
func decodeClaim(r SessionRecord, input Payload) *Claim {
	if r.Tool != ToolReservePath && r.Tool != ToolRegister || r.Wire.Redacted {
		return nil
	}
	args, err := CallArguments(r.Tool, input.Bytes())
	if err != nil {
		return nil
	}
	if r.Tool == ToolReservePath {
		var in struct {
			Path string `json:"path"`
			Form string `json:"form"`
		}
		if decodeArgs(args, &in) != nil {
			return nil
		}
		return &Claim{Tool: r.Tool, Path: in.Path, Form: in.Form}
	}
	var in claimInput
	if decodeArgs(args, &in) != nil {
		return nil
	}
	return &Claim{Tool: r.Tool, Path: in.Path, Form: in.Form, Realizes: slices.Clone(in.Realizes), Why: in.Why}
}

// stepClaims returns the answered reserve_path and register calls of the
// task's current step, in call order.
func (s stateValue) stepClaims(t taskValue) []Claim {
	var calls []ToolView
	for key, tool := range s.tools {
		if key.Session == t.Session && t.Step != "" && tool.Record.Step == t.Step && tool.Claim != nil && tool.Replied.BatchID != "" && tool.OK {
			calls = append(calls, tool)
		}
	}
	slices.SortFunc(calls, func(a, b ToolView) int { return compareUint(a.Requested.Revision, b.Requested.Revision) })
	var out []Claim
	for _, call := range calls {
		out = append(out, *call.Claim)
	}
	return out
}

// addClaims adds a step's claims: a reservation once per path, and the latest
// proposal per path.
func addClaims(existing, added []Claim) []Claim {
	out := slices.Clone(existing)
	for _, c := range added {
		i := slices.IndexFunc(out, func(x Claim) bool { return x.Tool == c.Tool && x.Path == c.Path })
		switch {
		case i < 0:
			out = append(out, c)
		case c.Tool == ToolRegister:
			out[i] = c
		}
	}
	return out
}

// Claims returns the claims a task's concluded steps made.
func (c *Coordinator) Claims(manager string) []Claim {
	t, ok := c.state.tasks[manager]
	if !ok {
		return nil
	}
	return cloneClaims(t.Claims)
}

func cloneClaims(claims []Claim) []Claim {
	out := slices.Clone(claims)
	for i := range out {
		out[i].Realizes = slices.Clone(out[i].Realizes)
	}
	return out
}

// claimsOf is a Manager's recorded claims and the ones its running step made.
func (e *Engine) claimsOf(manager string, s *stepRun) []Claim {
	claims := e.cfg.Coordinator.Claims(manager)
	if s != nil && s.manager == manager {
		claims = addClaims(claims, s.pending)
	}
	return claims
}

// holderHint names the other Manager whose reservation holds a path, so that a
// refused Manager can wait for that Manager's delivery, which brings the file
// into its workspace, or ask it (core rules S4, M1, T10).
func (e *Engine) holderHint(s *stepRun, path string) string {
	registry := e.aggregate().Registry()
	for _, r := range registry.Entries() {
		if p := r.Artifact().Path().String(); pathsOverlap(p, path) {
			return fmt.Sprintf(" %s is registered to the Manager %s and already in your workspace; use it as it is, or send that Manager the change you need", p, ownerManager(r.Owner()))
		}
	}
	for _, t := range e.cfg.Coordinator.Tasks() {
		if t.Manager == s.manager {
			continue
		}
		for _, c := range e.claimsOf(t.Manager, s) {
			if c.Tool == ToolReservePath && pathsOverlap(c.Path, path) {
				return fmt.Sprintf(" The Manager %s reserved %s; once its delivery is accepted, the file is in your workspace and you may use it without changing it. Wait for it with {\"on\":[{\"kind\":\"delivery\",\"ref\":%q}]} or send it a question", t.Manager, c.Path, t.Manager)
			}
		}
	}
	return ""
}

// pathsOverlap reports whether two repository paths are equal or one lies
// below the other.
func pathsOverlap(a, b string) bool {
	return a == b || strings.HasPrefix(a, b+"/") || strings.HasPrefix(b, a+"/")
}

// ownerManager names the Manager of an artifact owner.
func ownerManager(o realization.Owner) string {
	if name, ok := o.Package(); ok {
		return name
	}
	return organization.Root
}

func requestOf(manager string, c Claim) (realization.PathRequest, error) {
	owner, err := managerOwner(manager)
	if err != nil {
		return realization.PathRequest{}, err
	}
	path, err := snapshot.NewPath(c.Path)
	if err != nil {
		return realization.PathRequest{}, fmt.Errorf("invalid path")
	}
	request, err := realization.NewPathRequest(realization.ClaimID(manager+":"+c.Path), owner, path, realization.Form(c.Form))
	if err != nil {
		return realization.PathRequest{}, fmt.Errorf("invalid reservation")
	}
	return request, nil
}

func registrationOf(manager string, c Claim) (realization.Registration, error) {
	owner, err := managerOwner(manager)
	if err != nil {
		return realization.Registration{}, err
	}
	path, err := snapshot.NewPath(c.Path)
	if err != nil {
		return realization.Registration{}, fmt.Errorf("invalid path")
	}
	var basis []model.Identity
	for _, d := range c.Realizes {
		id, found := definitionIdentity(d)
		if !found {
			return realization.Registration{}, fmt.Errorf("realizes must name Definitions as package/id")
		}
		basis = append(basis, id)
	}
	artifact, err := realization.NewArtifact(path, realization.Form(c.Form), basis, c.Why)
	if err != nil {
		return realization.Registration{}, fmt.Errorf("invalid registry record: %w", err)
	}
	record, err := realization.NewRegistration(owner, artifact)
	if err != nil {
		return realization.Registration{}, fmt.Errorf("invalid registry record: %w", err)
	}
	return record, nil
}

// freshClaims forms the run-wide claim set at the current aggregate version
// from every task's reservations, root first, that the aggregate does not yet
// register.
func (e *Engine) freshClaims(s *stepRun) (realization.PathClaims, error) {
	facts := e.aggregate().PathFacts()
	registry := e.aggregate().Registry()
	var claims realization.PathClaims
	for _, t := range e.cfg.Coordinator.Tasks() {
		for _, c := range e.claimsOf(t.Manager, s) {
			if c.Tool != ToolReservePath {
				continue
			}
			request, err := requestOf(t.Manager, c)
			if err != nil {
				return realization.PathClaims{}, err
			}
			if _, registered := registry.Lookup(request.Path()); registered {
				continue
			}
			next, decision := realization.ReserveNewPath(facts, claims, request)
			if decision.Status() != realization.PathAdmissible {
				return realization.PathClaims{}, fmt.Errorf("reservation of %s no longer fits: %s", request.Path(), findingText(decision.Findings()))
			}
			claims = next
		}
	}
	return claims, nil
}

// proposals are a Manager's registry proposals that differ from the
// aggregate's records.
func (e *Engine) proposals(manager string, s *stepRun) (map[snapshot.Path]realization.Registration, error) {
	registry := e.aggregate().Registry()
	out := map[snapshot.Path]realization.Registration{}
	for _, c := range e.claimsOf(manager, s) {
		if c.Tool != ToolRegister {
			continue
		}
		record, err := registrationOf(manager, c)
		if err != nil {
			return nil, err
		}
		path := record.Artifact().Path()
		if current, ok := registry.Lookup(path); ok && sameRecord(current, record) {
			continue
		}
		out[path] = record
	}
	return out, nil
}

func sameRecord(a, b realization.Registration) bool {
	x, y := a.Artifact(), b.Artifact()
	return a.Owner() == b.Owner() && x.Path() == y.Path() && x.Form() == y.Form() && x.Why() == y.Why() && slices.Equal(x.Basis(), y.Basis())
}
