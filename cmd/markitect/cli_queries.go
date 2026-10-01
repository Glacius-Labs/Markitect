package main

import (
	"fmt"

	"github.com/Glacius-Labs/Markitect/internal/app"
	"github.com/Glacius-Labs/Markitect/internal/core"
	"github.com/Glacius-Labs/Markitect/internal/source"
)

func runInventory(o commandOptions, emit func(any) int, fail func(error) int) int {
	snap, err := source.Load(o.root, o.revision)
	if err != nil {
		return fail(err)
	}
	items := app.MarkdownInventory(snap)
	coverage := "ordinary Markdown candidates only; no resource classification, semantic dependency inference or validity claim"
	if _, ok := snap.Files["markitect.yaml"]; ok {
		p, err := app.Parse(snap)
		if err != nil {
			return fail(err)
		}
		items = append(p.Inventory, items...)
		coverage = "typed canonical resources and ordinary Markdown candidates; generated views are not counted twice"
	}
	return emit(report{Tool: "Markitect", Version: version, Revision: snap.Revision, Provisional: snap.Provisional, Digest: snap.Digest(), Status: "inventory", Coverage: coverage, Inventory: items})
}

func runFind(o commandOptions, p *app.Project, toolDigest string, emit func(any) int, fail func(error) int) int {
	matches, err := app.Find(p, app.FindQuery{Query: o.query, Kind: o.kind, Namespace: o.namespace, Package: o.packageName})
	if err != nil {
		return fail(err)
	}
	return emit(queryEnvelope{Version: version, ToolDigest: toolDigest, Revision: p.Snapshot.Revision, Provisional: p.Snapshot.Provisional, SnapshotDigest: p.Snapshot.Digest(), Result: matches})
}

func runExplain(o commandOptions, p *app.Project, toolDigest string, emit func(any) int, fail func(error) int) int {
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
	key := "/" + o.kind + "/" + o.name
	if o.namespace != "" {
		key = o.namespace + "/" + o.kind + "/" + o.name
	}
	if o.packageName != "" {
		key = o.packageName + "::" + key
	}
	explanation, err := app.Explain(p, key)
	if err != nil {
		return fail(err)
	}
	return emit(queryEnvelope{Version: version, ToolDigest: toolDigest, Revision: p.Snapshot.Revision, Provisional: p.Snapshot.Provisional, SnapshotDigest: p.Snapshot.Digest(), Result: explanation})
}

func runContext(o commandOptions, p *app.Project, toolDigest string, emit func(any) int, fail func(error) int) int {
	if o.kind == "" || o.name == "" || o.namespace == "" {
		return fail(fmt.Errorf("context requires --kind, --name and --namespace"))
	}
	key := (core.Ref{Package: o.packageName, Namespace: o.namespace, Kind: o.kind, Name: o.name}).GraphKey("", "", "")
	c, err := app.CompileContext(p, key, version, toolDigest)
	if err != nil {
		return fail(err)
	}
	return emit(c)
}

func runImpact(o commandOptions, p *app.Project, emit func(any) int, fail func(error) int) int {
	if o.base == "" || o.revision == "" {
		return fail(fmt.Errorf("impact requires fixed --base and --revision"))
	}
	previous, err := app.Load(o.root, o.base)
	if err != nil {
		return fail(err)
	}
	if len(previous.Diagnostics) > 0 {
		return fail(fmt.Errorf("base has unresolved diagnostics; inspect the base separately"))
	}
	return emit(app.Changes(previous, p))
}
