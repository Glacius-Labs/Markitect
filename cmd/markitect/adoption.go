package main

import (
	"fmt"
	"path/filepath"

	"github.com/Glacius-Labs/Markitect/internal/adoption"
	"github.com/Glacius-Labs/Markitect/internal/app"
	"github.com/Glacius-Labs/Markitect/internal/copyme"
)

func runPrepare(o commandOptions, emit func(any) int, fail func(error) int) int {
	if o.scope == "" || o.output == "" || (!o.write && o.expect != "") || (o.write && !adoption.ValidHash(o.expect)) {
		return fail(fmt.Errorf("prepare requires --scope and --output; preview first, then --write --expect HANDOFF_DIGEST"))
	}
	data, err := app.ReadAdoptionRecord(o.scope)
	if err != nil {
		return fail(err)
	}
	var scope adoption.Scope
	if err = adoption.Decode(data, &scope); err != nil {
		return fail(err)
	}
	result, err := app.PrepareAdoption(scope, o.output, o.expect, o.write)
	if err != nil {
		if result != nil {
			if code := emit(result); code != 0 {
				return code
			}
		}
		return fail(err)
	}
	return emit(result)
}

func runCopyMe(o commandOptions, emit func(any) int, fail func(error) int) int {
	if o.workspace == "" || o.queue == "" {
		return fail(fmt.Errorf("copy-me requires --workspace and --queue; --decision is an optional exact review record"))
	}
	handoff, blobs, err := app.ReadAdoptionWorkspace(o.workspace)
	if err != nil {
		return fail(err)
	}
	queueBytes, err := app.ReadAdoptionRecord(o.queue)
	if err != nil {
		return fail(err)
	}
	var queue copyme.Queue
	if err = adoption.Decode(queueBytes, &queue); err != nil {
		return fail(err)
	}
	candidateBytes := map[string][]byte{}
	for _, c := range queue.Candidates {
		if _, ok := candidateBytes[c.Path]; ok {
			return fail(fmt.Errorf("duplicate candidate path %s", c.Path))
		}
		data, err := app.ReadAdoptionCandidate(filepath.Dir(o.queue), c.Path)
		if err != nil {
			return fail(err)
		}
		candidateBytes[c.Path] = data
	}
	var decisionBytes []byte
	if o.decision != "" {
		decisionBytes, err = app.ReadAdoptionRecord(o.decision)
		if err != nil {
			return fail(err)
		}
	}
	report, err := copyme.Validate(handoff, blobs, queueBytes, candidateBytes, decisionBytes)
	if err != nil {
		return fail(err)
	}
	return emit(report)
}
