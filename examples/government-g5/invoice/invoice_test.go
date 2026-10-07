package invoice

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
)

func TestProposedDeliveryRuleAndRealizationAgree(t *testing.T) {
	modelBytes, err := os.ReadFile("../government.yaml")
	if err != nil {
		t.Fatal(err)
	}
	var source struct {
		Definitions []struct {
			Kind     string `json:"kind"`
			Metadata struct {
				Name string `json:"name"`
			} `json:"metadata"`
			Spec struct {
				Statement string `json:"statement"`
			} `json:"spec"`
		} `json:"definitions"`
	}
	if err := json.Unmarshal(modelBytes, &source); err != nil {
		t.Fatalf("decode candidate GovernmentSource: %v", err)
	}
	statement := ""
	for _, definition := range source.Definitions {
		if definition.Kind == "Requirement" && definition.Metadata.Name == "delivery-rule" {
			statement = definition.Spec.Statement
		}
	}
	if strings.Contains(statement, "plus the published delivery charge") {
		rateBytes, err := os.ReadFile("../delivery/rate.txt")
		if err != nil {
			t.Fatal(err)
		}
		var charge int
		if _, err := fmt.Sscanf(strings.TrimSpace(string(rateBytes)), "charge=%d", &charge); err != nil {
			t.Fatal(err)
		}
		if got, want := Total(3, 4, charge), 14; got != want {
			t.Fatalf("candidate realization total = %d, want %d", got, want)
		}
		return
	}
	rateBytes, err := os.ReadFile("../delivery/rate.txt")
	if err != nil {
		t.Fatal(err)
	}
	var charge int
	if _, err := fmt.Sscanf(strings.TrimSpace(string(rateBytes)), "charge=%d", &charge); err != nil {
		t.Fatal(err)
	}
	if charge != 2 {
		t.Fatalf("published delivery charge = %d, want 2", charge)
	}
	if got, want := Total(3, 4, charge), 12; got != want {
		t.Fatalf("prior realization total = %d, want %d", got, want)
	}
}
