package main

import (
	"fmt"
	"os"
	"strings"
)

type assertion struct {
	path      string
	fragments []string
}

func main() {
	assertions := []assertion{
		{path: "examples/canonical-workflow/definitions/workflow.rule.yaml", fragments: []string{"canonical workflow definitions and their schema as the authoritative meaning", "reviewable representation"}},
		{path: "examples/canonical-workflow/definitions/workflow.process.yaml", fragments: []string{"record the proposed workflow change in its canonical definition", "review the source and its references"}},
		{path: "examples/canonical-workflow/definitions/workflow.responsibility.yaml", fragments: []string{"project-designated workflow owner", "accountability"}},
		{path: "examples/canonical-workflow/definitions/workflow.gate.yaml", fragments: []string{"workflow owner has reviewed the canonical definition"}},
	}
	for _, item := range assertions {
		data, err := os.ReadFile(item.path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s: %v\n", item.path, err)
			os.Exit(1)
		}
		content := strings.ToLower(strings.Join(strings.Fields(string(data)), " "))
		for _, fragment := range item.fragments {
			if !strings.Contains(content, fragment) {
				fmt.Fprintf(os.Stderr, "%s is missing canonical workflow assertion %q\n", item.path, fragment)
				os.Exit(1)
			}
		}
	}
	fmt.Println("canonical workflow assertions passed")
}
