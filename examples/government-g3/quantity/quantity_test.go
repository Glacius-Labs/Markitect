package quantity

import (
	"os"
	"strings"
	"testing"
)

func TestLocalQuantityIsValid(t *testing.T) {
	data, err := os.ReadFile("quantity.txt")
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(string(data)) != "units=3" {
		t.Fatalf("quantity local contract = %q, want units=3", strings.TrimSpace(string(data)))
	}
}
