package apply_test

import "testing"

// documentedExecution is the shape the Library walkthrough shows: complete
// project budget and profiles, one exact package override and one check, no
// layout (ownership then rests on the registry shards alone).
func documentedExecution(t *testing.T) string {
	t.Helper()
	return `apiVersion: markitect.execution/v1
strategy: shared-workspace-v1
budget:
  executionTime: 25m
  turnTimeout: 10m
  agentTurns: 120
  rounds: 4
profiles:
  execute: library-agent
  verify: library-agent
packages:
  lending:
    profiles: {verify: library-agent}
checks:
  - id: project-tests
    profile: library-tests
    directory: realization
    args: [test, ./...]
    requiredBy: library/callable-interface
    timeout: 2m
`
}
