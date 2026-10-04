package cli

import (
	"fmt"

	"github.com/Glacius-Labs/Markitect/internal/core"
	"github.com/Glacius-Labs/Markitect/internal/host"
	"github.com/Glacius-Labs/Markitect/internal/infrastructure/source"
)

func runInventory(o commandOptions, emit func(any) int, fail func(error) int) int {
	snap, err := source.Load(o.root, o.revision)
	if err != nil {
		return fail(err)
	}
	items := host.MarkdownInventory(snap)
	coverage := "ordinary Markdown candidates only; no resource classification, semantic dependency inference or validity claim"
	if _, ok := snap.Files["markitect.yaml"]; ok {
		p, err := host.Parse(snap)
		if err != nil {
			return fail(err)
		}
		items = append(p.Inventory, items...)
		coverage = "typed canonical resources and ordinary Markdown candidates; generated views are not counted twice"
	}
	return emit(report{Tool: "Markitect", Version: version, Revision: snap.ID, Provisional: snap.Provisional, Digest: snap.Digest(), Status: "inventory", Coverage: coverage, Inventory: items})
}

func runFind(o commandOptions, p *host.Project, toolDigest string, emit func(any) int, fail func(error) int) int {
	matches, err := host.Find(p, host.FindQuery{Query: o.query, APIVersion: o.apiVersion, Kind: o.kind, Namespace: o.namespace, Package: o.packageName})
	if err != nil {
		return fail(err)
	}
	return emit(queryEnvelope{Version: version, ToolDigest: toolDigest, Revision: p.Snapshot.ID, Provisional: p.Snapshot.Provisional, SnapshotDigest: p.Snapshot.Digest(), Result: matches})
}

func runExplain(o commandOptions, p *host.Project, toolDigest string, emit func(any) int, fail func(error) int) int {
	if o.kind == "" || o.name == "" {
		return fail(fmt.Errorf("explain requires --kind and --name"))
	}
	if o.kind == "Project" {
		if o.namespace != "" {
			return fail(fmt.Errorf("Project explain identity does not accept --namespace"))
		}
	} else if o.namespace == "" {
		return fail(fmt.Errorf("explain requires --namespace for namespaced resources"))
	}
	key := (core.Ref{APIVersion: o.apiVersion, Kind: o.kind, Name: o.name, Namespace: o.namespace, Package: o.packageName}).GraphKey("", "", "")
	explanation, err := host.Explain(p, key)
	if err != nil {
		return fail(err)
	}
	return emit(queryEnvelope{Version: version, ToolDigest: toolDigest, Revision: p.Snapshot.ID, Provisional: p.Snapshot.Provisional, SnapshotDigest: p.Snapshot.Digest(), Result: explanation})
}

func runContext(o commandOptions, p *host.Project, toolDigest string, emit func(any) int, fail func(error) int) int {
	if o.kind == "" || o.name == "" || o.namespace == "" {
		return fail(fmt.Errorf("context requires --kind, --name and --namespace"))
	}
	key := (core.Ref{APIVersion: o.apiVersion, Package: o.packageName, Namespace: o.namespace, Kind: o.kind, Name: o.name}).GraphKey("", "", "")
	var c *host.Context
	var err error
	if o.analyzePolicyFailures {
		c, err = host.AnalyzeContext(p, key, version, toolDigest)
	} else {
		c, err = host.CompileContext(p, key, version, toolDigest)
	}
	if err != nil {
		return fail(err)
	}
	if code := emit(c); code != 0 {
		return code
	}
	return analysisExitCode(c.Analysis)
}

func runContextManifest(o commandOptions, toolDigest string, emit func(any) int, fail func(error) int) int {
	p, err := host.Load(o.root, o.revision)
	if err != nil {
		return fail(err)
	}
	manifestBytes, ok := p.Snapshot.Files[o.runManifest]
	if !ok {
		return fail(fmt.Errorf("run manifest %q is not a file in the selected Git snapshot", o.runManifest))
	}
	c, err := host.CompileRunContext(p, o.runManifest, manifestBytes, version, toolDigest)
	if err != nil {
		return fail(err)
	}
	if code := emit(c); code != 0 {
		return code
	}
	if !c.Complete {
		return 2
	}
	return 0
}

func runImpact(o commandOptions, p *host.Project, emit func(any) int, fail func(error) int) int {
	if o.base == "" || o.revision == "" {
		return fail(fmt.Errorf("impact requires fixed --base and --revision"))
	}
	previous, err := host.Load(o.root, o.base)
	if err != nil {
		return fail(err)
	}
	if !o.analyzePolicyFailures && len(previous.Diagnostics) > 0 {
		return fail(fmt.Errorf("base has unresolved diagnostics; inspect the base separately"))
	}
	var result *host.Impact
	if o.analyzePolicyFailures {
		result, err = host.AnalyzeImpact(previous, p)
	} else {
		result = host.Changes(previous, p)
	}
	if err != nil {
		return fail(err)
	}
	if code := emit(result); code != 0 {
		return code
	}
	return analysisExitCode(result.Analysis)
}

func analysisExitCode(analysis *host.AnalysisEvidence) int {
	if analysis == nil {
		return 0
	}
	if analysis.Candidate.PolicyStatus == "failed" || analysis.Base != nil && analysis.Base.PolicyStatus == "failed" {
		return 1
	}
	return 0
}
