package host

import "testing"

func TestDecodeCanonicalSourceConfigIsClosedAndRejectsAliasesAndDuplicates(t *testing.T) {
	valid := []byte("apiVersion: markitect.canonical/v1alpha1\nkind: Source\ndefinitions: [definitions/service.yaml]\nmodules: []\n")
	if _, err := DecodeCanonicalSourceConfig(valid); err != nil {
		t.Fatalf("valid source config: %v", err)
	}
	cases := map[string][]byte{
		"unknown field":   []byte("apiVersion: markitect.canonical/v1alpha1\nkind: Source\ndefinitions: [definitions/service.yaml]\nmodules: []\nsurprise: true\n"),
		"alias":           []byte("apiVersion: markitect.canonical/v1alpha1\nkind: Source\ndefinitions: [&file definitions/service.yaml, *file]\nmodules: []\n"),
		"duplicate field": []byte("apiVersion: markitect.canonical/v1alpha1\nkind: Source\ndefinitions: [definitions/service.yaml]\ndefinitions: [definitions/other.yaml]\nmodules: []\n"),
	}
	for name, input := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := DecodeCanonicalSourceConfig(input); err == nil {
				t.Fatalf("DecodeCanonicalSourceConfig accepted %s", name)
			}
		})
	}
}

func TestDecodeCanonicalSourceConfigValidatesRuntimeProjectionBindings(t *testing.T) {
	base := "apiVersion: markitect.canonical/v1alpha1\nkind: Source\ndefinitions: []\nmodules: []\n"
	valid := base + `projectionBindings:
  - projection:
      apiVersion: markitect.foundation/v1
      kind: Projection
      namespace: commerce
      name: application-dotnet
    module: dotnet-runtime
`
	if config, err := DecodeCanonicalSourceConfig([]byte(valid)); err != nil || len(config.ProjectionBindings) != 1 {
		t.Fatalf("valid runtime binding: config=%#v err=%v", config, err)
	}
	cases := map[string]string{
		"duplicate projection": base + `projectionBindings:
  - projection: {apiVersion: markitect.foundation/v1, kind: Projection, namespace: commerce, name: api}
    module: dotnet-runtime
  - projection: {apiVersion: markitect.foundation/v1, kind: Projection, namespace: commerce, name: api}
    module: markdown-runtime
`,
		"missing module": base + `projectionBindings:
  - projection: {apiVersion: markitect.foundation/v1, kind: Projection, namespace: commerce, name: api}
`,
		"wrong identity": base + `projectionBindings:
  - projection: {apiVersion: example/v1, kind: UseCase, namespace: commerce, name: api}
    module: dotnet-runtime
`,
	}
	for name, input := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := DecodeCanonicalSourceConfig([]byte(input)); err == nil {
				t.Fatalf("DecodeCanonicalSourceConfig accepted malformed binding: %s", name)
			}
		})
	}
}
