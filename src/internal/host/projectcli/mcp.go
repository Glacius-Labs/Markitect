package projectcli

import (
	"context"
	"io"
	"os"
	"os/signal"

	"errors"
	"github.com/Glacius-Labs/Markitect/src/internal/host/mcp"
	"github.com/Glacius-Labs/Markitect/src/internal/host/projectapp"
	"github.com/Glacius-Labs/Markitect/src/internal/host/projectcoverage"
	"github.com/Glacius-Labs/Markitect/src/internal/host/projectexplore"
	"github.com/Glacius-Labs/Markitect/src/internal/host/projectonboarding"
	"github.com/Glacius-Labs/Markitect/src/internal/host/projectsetup"
	"github.com/Glacius-Labs/Markitect/src/internal/host/projectwork"
	"github.com/Glacius-Labs/Markitect/src/internal/modules/projectmodel"
)

// runMCPCommand owns stdio shutdown. All product behavior stays in the shared
// facade; stdout contains protocol messages only, never CLI progress output.
func runMCPCommand(root string, out io.Writer) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	done := make(chan struct{})
	defer close(done)
	go func() {
		select {
		case <-ctx.Done():
			_ = os.Stdin.Close()
		case <-done:
		}
	}()
	server, err := projectMCP(root, projectOperations())
	if err != nil {
		return err
	}
	return server.Serve(ctx, os.Stdin, out)
}

func projectMCP(root string, o projectapp.Operations) (*mcp.Server, error) {
	server, err := mcp.New(root, o)
	if err != nil {
		return nil, err
	}
	// Selection roots are fixed by this composition, never accepted from tool args.
	mcp.Register(server, "project_setup", "Preview or apply a digest-bound runtime edit from explicit local tools.", true, func(ctx context.Context, r mcpSetupInput) (projectsetup.Preview, error) {
		return o.Setup(projectapp.SetupOperation{Root: root, Options: r.Options, Write: r.Write, ExpectedDigest: r.ExpectedDigest})
	})
	mcp.Register(server, "project_doctor", "Inspect selected local tools and authentication prerequisites without starting roles.", false, func(ctx context.Context, r mcpDoctorInput) (projectsetup.DoctorReport, error) {
		return o.Doctor(projectapp.DoctorOperation{Root: root, Options: r.Options})
	})
	mcp.Register(server, "project_onboard", "Preview or install current project-local Codex/Claude guidance without replacing custom content.", true, func(ctx context.Context, r mcpOnboardInput) (projectonboarding.Plan, error) {
		return o.Onboard(projectapp.OnboardOperation{Root: root, Options: r.Options, Write: r.Write, ExpectedDigest: r.ExpectedDigest})
	})
	mcp.Register(server, "project_explore", "List/read or preview/write a typed one-scope Work Item with exact model binding and CAS.", true, func(ctx context.Context, r mcpExploreInput) (projectapp.ExploreResult, error) {
		return o.Explore(projectapp.ExploreOperation{Selection: projectapp.Selection{Root: root, Revision: r.Revision}, ExplorationID: r.ExplorationID, Record: r.Record, Write: r.Write, ExpectedDigest: r.ExpectedDigest})
	})
	mcp.Register(server, "project_readiness", "Inspect current scope or explicitly acknowledge its exact structure under caller authority.", true, func(ctx context.Context, r mcpReadinessInput) (projectapp.ReadinessResult, error) {
		return o.Readiness(projectapp.ReadinessOperation{Selection: projectapp.Selection{Root: root, Revision: r.Revision}, ExplorationID: r.ExplorationID, ScopeID: r.ScopeID, Acknowledgement: r.Acknowledgement, Write: r.Write, ExpectedDigest: r.ExpectedDigest})
	})
	mcp.Register(server, "project_brownfield", "Preview/read or apply a typed fixed-source Brownfield ledger stage; inputs never fabricate adoption receipts.", true, func(ctx context.Context, r mcpBrownfieldInput) (projectapp.BrownfieldResult, error) {
		return o.Brownfield(projectapp.BrownfieldOperation{Root: root, SourceRoot: r.SourceRoot, Revision: r.Revision, SessionID: r.SessionID, Action: r.Action, Input: r.Input, Write: r.Write, ExpectedDigest: r.ExpectedDigest})
	})
	mcp.Register(server, "project_brownfield_run", "Preview or execute an existing Brownfield Manager stage with durable attempt identity and cancellation.", true, func(ctx context.Context, r mcpBrownfieldRunInput) (projectapp.BrownfieldManagerRunOutput, error) {
		return o.BrownfieldRun(ctx, projectapp.BrownfieldRunOperation{Root: root, SourceRoot: r.SourceRoot, SessionID: r.SessionID, Request: r.Request, Write: r.Write, ExpectedDigest: r.ExpectedDigest}, o.Invoker)
	})
	mcp.Register(server, "project_init", "Preview or create the canonical project control plane on a normal feature branch.", true, func(ctx context.Context, r mcpInitInput) (projectwork.InitPlan, error) {
		return o.Init(projectapp.InitOperation{Root: root, Name: r.Name, Write: r.Write})
	})
	mcp.Register(server, "project_check", "Compile exact selected project inputs and return actionable findings/coverage.", false, func(ctx context.Context, r mcpRevisionInput) (projectapp.CheckResult, error) {
		result, err := o.Check(projectapp.Selection{Root: root, Revision: r.Revision})
		if err != nil {
			return result, err
		}
		if result.Report.Status != "succeeded" || (result.Source.CoverageMode == "full" && (result.Coverage == nil || !result.Coverage.Conforming)) {
			return result, errors.New("selected project model or coverage does not conform")
		}
		return result, nil
	})
	mcp.Register(server, "project_index", "Read the exact selected compiled project model and ownership index.", false, func(ctx context.Context, r mcpRevisionInput) (projectmodel.Report, error) {
		return o.Index(projectapp.Selection{Root: root, Revision: r.Revision})
	})
	mcp.Register(server, "project_context", "Read a selected Manager context under accepted-history rules.", false, func(ctx context.Context, r mcpContextInput) (projectmodel.ManagerContext, error) {
		return o.Context(projectapp.ContextOperation{Selection: projectapp.Selection{Root: root, Revision: r.Revision}, ManagerID: r.ManagerID})
	})
	mcp.Register(server, "project_document", "Generate the current readable model document; explicit write retains guarded writer rules.", true, func(ctx context.Context, r mcpDocumentInput) (string, error) {
		return o.Document(projectapp.DocumentOperation{Selection: projectapp.Selection{Root: root, Revision: r.Revision}, Write: r.Write})
	})
	mcp.Register(server, "project_edit", "Preview or apply an exact canonical model mutation under caller authority and expected digest.", true, func(ctx context.Context, r mcpEditInput) (projectwork.EditPlan, error) {
		return o.Edit(projectapp.EditOperation{Selection: projectapp.Selection{Root: root, Revision: r.Revision}, Mutation: r.Mutation, Write: r.Write, ExpectedDigest: r.ExpectedDigest})
	})
	mcp.Register(server, "project_impact", "Compare two explicit revisions with deterministic conservative impact.", false, func(ctx context.Context, r mcpImpactInput) (projectmodel.ChangeImpact, error) {
		return o.Impact(projectapp.ImpactOperation{Root: root, BaseRevision: r.BaseRevision, Revision: r.Revision})
	})
	mcp.Register(server, "project_coverage", "Inspect actual selected repository coverage and declared ownership gaps.", false, func(ctx context.Context, r mcpRevisionInput) (projectcoverage.Report, error) {
		result, err := o.Coverage(projectapp.Selection{Root: root, Revision: r.Revision})
		if err != nil {
			return result, err
		}
		if !result.Conforming {
			return result, errors.New("whole-repository coverage is not conforming")
		}
		return result, nil
	})

	return server, nil
}

