package main

import (
	"fmt"
	"os"
	"strings"
)

func main() {
	handler, err := os.ReadFile("src/Commerce/CreateOrderHandler.cs")
	if err != nil {
		fail("CreateOrderHandler.cs: %v", err)
	}
	if !strings.Contains(string(handler), "class CreateOrderHandler") {
		fail("CreateOrderHandler.cs is missing the fixture type marker")
	}

	effectAxis, err := os.ReadFile("src/Commerce/EffectAxis.cs")
	if err != nil {
		fail("EffectAxis.cs: %v", err)
	}
	effectText := string(effectAxis)
	requiredBoundary := "public string Boundary { get; init; } = \"application\";"
	if !strings.Contains(effectText, "record EffectAxis") || !strings.Contains(effectText, requiredBoundary) {
		fail("EffectAxis.cs must retain the explicit application boundary property")
	}
	if strings.Contains(effectText, "Inventory") || strings.Contains(effectText, "Payment") {
		fail("EffectAxis.cs contains unmodeled enum categories")
	}

	project, err := os.ReadFile("src/Commerce/Commerce.csproj")
	if err != nil {
		fail("Commerce.csproj: %v", err)
	}
	if !strings.Contains(string(project), "<TargetFramework>net8.0</TargetFramework>") {
		fail("Commerce.csproj is missing the fixture target framework")
	}
	fmt.Println("projection fixture contains the explicit boundary and declared file markers")
}

func fail(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
