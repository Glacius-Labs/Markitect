package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"runtime"

	"github.com/Glacius-Labs/Markitect/internal/buildinfo"
	"go.yaml.in/yaml/v3"
)

// Local preparation never calls GitHub or publishes assets.
func runLocal(command string, args []string, out, errout io.Writer) error {
	flags := flag.NewFlagSet("markitect-release "+command, flag.ContinueOnError)
	flags.SetOutput(errout)
	var root, revision, output, bundle, digest string
	var write bool
	switch command {
	case "package", "bundle", "install":
		flags.StringVar(&root, "repo", ".", "repository root")
	}
	switch command {
	case "package", "bundle":
		flags.StringVar(&output, "output", "", "absent package directory or bundle ZIP file")
	}
	if command == "bundle" {
		flags.StringVar(&revision, "revision", "", "fixed Git commit to bundle")
	}
	if command == "install" {
		flags.StringVar(&bundle, "bundle", "", "local bundle ZIP")
		flags.StringVar(&digest, "sha256", "", "expected bundle SHA-256")
		flags.BoolVar(&write, "write", false, "apply the verified installation plan")
	}
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if flags.NArg() != 0 {
		return errors.New("unexpected positional arguments")
	}
	emit := func(value any) error {
		data, err := yaml.Marshal(value)
		if err != nil {
			return err
		}
		n, err := out.Write(data)
		if err == nil && n != len(data) {
			return io.ErrShortWrite
		}
		return err
	}
	switch command {
	case "version":
		_, err := fmt.Fprintf(out, "Markitect %s (%s/%s)\n", buildinfo.Version(), runtime.GOOS, runtime.GOARCH)
		return err
	case "licenses":
		n, err := io.WriteString(out, buildinfo.Notices)
		if err == nil && n != len(buildinfo.Notices) {
			return io.ErrShortWrite
		}
		return err
	case "package":
		return packageSource(root, output, emit)
	case "bundle":
		return bundleSource(root, revision, output, emit)
	case "install":
		return installBundle(root, bundle, digest, write, emit)
	}
	return fmt.Errorf("unknown local release command %q", command)
}