type mcpSetupInput struct {
	Options        projectsetup.Options `json:"options"`
	Write          bool                 `json:"write"`
	ExpectedDigest string               `json:"expectedDigest,omitempty"`
}
type mcpDoctorInput struct {
	Options projectsetup.Options `json:"options"`
}
type mcpOnboardInput struct {
	Options        projectonboarding.Options `json:"options"`
	Write          bool                      `json:"write"`
	ExpectedDigest string                    `json:"expectedDigest,omitempty"`
}
type mcpExploreInput struct {
	Revision       string                 `json:"revision,omitempty"`
	ExplorationID  string                 `json:"explorationId,omitempty"`
	Record         *projectexplore.Record `json:"record,omitempty"`
	Write          bool                   `json:"write"`
	ExpectedDigest string                 `json:"expectedDigest,omitempty"`
}
type mcpReadinessInput struct {
	Revision        string                                    `json:"revision,omitempty"`
	ExplorationID   string                                    `json:"explorationId"`
	ScopeID         string                                    `json:"scopeId"`
	Acknowledgement *projectapp.StructureAcknowledgementInput `json:"acknowledgement,omitempty"`
	Write           bool                                      `json:"write"`
	ExpectedDigest  string                                    `json:"expectedDigest,omitempty"`
}
type mcpBrownfieldInput struct {
	SourceRoot     string                     `json:"sourceRoot,omitempty"`
	Revision       string                     `json:"revision,omitempty"`
	SessionID      string                     `json:"sessionId,omitempty"`
	Action         string                     `json:"action"`
	Input          projectapp.BrownfieldInput `json:"input"`
	Write          bool                       `json:"write"`
	ExpectedDigest string                     `json:"expectedDigest,omitempty"`
}
type mcpBrownfieldRunInput struct {
	SourceRoot     string                               `json:"sourceRoot,omitempty"`
	SessionID      string                               `json:"sessionId"`
	Request        projectapp.BrownfieldManagerRunInput `json:"request"`
	Write          bool                                 `json:"write"`
	ExpectedDigest string                               `json:"expectedDigest,omitempty"`
}

type mcpInitInput struct {
	Name  string `json:"name"`
	Write bool   `json:"write"`
}
type mcpRevisionInput struct {
	Revision string `json:"revision,omitempty"`
}
type mcpContextInput struct {
	Revision  string `json:"revision,omitempty"`
	ManagerID string `json:"managerId"`
}
type mcpDocumentInput struct {
	Revision string `json:"revision,omitempty"`
	Write    bool   `json:"write"`
}
type mcpEditInput struct {
	Revision       string               `json:"revision,omitempty"`
	Mutation       projectwork.Mutation `json:"mutation"`
	Write          bool                 `json:"write"`
	ExpectedDigest string               `json:"expectedDigest,omitempty"`
}
type mcpImpactInput struct {
	BaseRevision string `json:"baseRevision"`
	Revision     string `json:"revision"`
}
