package authoring

import (
	"strings"
	"testing"
)

func TestProjectPolicyExceptionContractAndGeneratedSchema(t *testing.T) {
	digest := "sha256:" + strings.Repeat("a", 64)
	source := []byte(`apiVersion: markitect.example.org/v1alpha1
kind: Project
metadata: {name: sample}
spec:
  policyDate: "2026-10-01"
  policyExceptions:
    - name: migration-window
      apiVersion: software.example.org/v1
      constraint: usecase-handler
      subject: engineering/software.example.org/v1/UseCase/export
      constraintDigest: ` + digest + `
      subjectDigest: ` + digest + `
      rationale: A separate migration is scheduled.
      owner: architecture
      decision: accepted for this snapshot
      expiresOn: "2026-10-02"
`)
	resource, err := Parse("markitect.yaml", source)
	if err != nil {
		t.Fatal(err)
	}
	if resource.Spec.PolicyDate != "2026-10-01" || len(resource.Spec.PolicyExceptions) != 1 || resource.Spec.PolicyExceptions[0].ExpiresOn != "2026-10-02" {
		t.Fatalf("Project policy config was not normalized: %#v", resource.Spec)
	}
	encoded, err := Encode(resource)
	if err != nil {
		t.Fatal(err)
	}
	reparsed, err := Parse("markitect.yaml", encoded)
	if err != nil || reparsed.Spec.PolicyExceptions[0].Decision != "accepted for this snapshot" {
		t.Fatalf("Project policy exception roundtrip failed: %v, %s", err, encoded)
	}
	schemas, err := Schemas()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(schemas["schema/Project.yaml"]), "policyExceptions") || !strings.Contains(string(schemas["schema/Project.yaml"]), "sha256:[0-9a-f]{64}") {
		t.Fatalf("Project schema omits the policy exception contract:\n%s", schemas["schema/Project.yaml"])
	}
}

func TestProjectPolicyExceptionParserRejectsUnsafeBindings(t *testing.T) {
	digest := "sha256:" + strings.Repeat("b", 64)
	valid := `apiVersion: markitect.example.org/v1alpha1
kind: Project
metadata: {name: sample}
spec:
  policyExceptions:
    - name: migration-window
      apiVersion: software.example.org/v1
      constraint: usecase-handler
      subject: engineering/software.example.org/v1/UseCase/export
      constraintDigest: ` + digest + `
      subjectDigest: ` + digest + `
      rationale: A separate migration is scheduled.
      owner: architecture
      decision: accepted for this snapshot
      expiresOn: 2026-10-02
`
	for _, test := range []struct {
		name   string
		source string
	}{
		{name: "expiry without pinned policy date", source: valid},
		{name: "invalid digest", source: strings.Replace(strings.Replace(valid, "spec:\n", "spec:\n  policyDate: \"2026-10-01\"\n", 1), digest, "sha256:bad", 1)},
		{name: "missing decision", source: strings.Replace(strings.Replace(valid, "spec:\n", "spec:\n  policyDate: \"2026-10-01\"\n", 1), "      decision: accepted for this snapshot\n", "", 1)},
	} {
		t.Run(test.name, func(t *testing.T) {
			if _, err := Parse("markitect.yaml", []byte(test.source)); err == nil {
				t.Fatal("expected policy exception parser rejection")
			}
		})
	}
}
