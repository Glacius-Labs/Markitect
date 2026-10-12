package cli

import (
	"fmt"
	"io"
)

func printUsage(out io.Writer) {
	fmt.Fprintln(out, "Markitect: desired engineering state and bounded reconciliation")
	fmt.Fprintln(out, "usage: markitect COMMAND [options]")
	fmt.Fprintln(out, "This is the legacy Project/Domain command tree. Model-first projects use the")
	fmt.Fprintln(out, "product binary's verbs instead: 'markitect init', 'markitect schema', 'markitect help'.")
	fmt.Fprintln(out, "  check, verify, inventory    Validate a project and its repository gates")
	fmt.Fprintln(out, "  model, context, impact, find, explain, review    Inspect fixed semantic inputs")
	fmt.Fprintln(out, "  authoring, render, format, reconcile, projection    Author and reconcile projections")
	fmt.Fprintln(out, "  prepare, copy-me             Capture selected evidence and validate proposals (source)")
	fmt.Fprintln(out, "  pack                         Build an offline content package")
	fmt.Fprintln(out, "  schema")
	fmt.Fprintln(out, "Use 'markitect help COMMAND' for applicable options.")
}
