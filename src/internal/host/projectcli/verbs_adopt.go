package projectcli

import (
	"context"
	"io"
	"os"
	"os/signal"

	"github.com/Glacius-Labs/Markitect/src/internal/host/mcp"
	"github.com/Glacius-Labs/Markitect/src/internal/host/projectapp"
)

// --- Adoption -------------------------------------------------------------

type adoptInput struct {
	Action   string                      `json:"action"`
	Session  string                      `json:"session,omitempty"`
	Revision string                      `json:"revision,omitempty"`
	Input    *projectapp.BrownfieldInput `json:"input,omitempty"`
	Expect   string                      `json:"expect,omitempty"`
	Write    bool                        `json:"write,omitempty"`
	Execute  bool                        `json:"execute,omitempty"`
}

var adoptVerb = define(verb{
	name: "adopt", group: "Adoption", effect: effectWrite, interrupt: true, inspect: "adopt status",
	summary:  "Adopt an existing repository into the project model, one session stage at a time.",
	synopsis: "adopt STAGE [--session ID] [--source-repo PATH] [--revision R] [--input FILE] [--expect DIGEST] [--write] [--execute]",
	notes:    "--input holds the stage's record under the stage's key, such as {\"start\": {...}}. Writes compare and swap against the session digest; adopt run --write --execute takes the run preview's previewDigest. MCP uses the server root as the source repository.",
	recovery: "Inspect the session with `adopt status`; correct the stage input and refresh that stage's preview and digest before writing.",
	subs: []subVerb{
		{name: "start", summary: "Discover the source and create a session; needs --revision.", effect: effectWrite, readOnly: true},
		{name: "begin", summary: "Add a reverse iteration for a Manager.", effect: effectWrite, readOnly: true},
		{name: "context", summary: "Show a Manager's proposal or integration context.", effect: effectRead, readOnly: true},
		{name: "propose", summary: "Record a Manager proposal.", effect: effectWrite, readOnly: true},
		{name: "integrate", summary: "Record the integration of child proposals.", effect: effectWrite, readOnly: true},
		{name: "iterate", summary: "Begin, propose and optionally integrate in one step.", effect: effectWrite, readOnly: true},
		{name: "run", summary: "Run a Manager agent for the propose or integrate phase.", effect: effectExecuteWrite, readOnly: true},
		{name: "resolve", summary: "Build and seal the resolution from human choices.", effect: effectWrite, readOnly: true},
		{name: "plan", summary: "Show the adoption plan.", effect: effectRead, readOnly: true},
		{name: "apply", summary: "Apply the reviewed plan to the target model and record the receipt.", effect: effectWrite},
		{name: "status", summary: "Show the session with readiness, scopes, conflicts and coverage.", effect: effectRead, readOnly: true},
	},
	args: []arg{argRepo,
		{name: "session", value: "ID", help: "Adoption session; every stage except start needs it."},
		{name: "source-repo", value: "PATH", cliOnly: true, help: "Source repository; defaults to --repo. MCP always uses the server root."},
		argRevision,
		{name: "input", kind: kindRecord, value: "FILE", help: "The stage's record, read relative to the source repository."},
		argExpect, argWrite, argExecute},
}, func(ctx context.Context, e env, in adoptInput) (any, error) {
	input := projectapp.BrownfieldInput{}
	if in.Input != nil {
		input = *in.Input
	}
	if in.Action == "run" {
		if input.Run == nil {
			return nil, usagef("adopt run requires --input with a \"run\" record")
		}
		return e.ops.BrownfieldRun(ctx, projectapp.BrownfieldRunOperation{
			Root: e.root, SourceRoot: e.sourceRoot, Revision: in.Revision, SessionID: in.Session,
			Request: *input.Run, Write: in.Write, ExpectedDigest: in.Expect,
		}, e.ops.Invoker)
	}
	if in.Execute {
		return nil, usagef("--execute applies only to adopt run")
	}
	return e.ops.Brownfield(projectapp.BrownfieldOperation{
		Root: e.root, SourceRoot: e.sourceRoot, Revision: in.Revision, SessionID: in.Session,
		Action: in.Action, Input: input, Write: in.Write, ExpectedDigest: in.Expect,
	})
})

// --- Information ----------------------------------------------------------

var (
	versionVerb  = verb{name: "version", group: "Information", effect: effectInfo, cliOnly: true, summary: "Print the Markitect version.", synopsis: "version"}
	licensesVerb = verb{name: "licenses", group: "Information", effect: effectInfo, cliOnly: true, summary: "Print third-party license notices.", synopsis: "licenses"}
	helpVerb     = verb{name: "help", group: "Information", effect: effectInfo, cliOnly: true, summary: "List the verbs, or show one verb's arguments.", synopsis: "help [VERB]"}
)

// verbTable lists every verb in help order.
func verbTable() []verb {
	return []verb{
		initVerb, configVerb, onboardVerb, doctorVerb, mcpVerb,
		schemaVerb, checkVerb, modelVerb, contextVerb, impactVerb, docsVerb, editVerb,
		exploreVerb, readyVerb, briefVerb,
		planVerb, runVerb, resumeVerb, repairVerb, statusVerb, verifyVerb, applyVerb, deliverVerb,
		adoptVerb,
		versionVerb, licensesVerb, helpVerb,
	}
}

const readOnlyInstructions = "This server is read-only: it serves reads and previews, and it writes nothing and starts no agents or checks, whatever the arguments. Tools use the explicitly selected repository. Only listed operations are available."

// newMCPServer registers every MCP verb on a server fixed to one root. MCP
// never takes the root or the adoption source from tool arguments.
func newMCPServer(e env) (*mcp.Server, error) {
	server, err := mcp.New(e.root)
	if err != nil {
		return nil, err
	}
	e.root, e.sourceRoot = server.Root(), server.Root()
	for _, v := range verbTable() {
		if v.mcpVerb(e.readOnly) {
			v.register(server, e, e.readOnly)
		}
	}
	if e.readOnly {
		server.SetInstructions(readOnlyInstructions)
	}
	return server, nil
}

// serveMCP owns stdio shutdown. stdout carries protocol messages only.
func serveMCP(e env, out io.Writer) error {
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
	server, err := newMCPServer(e)
	if err != nil {
		return err
	}
	return server.Serve(ctx, os.Stdin, out)
}
