package release

import (
	"archive/zip"
	"bytes"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/core/snapshot"
)

func bundleSnapshot() *snapshot.Snapshot {
	files := map[string][]byte{
		"go.mod":                       []byte("module github.com/Glacius-Labs/Markitect\n\ngo 1.27.1\n"),
		"go.sum":                       []byte("go.yaml.in/yaml/v3 v3.0.5 h1:fixture\n"),
		"README.md":                    []byte("Markitect\r\n"),
		"LICENSE":                      []byte("Apache License\r\nVersion 2.0\r\n"),
		"cmd/markitect/main.go":        []byte("package main\n"),
		"internal/host/cli/version.go": []byte("package cli\nvar version = \"1.2.3\"\n"),
		"internal/core/model.go":       []byte("package core\n"),
		"internal/host/embedded/resources/skill.yaml": []byte("kind: Skill\n"),
		"integration/run-markitect.go":                []byte("package main\r\n"),
		"integration/run-markitect_test.go":           []byte("package main\r\n"),
		"schema/manifest.yaml":                        []byte("schemaVersion: 1\r\n"),
	}
	modes := make(map[string]string, len(files))
	for name := range files {
		modes[name] = "100644"
	}
	return &snapshot.Snapshot{ID: strings.Repeat("a", 40), Files: files, Modes: modes}
}

type bundleTestEntry struct {
	name string
	data []byte
	mode os.FileMode
}

func rewriteBundle(t *testing.T, data []byte, change func(map[string][]byte)) []byte {
	t.Helper()
	files := archiveContents(t, data)
	change(files)
	entries := make([]bundleTestEntry, 0, len(files))
	for name, content := range files {
		entries = append(entries, bundleTestEntry{name: name, data: content, mode: 0644})
	}
	return writeBundleTestZip(t, entries)
}

func rewriteBundleEntries(t *testing.T, data []byte, change func(*[]bundleTestEntry)) []byte {
	t.Helper()
	entries := bundleTestEntries(t, data)
	change(&entries)
	return writeBundleTestZip(t, entries)
}

func rewriteBundleEntriesWithMode(t *testing.T, data []byte, change func(*[]bundleTestEntry)) []byte {
	t.Helper()
	entries := bundleTestEntries(t, data)
	change(&entries)
	return writeBundleTestZip(t, entries)
}

func bundleTestEntries(t *testing.T, data []byte) []bundleTestEntry {
	t.Helper()
	files := archiveContents(t, data)
	entries := make([]bundleTestEntry, 0, len(files))
	for name, content := range files {
		entries = append(entries, bundleTestEntry{name: name, data: content, mode: 0644})
	}
	return entries
}

func writeBundleTestZip(t *testing.T, entries []bundleTestEntry) []byte {
	t.Helper()
	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)
	for _, item := range entries {
		header := &zip.FileHeader{Name: item.name, Method: zip.Store}
		header.SetModTime(time.Date(1980, 1, 1, 0, 0, 0, 0, time.UTC))
		header.SetMode(item.mode)
		entry, err := writer.CreateHeader(header)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := entry.Write(item.data); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return buffer.Bytes()
}

func assertBundleRejected(t *testing.T, data []byte) {
	t.Helper()
	if _, err := ParseBundle(data, digestBytes(data)); err == nil {
		t.Fatal("invalid release bundle was accepted")
	}
}

func cloneBundleFiles(files map[string][]byte) map[string][]byte {
	clone := make(map[string][]byte, len(files))
	for name, data := range files {
		clone[name] = append([]byte(nil), data...)
	}
	return clone
}
