package publish

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"go.yaml.in/yaml/v3"
)

type namedBytes map[string][]byte

func assetNames(tag string) []string {
	return []string{"markitect-" + tag + "-bundle.zip", "markitect-" + tag + "-linux-amd64", "markitect-" + tag + "-windows-amd64.exe", "markitect-" + tag + "-provenance.yaml"}
}

func readAssets(dir, tag string) (namedBytes, error) {
	dirInfo, err := os.Lstat(dir)
	if err != nil {
		return nil, fmt.Errorf("inspect asset directory: %w", err)
	}
	if !dirInfo.IsDir() || dirInfo.Mode()&os.ModeSymlink != 0 {
		return nil, errors.New("asset directory must be a real directory, not a symlink")
	}
	names := assetNames(tag)
	expected := make(map[string]bool, len(names))
	for _, n := range names {
		expected[n] = true
	}
	dirEntries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	entries := make([]string, 0, len(dirEntries))
	for _, entry := range dirEntries {
		entries = append(entries, filepath.Join(dir, entry.Name()))
	}
	if len(entries) != len(names) {
		return nil, fmt.Errorf("asset directory must contain exactly four regular files named %s", strings.Join(names, ", "))
	}
	out := make(namedBytes, len(names))
	for _, path := range entries {
		fileInfo, err := os.Lstat(path)
		if err != nil {
			return nil, err
		}
		if !fileInfo.Mode().IsRegular() {
			return nil, fmt.Errorf("asset %s is not a regular file", filepath.Base(path))
		}
		name := filepath.Base(path)
		if !expected[name] {
			return nil, fmt.Errorf("unexpected release asset %q", name)
		}
		limit := int64(256 << 20)
		if name == names[0] {
			limit = 64 << 20
		} else if name == names[3] {
			limit = 64 << 10
		}
		data, err := readLimited(path, limit)
		if err != nil {
			return nil, err
		}
		out[name] = data
	}
	for _, name := range names {
		if _, ok := out[name]; !ok {
			return nil, fmt.Errorf("release asset %q is missing", name)
		}
	}
	prov, err := parseProvenance(out[names[3]])
	if err != nil {
		return nil, err
	}
	if prov.Tag != tag || prov.SchemaVersion != "1" || strings.TrimSpace(prov.Toolchain) == "" {
		return nil, errors.New("provenance schema, tag, or toolchain is invalid")
	}
	if len(prov.Assets) != 3 {
		return nil, errors.New("provenance must bind exactly the three payload assets")
	}
	for i, name := range names[:3] {
		if prov.Assets[i].Name != name || !digestPattern.MatchString(prov.Assets[i].SHA256) || prov.Assets[i].SHA256 != digest(out[name]) {
			return nil, fmt.Errorf("provenance digest for %s is invalid", name)
		}
	}
	return out, nil
}

// Parsing checks duplicate YAML keys as well as unknown fields. The provenance
// is a claim until its selected run artifact has been independently fetched.
func parseProvenance(data []byte) (Provenance, error) {
	var node yaml.Node
	dec := yaml.NewDecoder(bytes.NewReader(data))
	if err := dec.Decode(&node); err != nil {
		return Provenance{}, fmt.Errorf("decode provenance: %w", err)
	}
	if err := rejectDuplicateYAML(&node); err != nil {
		return Provenance{}, err
	}
	var extra yaml.Node
	if err := dec.Decode(&extra); err != io.EOF {
		return Provenance{}, errors.New("provenance must contain one YAML document")
	}
	dec = yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	var p Provenance
	if err := dec.Decode(&p); err != nil {
		return p, fmt.Errorf("decode provenance fields: %w", err)
	}
	if p.SchemaVersion != "1" || p.Tag == "" || !commitPattern.MatchString(p.SourceCommit) || p.WorkflowRunID == "" || p.WorkflowRunAttempt == "" || p.WorkflowRun == "" || p.Toolchain == "" {
		return p, errors.New("provenance is missing required schema or source/run fields")
	}
	return p, nil
}

