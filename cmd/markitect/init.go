package main

import (
	"fmt"

	"github.com/Glacius-Labs/Markitect/internal/host"
)

func runInit(root, name, namespace, path string, write bool, emit func(any) int, fail func(error) int) int {
	if name == "" || namespace == "" {
		return fail(fmt.Errorf("init requires --name and --namespace; --path is optional (defaults to .markitect/areas/<namespace>); inspect the preview before applying with --write"))
	}
	plan, err := host.Init(root, host.InitOptions{Name: name, Namespace: namespace, Path: path}, write)
	if err != nil {
		if plan != nil {
			if code := emit(map[string]any{"status": "failed", "version": version, "plan": plan}); code != 0 {
				return code
			}
		}
		return fail(err)
	}
	status := "planned"
	if plan.Applied {
		status = "initialized"
	}
	return emit(map[string]any{"status": status, "version": version, "plan": plan})
}
