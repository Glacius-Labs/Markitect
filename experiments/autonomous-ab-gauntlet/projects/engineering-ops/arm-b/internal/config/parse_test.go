package config

import "testing"

func TestParseConfig(t *testing.T) {
	document, err := Parse([]byte(`{"name":"relay","labels":{"region":"eu"}}`))
	if err != nil {
		t.Fatal(err)
	}
	if document.Name != "relay" || document.Labels["region"] != "eu" {
		t.Fatalf("unexpected document: %#v", document)
	}
}
