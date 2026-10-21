package apply

import (
	"crypto/sha256"
	"fmt"
	"strings"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/organization"
	"github.com/Glacius-Labs/Markitect/internal/run"
)

// RunBinding is what an Apply binds its Run to besides the captured stand and
// the request: the strategy with its work sequence, the effective settings and
// the implementation identity. Apply and VerifyCaptured resolve it the same way
// from one passed preparation before any Agent work; run.start records it and
// the report repeats it. Head is the captured source revision and Request the
// digest of the owner's request. Entry names the entry that ran: "apply" or
// "verify-captured", which runs only the strategy's Verify and so no work
// Pass. Composition is the digest of the strategy's Pass declarations and
// Root duty as this build implements them: it tells builds apart that the
// build identity alone does not, such as a modified tree.
type RunBinding struct {
	Entry       string           `json:"entry"`
	Head        string           `json:"head"`
	Request     string           `json:"request"`
	Strategy    Strategy         `json:"strategy"`
	Work        []run.Purpose    `json:"work"`
	Corrections []run.Purpose    `json:"corrections"`
	Composition string           `json:"composition"`
	Build       string           `json:"build"`
	Execution   string           `json:"execution"`
	Defaults    string           `json:"defaults"`
	Budget      BudgetBinding    `json:"budget"`
	Managers    []ManagerBinding `json:"managers"`
	Checks      []CheckBinding   `json:"checks"`
}

// BudgetBinding is the authored budget with the run's effective deadline.
type BudgetBinding struct {
	ExecutionTime string    `json:"executionTime"`
	TurnTimeout   string    `json:"turnTimeout"`
	AgentTurns    int       `json:"agentTurns"`
	Passes        int       `json:"passes"`
	Deadline      time.Time `json:"deadline"`
}

// ManagerBinding is one Manager's resolved roles, team and model context.
type ManagerBinding struct {
	Manager    string          `json:"manager"`
	Execute    RoleBinding     `json:"execute"`
	Verify     RoleBinding     `json:"verify"`
	Helpers    []HelperBinding `json:"helpers"`
	MaxHelpers int             `json:"maxHelpers"`
	Context    ContextBinding  `json:"context"`
}

// ContextBinding is the coverage record of a Manager's standing Brief: its
// digest, the Definitions it states in full and those it only lists for
// reference (Root's project inventory), each by identity. Every other
// Definition of the bound model is available through look_up. Whether a turn
// delivered the Brief, or relied on its conversation, run evidence records per
// turn.
type ContextBinding struct {
	Brief     string   `json:"brief"`
	Full      []string `json:"full"`
	Reference []string `json:"reference"`
}

// boundManager is one Manager's resolved settings with its context record.
type boundManager struct {
	settings organization.ManagerSettings
	context  ContextBinding
}

// RoleBinding is a resolved profile with its prepared identity and the content
// of its instructions.
type RoleBinding struct {
	Profile      string   `json:"profile"`
	Identity     string   `json:"identity"`
	Instructions []string `json:"instructions"`
}

// HelperBinding is one resolved helper role.
type HelperBinding struct {
	Role   string `json:"role"`
	Writes bool   `json:"writes"`
	RoleBinding
}

// CheckBinding is one required check as its request runs.
type CheckBinding struct {
	Name       string `json:"name"`
	RequiredBy string `json:"requiredBy,omitempty"`
	Image      string `json:"image"`
	Command    string `json:"command"`
	Timeout    string `json:"timeout"`
}

// bindRun records the effective values of one passed preparation and the bound
// request.
func bindRun(entry string, prepared CapturedSettingsAttempt, request string, strategy Strategy, build string, budget ExecutionBudget, deadline time.Time, managers []boundManager, checks []run.Check) RunBinding {
	b := RunBinding{Entry: entry, Head: prepared.Structural().Model().Inputs().Head, Request: fmt.Sprintf("sha256:%x", sha256.Sum256([]byte(request))), Strategy: strategy, Corrections: strategy.corrections(), Build: build,
		Budget: BudgetBinding{ExecutionTime: budget.ExecutionTime.Value.String(), TurnTimeout: budget.TurnTimeout.Value.String(), AgentTurns: budget.AgentTurns.Value, Passes: budget.Passes.Value, Deadline: deadline.UTC()}}
	b.Work = []run.Purpose{}
	for _, spec := range strategy.work() {
		if entry == entryApply {
			b.Work = append(b.Work, spec.Purpose)
		}
	}
	b.Composition = strategy.digest()
	if raw := prepared.Structural().Model().Inputs().Raw.Execution; len(raw) == 1 {
		b.Execution = raw[0].RepositoryPath.String() + "@" + raw[0].File.ContentID().String()
	}
	if source := prepared.Defaults().ProductSource(); source.File != "" {
		b.Defaults = source.File + "@" + source.ContentID.String()
	}
	for _, bound := range managers {
		m := bound.settings
		mb := ManagerBinding{Manager: m.Package, Execute: roleBinding(m.Execute), Verify: roleBinding(m.Verify), Helpers: []HelperBinding{}, MaxHelpers: m.MaxHelpers, Context: bound.context}
		for _, h := range m.Helpers {
			mb.Helpers = append(mb.Helpers, HelperBinding{Role: h.Role, Writes: h.Writes, RoleBinding: roleBinding(h.RoleSettings)})
		}
		b.Managers = append(b.Managers, mb)
	}
	b.Checks = []CheckBinding{}
	for _, c := range checks {
		b.Checks = append(b.Checks, CheckBinding{Name: c.Name, RequiredBy: c.RequiredBy, Image: c.Request.ImageID,
			Command: strings.Join(append([]string{c.Request.Executable}, c.Request.Args...), " "), Timeout: c.Request.Timeout.String()})
	}
	return b
}

func roleBinding(r organization.RoleSettings) RoleBinding {
	b := RoleBinding{Profile: r.Profile.Value.Name, Identity: r.Profile.Value.Identity, Instructions: []string{}}
	for _, instruction := range r.Instructions {
		b.Instructions = append(b.Instructions, instruction.Value.Source.File+"@"+instruction.Value.Source.ContentID.String())
	}
	return b
}
