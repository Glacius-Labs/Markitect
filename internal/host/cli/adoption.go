package cli

import (
	"fmt"
	"path/filepath"

	"github.com/Glacius-Labs/Markitect/internal/host"
	"github.com/Glacius-Labs/Markitect/internal/modules/adoption/capture"
	"github.com/Glacius-Labs/Markitect/internal/modules/adoption/review"
)

func runPrepare(o commandOptions, emit func(any) int, fail func(error) int) int {
	if o.scope == "" || o.output == "" || (!o.write && o.expect != "") || (o.write && !capture.ValidHash(o.expect)) {
		return fail(fmt.Errorf("prepare requires --scope and --output; preview first, then --write --expect HANDOFF_DIGEST"))
	}
	data, err := host.ReadAdoptionRecord(o.scope)
	if err != nil {
		return fail(err)
	}
	var scope capture.Scope
	if err = capture.Decode(data, &scope); err != nil {
		return fail(err)
	}
	result, err := host.PrepareAdoption(scope, o.output, o.expect, o.write)
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
	handoff, blobs, err := host.ReadAdoptionWorkspace(o.workspace)
	if err != nil {
		return fail(err)
	}
	queueBytes, err := host.ReadAdoptionRecord(o.queue)
	if err != nil {
		return fail(err)
	}
	var queue review.Queue
	if err = capture.Decode(queueBytes, &queue); err != nil {
		return fail(err)
	}
	candidateBytes := map[string][]byte{}
	for _, c := range queue.Candidates {
		if _, ok := candidateBytes[c.Path]; ok {
			return fail(fmt.Errorf("duplicate candidate path %s", c.Path))
		}
		data, err := host.ReadAdoptionCandidate(filepath.Dir(o.queue), c.Path)
		if err != nil {
			return fail(err)
		}
		candidateBytes[c.Path] = data
	}
	var decisionBytes []byte
	if o.decision != "" {
		decisionBytes, err = host.ReadAdoptionRecord(o.decision)
		if err != nil {
			return fail(err)
		}
	}
	report, err := review.Validate(handoff, blobs, queueBytes, candidateBytes, decisionBytes)
	if err != nil {
		return fail(err)
	}
	return emit(report)
}
