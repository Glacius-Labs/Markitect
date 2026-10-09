package projectcli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"

	"github.com/Glacius-Labs/Markitect/internal/host/knowledgeevidence"
	"github.com/Glacius-Labs/Markitect/internal/host/knowledgeprotocol"
	"github.com/Glacius-Labs/Markitect/internal/host/projectapp"
	"github.com/Glacius-Labs/Markitect/internal/host/projectgraph"
)

func runKnowledge(opts options, out io.Writer) error {
	query, err := knowledgeQuery(opts)
	if err != nil {
		return err
	}
	result, err := projectOperations().Knowledge(projectapp.KnowledgeOperation{
		Selection: projectapp.Selection{Root: opts.repo, Revision: opts.revision},
		Scope:     knowledgeScope(opts), Query: query, Records: knowledgeRecords(opts),
	})
	if err != nil {
		return err
	}
	return writeJSON(out, result)
}

func runKnowledgeMCP(opts options, out io.Writer) error {
	operations := projectOperations()
	selection := projectapp.Selection{Root: opts.repo, Revision: opts.revision}
	return knowledgeprotocol.Serve(os.Stdin, out, func(request knowledgeprotocol.ToolRequest) (any, error) {
		return operations.Knowledge(projectapp.KnowledgeOperation{
			Selection: selection, Scope: request.Scope, Query: request.Query, Records: request.Records,
		})
	})
}

func knowledgeScope(opts options) projectgraph.Selection {
	return projectgraph.Selection{ManagerID: opts.manager, ProjectScope: opts.knowledgeScope == "project"}
}

func knowledgeRecords(opts options) knowledgeevidence.Selection {
	selection := knowledgeevidence.Selection{IncludeBriefingHistory: opts.knowledgeBriefingHistory}
	if opts.knowledgeRunID != "" {
		selection.RunIDs = []string{opts.knowledgeRunID}
	}
	if opts.knowledgeExplorationID != "" {
		selection.ExplorationIDs = []string{opts.knowledgeExplorationID}
	}
	if opts.knowledgeSessionID != "" {
		selection.BrownfieldSessionIDs = []string{opts.knowledgeSessionID}
	}
	return selection
}

func knowledgeQuery(opts options) (projectgraph.Request, error) {
	actions := map[string]projectgraph.Action{
		"graph": projectgraph.ActionGraph, "relations": projectgraph.ActionRelations,
		"explain": projectgraph.ActionExplain, "trace": projectgraph.ActionTrace,
		"history": projectgraph.ActionHistory, "coverage": projectgraph.ActionCoverage,
	}
	action, ok := actions[opts.knowledgeAction]
	if !ok {
		return projectgraph.Request{}, errors.New("project knowledge action is unsupported")
	}
	request := projectgraph.Request{Action: action, TargetID: opts.knowledgeNodeID, Reverse: opts.knowledgeReverse, Bidirectional: opts.knowledgeBidirectional}
	if action != projectgraph.ActionTrace {
		return request, nil
	}
	for _, bound := range []struct {
		flag string
		text string
		dest *int
	}{{"max-depth", opts.knowledgeMaxDepth, &request.MaxDepth}, {"max-steps", opts.knowledgeMaxSteps, &request.MaxSteps}, {"max-results", opts.knowledgeMaxResults, &request.MaxResults}} {
		if bound.text == "" {
			continue
		}
		value, err := strconv.Atoi(bound.text)
		if err != nil || value <= 0 {
			return projectgraph.Request{}, fmt.Errorf("--%s must be a positive integer", bound.flag)
		}
		*bound.dest = value
	}
	return request, nil
}
