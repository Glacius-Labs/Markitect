package projectmodel

import "github.com/Glacius-Labs/Markitect/internal/core"

const (
	managerKind        = "Manager"
	statementKind      = "Statement"
	artifactKind       = "Artifact"
	checkKind          = "Check"
	decisionKind       = "Decision"
	identityChangeKind = "IdentityChange"
)

// Schema returns the closed structural contract for project-world definitions.
func Schema() core.Schema {
	api := APIVersion
	statementRef := &core.KindIdentity{APIVersion: api, Kind: statementKind}
	managerRef := &core.KindIdentity{APIVersion: api, Kind: managerKind}
	checkRef := &core.KindIdentity{APIVersion: api, Kind: checkKind}
	decisionRef := &core.KindIdentity{APIVersion: api, Kind: decisionKind}
	str := func(purpose string, min, max int) core.Property {
		return core.Property{Purpose: purpose, Type: core.TypeString, MinCount: min, MaxCount: max}
	}
	ref := func(purpose string, target *core.KindIdentity, min, max int) core.Property {
		return core.Property{Purpose: purpose, Type: core.TypeReference, Target: target, MinCount: min, MaxCount: max}
	}
	enum := func(purpose string, values ...string) core.Property {
		return core.Property{Purpose: purpose, Type: core.TypeEnum, Values: values, MinCount: 1, MaxCount: 1}
	}
	boolean := core.Property{Purpose: "Whether the value is explicitly enabled.", Type: core.TypeBoolean, MinCount: 0, MaxCount: 1}
	pathList := str("Repository-relative exact path or trailing-slash directory prefix.", 0, core.Unbounded)
	return core.Schema{
		APIVersion: api,
		Purpose:    "A project-owned model of managers, statements, expected artifacts, checks, and decisions.",
		Kinds: map[string]core.Kind{
			managerKind: {Purpose: "A manager responsibility and its explicitly delegated repository paths.", Properties: map[string]core.Property{
				"parent":       ref("The directly responsible parent Manager, when this is not the root Manager.", managerRef, 0, 1),
				"owns":         pathList,
				"instructions": str("Local decision and management instructions.", 0, 1),
			}},
			statementKind: {Purpose: "A typed project concept, rule, use case, architecture statement, or workflow.", Properties: map[string]core.Property{
				"category":    enum("Statement category.", "concept", "rule", "use-case", "architecture", "workflow"),
				"description": str("The statement expressed as readable project intent.", 1, 1),
				"public":      boolean,
				"uses":        ref("Statements whose relevant context this statement uses.", statementRef, 0, core.Unbounded),
				"requires":    ref("Statements whose obligations this statement requires.", statementRef, 0, core.Unbounded),
			}},
			artifactKind: {Purpose: "An expected project artifact and its declared realization and checks.", Properties: map[string]core.Property{
				"role":     enum("Artifact role.", "implementation", "documentation", "verification", "configuration"),
				"realizes": ref("Statements this artifact is expected to realize.", statementRef, 0, core.Unbounded),
				"paths":    pathList,
				"checks":   ref("Checks declared for this artifact.", checkRef, 0, core.Unbounded),
				"required": boolean,
				"reason":   str("Reason the artifact is required or applicable.", 0, 1),
			}},
			checkKind: {Purpose: "A declared literal command and its model scope and limitations.", Properties: map[string]core.Property{
				"command":    str("Literal argv passed to the check executable.", 1, core.Unbounded),
				"uses":       ref("Statements relevant to this check.", statementRef, 0, core.Unbounded),
				"limitation": str("What the check does not establish.", 0, 1),
			}},
			decisionKind: {Purpose: "An explicit decision about a project statement by a manager.", Properties: map[string]core.Property{
				"subject":    ref("Statement this decision addresses.", statementRef, 1, 1),
				"decision":   str("Decision text.", 1, 1),
				"reason":     str("Reason for the decision.", 1, 1),
				"actor":      ref("Manager recorded as the decision actor; this does not authenticate a person.", managerRef, 1, 1),
				"supersedes": ref("Decision explicitly superseded by this decision.", decisionRef, 0, 1),
				"public":     boolean,
			}},
			identityChangeKind: {Purpose: "An explicit historical Statement identity transition claim; previous is historical identity data, not a live reference.", Properties: map[string]core.Property{
				"operation": enum("Declared identity operation.", "renamed", "replaced", "retired"),
				"previous": {Purpose: "Full historical Statement identity. This closed object is not resolved against current definitions.", Type: core.TypeObject, MinCount: 1, MaxCount: 1, Properties: map[string]core.Property{
					"apiVersion": str("Historical identity API version.", 1, 1),
					"kind":       str("Historical identity kind; must be Statement.", 1, 1),
					"namespace":  str("Historical identity namespace; empty denotes the global namespace.", 1, 1),
					"name":       str("Historical identity name.", 1, 1),
				}},
				"subject":      ref("Current Statement explicitly associated with a rename or replacement; absent for retirement.", statementRef, 0, 1),
				"reason":       str("Reason for declaring this identity change.", 1, 1),
				"actorManager": ref("Manager recorded as the actor; this does not authenticate a person.", managerRef, 1, 1),
				"public":       boolean,
			}},
		},
	}
}
