package model

import "fmt"

type bodyResult struct {
	status      []Status
	coordinates Status
	links       map[Identity][]BodyLink
	findings    []identityFinding
}

func checkBodyLinks(scopes []sourceScope, graph dependencyGraph, values valueResult, limits Limits) bodyResult {
	result := bodyResult{status: append([]Status(nil), values.status...), coordinates: Passed, links: map[Identity][]BodyLink{}}
	counts := map[Identity]int{}
	for _, scope := range scopes {
		if !scope.inventoryComplete {
			result.coordinates = NotCompleted
		}
		for _, definition := range scope.definitions {
			counts[identityKey(definition.identity)]++
		}
	}
	for model, scope := range scopes {
		for _, definition := range scope.definitions {
			body, _ := lookup(definition.envelope, "body")
			if body.kind != StringValue {
				result.status[model] = combineStatus(result.status[model], NotCompleted)
				result.coordinates = NotCompleted
				continue
			}
			addFinding := func(code, message string, position *Position) {
				result.findings = append(result.findings, identityFinding{scope: model, diagnostic: Diagnostic{
					Code: code, Message: message, Source: source(scope.origin, definition.file), Position: position,
				}})
			}
			parsed := parseBodySyntax(body.text, limits)
			mapped := append([]bodyOccurrence(nil), parsed.occurrences...)
			needsMap := len(mapped) > 0
			for _, issue := range parsed.problems {
				span := BodySpan{}
				if !issue.incomplete {
					needsMap = true
					span = BodySpan{StartByte: issue.start, EndByte: issue.start + 2}
				}
				mapped = append(mapped, bodyOccurrence{span: span})
			}
			spans := make([]*SourceSpan, len(mapped))
			if needsMap {
				spans = bodySourceSpans(definition.file, body, mapped)
			}
			for index, issue := range parsed.problems {
				var position *Position
				if issue.incomplete {
					result.status[model] = combineStatus(result.status[model], NotCompleted)
				} else {
					result.status[model] = Failed
					if span := spans[len(parsed.occurrences)+index]; span != nil {
						position = &span.Start
					}
				}
				result.coordinates = NotCompleted
				addFinding(issue.code, fmt.Sprintf("body byte %d: %s", issue.start, issue.message), position)
			}
			context := definition.references
			context.complete = context.declarationsBound
			var links []BodyLink
			for index, occurrence := range parsed.occurrences {
				id, status, code, message := resolveAuthorReference(context, occurrence.written, scopes)
				result.status[model] = combineStatus(result.status[model], status)
				var position *Position
				if spans[index] != nil {
					position = &spans[index].Start
				} else {
					result.coordinates = NotCompleted
					addFinding("coordinate.yaml_mapping", fmt.Sprintf("body byte %d: exact YAML source span is not proved; content recognition and binding remain independent.", occurrence.span.StartByte), nil)
				}
				if code != "" {
					addFinding(code, fmt.Sprintf("body byte %d: %s", occurrence.span.StartByte, message), position)
				}
				if status != Passed {
					result.coordinates = NotCompleted
					continue
				}
				span := occurrence.span
				links = append(links, BodyLink{Target: id, Display: occurrence.display, Coordinates: &LinkCoordinates{Body: &span, Source: spans[index]}})
			}
			if counts[identityKey(definition.identity)] == 1 {
				// Missing coordinate metadata never removes recognized, bound
				// occurrences. These are still private, not completed public IR.
				result.links[definition.identity] = links
			}
		}
	}
	closure := dependencyGraph{edges: graph.edges, status: result.status}
	propagateDependencyStatus(&closure)
	return result
}