func rejectDuplicateYAML(n *yaml.Node) error {
	if n.Anchor != "" || n.Kind == yaml.AliasNode {
		return errors.New("provenance YAML anchors and aliases are not allowed")
	}
	if n.Tag != "" && n.Tag != "!!map" && n.Tag != "!!seq" && n.Tag != "!!str" {
		return fmt.Errorf("provenance contains unsupported YAML tag %s", n.Tag)
	}
	if n.Kind == yaml.MappingNode {
		seen := map[string]bool{}
		for i := 0; i < len(n.Content); i += 2 {
			k := n.Content[i]
			if k.Kind != yaml.ScalarNode || k.Tag != "!!str" {
				return errors.New("provenance mapping keys must be strings")
			}
			if seen[k.Value] {
				return fmt.Errorf("provenance contains duplicate field %q", k.Value)
			}
			seen[k.Value] = true
		}
	}
	for _, child := range n.Content {
		if err := rejectDuplicateYAML(child); err != nil {
			return err
		}
	}
	return nil
}

func downloadAndValidateArtifact(ctx context.Context, r Runner, runID, name, tag string, run runInfo) (namedBytes, error) {
	dir, err := os.MkdirTemp("", "markitect-release-artifact-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(dir)
	if err := call(r, ctx, "", "run", "download", runID, "--repo", "github.com/"+repository, "--name", name, "--dir", dir); err != nil {
		return nil, fmt.Errorf("download selected run artifact: %w", err)
	}
	assets, err := readAssets(dir, tag)
	if err != nil {
		return nil, fmt.Errorf("downloaded artifact is invalid: %w", err)
	}
	prov, err := parseProvenance(assets[assetNames(tag)[3]])
	if err != nil {
		return nil, err
	}
	wantURL := fmt.Sprintf("https://github.com/%s/actions/runs/%d/attempts/%d", repository, run.ID, run.RunAttempt)
	if prov.SourceCommit != run.HeadSHA || prov.WorkflowRunID != strconv.FormatInt(run.ID, 10) || prov.WorkflowRunAttempt != strconv.Itoa(run.RunAttempt) || prov.WorkflowRun != wantURL {
		return nil, errors.New("artifact provenance does not match selected workflow run, attempt, source commit, and repository URL")
	}
	return assets, nil
}

func digest(data []byte) string { sum := sha256.Sum256(data); return hex.EncodeToString(sum[:]) }
func assetRows(files namedBytes) []Asset {
	rows := make([]Asset, 0, len(files))
	for name, data := range files {
		rows = append(rows, Asset{Name: name, SHA256: digest(data)})
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].Name < rows[j].Name })
	return rows
}
func sameAssetBytes(a, b namedBytes) bool {
	if len(a) != len(b) {
		return false
	}
	for n, d := range a {
		if !bytes.Equal(d, b[n]) {
			return false
		}
	}
	return true
}

func stageVerifiedAssets(files namedBytes, tag string) (string, error) {
	dir, err := os.MkdirTemp("", "markitect-release-upload-")
	if err != nil {
		return "", err
	}
	for _, name := range assetNames(tag) {
		data, ok := files[name]
		if !ok {
			os.RemoveAll(dir)
			return "", fmt.Errorf("verified payload %s is missing", name)
		}
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, data, 0600); err != nil {
			os.RemoveAll(dir)
			return "", err
		}
		readback, err := readLimited(path, 256<<20)
		if err != nil || !bytes.Equal(data, readback) {
			os.RemoveAll(dir)
			return "", fmt.Errorf("staged payload %s failed readback verification", name)
		}
	}
	return dir, nil
}

func readLimited(name string, limit int64) ([]byte, error) {
	f, err := os.Open(name)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > limit {
		return nil, fmt.Errorf("%s exceeds size limit", name)
	}
	return data, nil
}
