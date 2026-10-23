package project_test

import "testing"

// documentedExecution contains structural project policy only. Runtime
// profiles and invocation limits are supplied outside Project.
func documentedExecution(t *testing.T) string {
	t.Helper()
	return `apiVersion: markitect.execution/v2
checks:
  - id: project-tests
    profile: library-tests
    directory: realization
    args: [test, ./...]
    requiredBy: library/callable-interface
    timeout: 2m
`
}
