package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/Glacius-Labs/Markitect/internal/app"
	"github.com/Glacius-Labs/Markitect/internal/release"
	"github.com/Glacius-Labs/Markitect/internal/source"
)

func printUsage(out io.Writer) {
	fmt.Fprintln(out, "Markitect — typed AI documentation and architecture")
	fmt.Fprintln(out, "usage: markitect COMMAND [options]")
	fmt.Fprintln(out, "  check, verify, inventory    Validate a project and its repository gates")
	fmt.Fprintln(out, "  model, context, impact, find, explain, review    Inspect fixed semantic inputs")
	fmt.Fprintln(out, "  init, authoring, render, format, reconcile    Author and reconcile projections")
	fmt.Fprintln(out, "  pack                         Build an offline content package")
	fmt.Fprintln(out, "  bundle, install, package, schema, version, licenses")
	fmt.Fprintln(out, "Use 'markitect help COMMAND' for applicable options.")
}

func commandFlags(command string) (map[string]bool, bool) {
	flags := map[string]bool{}
	var names []string
	switch command {
	case "check", "verify", "inventory", "model":
		names = []string{"repo", "revision"}
	case "context", "explain":
		names = []string{"repo", "revision", "api-version", "kind", "name", "namespace", "package"}
		if command == "context" {
			names = append(names, "run")
		}
	case "review":
		names = []string{"repo", "revision", "api-version", "kind", "name", "namespace", "package", "config", "report", "evidence"}
	case "find":
		names = []string{"repo", "revision", "api-version", "query", "kind", "namespace", "package"}
	case "impact":
		names = []string{"repo", "revision", "base"}
	case "render", "format":
		names = []string{"repo", "revision", "write", "check"}
	case "reconcile":
		names = []string{"repo", "revision", "action", "adapter", "plan", "write"}
	case "schema":
		names = []string{"repo", "write", "check"}
	case "package":
		names = []string{"repo", "output"}
	case "bundle", "pack":
		names = []string{"repo", "revision", "output"}
	case "install":
		names = []string{"repo", "bundle", "sha256", "write"}
	case "init":
		names = []string{"repo", "name", "namespace", "path", "write"}
	case "authoring", "version", "licenses":
	default:
		return nil, false
	}
	for _, name := range names {
		flags[name] = true
	}
	return flags, true
}

func runBundle(root, revision, output string, emit func(any) int, fail func(error) int) int {
	if revision == "" || output == "" {
		return fail(fmt.Errorf("bundle requires --revision and --output pointing to an absent ZIP file"))
	}
	snapshot, err := source.Load(root, revision)
	if err != nil {
		return fail(err)
	}
	data, err := release.BuildBundle(snapshot, version)
	if err != nil {
		return fail(err)
	}
	file, err := os.OpenFile(output, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if err != nil {
		return fail(err)
	}
	_, writeErr := file.Write(data)
	if writeErr == nil {
		writeErr = file.Sync()
	}
	closeErr := file.Close()
	if writeErr != nil || closeErr != nil {
		// This path was exclusively created by this invocation. Incomplete bundles
		// must never be mistaken for a successful release.
		_ = os.Remove(output)
		if writeErr != nil {
			return fail(writeErr)
		}
		return fail(closeErr)
	}
	digest := sha256.Sum256(data)
	return emit(map[string]any{"status": "bundled", "version": version, "sourceCommit": snapshot.ID, "sha256": hex.EncodeToString(digest[:]), "output": output})
}

func runInstall(root, path, expectedSHA string, write bool, emit func(any) int, fail func(error) int) int {
	if path == "" || expectedSHA == "" {
		return fail(fmt.Errorf("install requires --bundle and --sha256; use --write to apply the verified plan"))
	}
	info, err := os.Lstat(path)
	if err != nil {
		return fail(err)
	}
	const limit = 64 << 20
	if !info.Mode().IsRegular() || info.Size() > limit {
		return fail(fmt.Errorf("release bundle must be a regular file no larger than 64 MiB"))
	}
	file, err := os.Open(path)
	if err != nil {
		return fail(err)
	}
	data, err := io.ReadAll(io.LimitReader(file, limit+1))
	closeErr := file.Close()
	if err != nil {
		return fail(err)
	}
	if closeErr != nil {
		return fail(closeErr)
	}
	if len(data) > limit {
		return fail(fmt.Errorf("release bundle grew beyond 64 MiB while reading"))
	}
	bundle, err := release.ParseBundle(data, expectedSHA)
	if err != nil {
		return fail(err)
	}
	plan, err := app.Install(root, bundle, write)
	if err != nil {
		if plan != nil {
			if code := emit(map[string]any{"status": "failed", "plan": plan}); code != 0 {
				return code
			}
		}
		return fail(err)
	}
	status := "planned"
	if write {
		status = "installed"
	}
	if plan.Kind == "noop" {
		status = "unchanged"
	}
	return emit(map[string]any{"status": status, "plan": plan})
}

func runPackage(o commandOptions, emit func(any) int, fail func(error) int) int {
	if o.output == "" {
		return fail(fmt.Errorf("package requires --output pointing to an absent directory"))
	}
	if _, err := os.Lstat(o.output); !os.IsNotExist(err) {
		return fail(fmt.Errorf("release output must not already exist"))
	}
	archive, lock, err := release.Package(o.root, version)
	if err != nil {
		return fail(err)
	}
	if err = os.MkdirAll(filepath.Join(o.output, ".markitect", "tool"), 0755); err != nil {
		return fail(err)
	}
	if err = os.WriteFile(filepath.Join(o.output, ".markitect", "tool", "source.zip"), archive, 0644); err != nil {
		return fail(err)
	}
	if err = os.WriteFile(filepath.Join(o.output, ".markitect", "tool", "lock.yaml"), lock, 0644); err != nil {
		return fail(err)
	}
	return emit(map[string]any{"status": "packaged", "version": version, "output": o.output})
}
