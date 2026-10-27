package main

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"path/filepath"

	"github.com/Glacius-Labs/Markitect/internal/buildinfo"
	"github.com/Glacius-Labs/Markitect/internal/tooling/release"
)

func packageSource(root, output string, emit func(any) error) error {
	if output == "" {
		return errors.New("package requires --output pointing to an absent directory")
	}
	if _, err := os.Lstat(output); !os.IsNotExist(err) {
		return errors.New("release output must not already exist")
	}
	archive, lock, err := release.Package(root, buildinfo.Version())
	if err != nil {
		return err
	}
	target := filepath.Join(output, ".markitect", "tool")
	if err := os.MkdirAll(target, 0755); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(target, "source.zip"), archive, 0644); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(target, "lock.yaml"), lock, 0644); err != nil {
		return err
	}
	return emit(map[string]any{"status": "packaged", "version": buildinfo.Version(), "output": output})
}

func bundleSource(root, revision, output string, emit func(any) error) error {
	if revision == "" || output == "" {
		return errors.New("bundle requires --revision and --output pointing to an absent ZIP file")
	}
	fixed, err := release.LoadSource(root, revision)
	if err != nil {
		return err
	}
	data, err := release.BuildBundle(fixed, buildinfo.Version())
	if err != nil {
		return err
	}
	file, err := os.OpenFile(output, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if err != nil {
		return err
	}
	_, writeErr := file.Write(data)
	if writeErr == nil {
		writeErr = file.Sync()
	}
	closeErr := file.Close()
	if writeErr != nil || closeErr != nil {
		// This path was created exclusively by this invocation; remove an
		// incomplete bundle so it cannot be mistaken for a successful artifact.
		_ = os.Remove(output)
		if writeErr != nil {
			return writeErr
		}
		return closeErr
	}
	digest := sha256.Sum256(data)
	return emit(map[string]any{"status": "bundled", "version": buildinfo.Version(), "sourceCommit": fixed.ID, "sha256": hex.EncodeToString(digest[:]), "output": output})
}

func installBundle(root, path, expectedSHA string, write bool, emit func(any) error) error {
	if path == "" || expectedSHA == "" {
		return errors.New("install requires --bundle and --sha256; use --write to apply the verified plan")
	}
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	const limit = 64 << 20
	if !info.Mode().IsRegular() || info.Size() > limit {
		return errors.New("release bundle must be a regular file no larger than 64 MiB")
	}
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	data, readErr := io.ReadAll(io.LimitReader(file, limit+1))
	closeErr := file.Close()
	if readErr != nil {
		return readErr
	}
	if closeErr != nil {
		return closeErr
	}
	if len(data) > limit {
		return errors.New("release bundle grew beyond 64 MiB while reading")
	}
	bundle, err := release.ParseBundle(data, expectedSHA)
	if err != nil {
		return err
	}
	plan, err := release.Install(root, bundle, write)
	if err != nil {
		if plan != nil {
			if emitErr := emit(map[string]any{"status": "failed", "plan": plan}); emitErr != nil {
				return emitErr
			}
		}
		return err
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
