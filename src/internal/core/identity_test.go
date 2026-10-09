package core

import "testing"

func TestDefinitionIdentityValidityMatchesCompiler(t *testing.T) {
	schema := Schema{APIVersion: "identity.example.org/v1", Purpose: "Identity contract fixture.",
		Kinds: map[string]Kind{"Statement": {Purpose: "Identity-only statement.", Properties: map[string]Property{}}}}
	valid := DefinitionIdentity{APIVersion: schema.APIVersion, Kind: "Statement", Name: "notification"}
	cases := []struct {
		name   string
		mutate func(*DefinitionIdentity)
		want   bool
	}{
		{"global", func(*DefinitionIdentity) {}, true},
		{"named", func(i *DefinitionIdentity) { i.Namespace = "example.notifications" }, true},
		{"missing API version", func(i *DefinitionIdentity) { i.APIVersion = "" }, false},
		{"malformed API version", func(i *DefinitionIdentity) { i.APIVersion = "invalid" }, false},
		{"missing kind", func(i *DefinitionIdentity) { i.Kind = "" }, false},
		{"malformed kind", func(i *DefinitionIdentity) { i.Kind = "bad kind" }, false},
		{"missing name", func(i *DefinitionIdentity) { i.Name = "" }, false},
		{"malformed name", func(i *DefinitionIdentity) { i.Name = "bad/name" }, false},
		{"malformed namespace", func(i *DefinitionIdentity) { i.Namespace = "bad/namespace" }, false},
		{"whitespace namespace", func(i *DefinitionIdentity) { i.Namespace = " " }, false},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			identity := valid
			test.mutate(&identity)
			definition := Definition{APIVersion: identity.APIVersion, Kind: identity.Kind,
				Metadata: Metadata{Namespace: identity.Namespace, Name: identity.Name},
				Purpose:  "Test structural identity.", Spec: map[string]any{}}
			_, diagnostics := Compile([]Schema{schema}, []Definition{definition}, "identity-regression")
			if identity.Valid() != test.want || (len(diagnostics) == 0) != test.want {
				t.Fatalf("identity=%+v Valid=%v diagnostics=%v wantValid=%v", identity, identity.Valid(), diagnostics, test.want)
			}
		})
	}
}
