package main

import (
	"github.com/Glacius-Labs/Markitect/internal/app"
)

func runFormat(o commandOptions, p *app.Project, emit func(any) int, fail func(error) int) int {
	files, err := app.Format(o.root, p, o.write)
	if err != nil {
		if len(files) > 0 {
			if code := emit(map[string]any{"status": "failed", "written": files, "recovery": "Inspect the listed paths and Git diff before retrying; the complete operation is not a filesystem transaction."}); code != 0 {
				return code
			}
		}
		return fail(err)
	}
	code := emit(map[string]any{"changed": files, "written": o.write})
	if code != 0 {
		return code
	}
	if !o.write && len(files) > 0 {
		return 1
	}
	return 0
}
