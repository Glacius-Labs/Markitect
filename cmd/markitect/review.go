package main

import (
	"fmt"
	"io"
	"os"
	"path"
	"strings"
	"unicode/utf8"

	"github.com/Glacius-Labs/Markitect/internal/app"
)

func runReview(root string, p *app.Project, packageName, namespace, kind, name, configPath, reportPath, evidencePath, toolDigest string, emit func(any) int, fail func(error) int) int {
	if p.Snapshot.Provisional || namespace == "" || kind == "" || name == "" || configPath == "" {
		return fail(fmt.Errorf("review requires --revision, --namespace, --kind, --name and --config"))
	}
	if (reportPath == "") == (evidencePath == "") {
		return fail(fmt.Errorf("review requires exactly one of --report or --evidence"))
	}
	if path.IsAbs(configPath) || path.Clean(configPath) != configPath || strings.ContainsAny(configPath, "\\:") || configPath == ".." || strings.HasPrefix(configPath, "../") {
		return fail(fmt.Errorf("review configuration must be a normalized repository-relative path"))
	}
	data, ok := p.Snapshot.Files[configPath]
	if !ok {
		return fail(fmt.Errorf("review configuration %s is absent from the fixed snapshot", configPath))
	}
	config, err := app.DecodeReviewConfig(data)
	if err != nil {
		return fail(err)
	}
	key := namespace + "/" + kind + "/" + name
	if packageName != "" {
		key = packageName + "::" + key
	}
	if reportPath != "" {
		report, err := readReviewFile(reportPath)
		if err != nil {
			return fail(err)
		}
		record, err := app.RecordReview(p, key, version, toolDigest, config, string(report))
		if err != nil {
			return fail(err)
		}
		return emit(record)
	}
	data, err = readReviewFile(evidencePath)
	if err != nil {
		return fail(err)
	}
	record, err := app.DecodeReviewRecord(data)
	if err != nil {
		return fail(err)
	}
	if record.Entry != key {
		return fail(fmt.Errorf("review evidence entry %s does not match requested entry %s", record.Entry, key))
	}
	before, err := app.Load(root, record.Revision)
	if err != nil {
		return fail(fmt.Errorf("cannot load original review snapshot: %w", err))
	}
	result, err := app.ReuseReview(before, p, record, version, toolDigest, config)
	if err != nil {
		return fail(err)
	}
	if code := emit(result); code != 0 {
		return code
	}
	if result.Status != "reusable" {
		return 1
	}
	return 0
}

func readReviewFile(name string) ([]byte, error) {
	f, err := os.Open(name)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil || !info.Mode().IsRegular() {
		return nil, fmt.Errorf("review input must be a regular file: %s", name)
	}
	data, err := io.ReadAll(io.LimitReader(f, (2<<20)+1))
	if err != nil {
		return nil, err
	}
	if len(data) > 2<<20 || !utf8.Valid(data) || strings.ContainsRune(string(data), 0) {
		return nil, fmt.Errorf("review input must be UTF-8 text without NUL and at most 2 MiB")
	}
	return data, nil
}
