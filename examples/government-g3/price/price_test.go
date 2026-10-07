package price

import (
	"os"
	"strings"
	"testing"
)

func TestLocalPriceIsValid(t *testing.T) {
	data, err := os.ReadFile("price.txt")
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(string(data)) != "unit=4" {
		t.Fatalf("price local contract = %q, want unit=4", strings.TrimSpace(string(data)))
	}
}
