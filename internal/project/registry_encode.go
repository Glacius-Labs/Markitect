package project

import (
	"bytes"
	"fmt"
	"slices"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/organization"
	"github.com/Glacius-Labs/Markitect/internal/realization"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
	"go.yaml.in/yaml/v3"
)

const registryAPIVersion = "markitect.registry/v1"

// registryShardPath is the shard that holds one owner's records.
func registryShardPath(owner realization.Owner) (snapshot.Path, error) {
	if owner.IsRoot() {
		return snapshot.NewPath(registryDirectory + "registry.yaml")
	}
	name, ok := owner.Package()
	if !ok {
		return snapshot.Path{}, fmt.Errorf("registry owner has no Package")
	}
	return snapshot.NewPath(registryDirectory + name + "/registry.yaml")
}

func ownerKey(owner realization.Owner) string {
	if owner.IsRoot() {
		return organization.Root
	}
	name, _ := owner.Package()
	return name
}

// publicationTree is the tree a verified aggregate publishes: its files, with
// every registry shard whose records changed regenerated from the aggregate's
// registry. A regenerated shard keeps the captured order of the records that
// remain and lists new records after them in path order.
func publicationTree(captured snapshot.Tree, capturedRegistry realization.Registry, aggregate snapshot.Tree, registry realization.Registry) (snapshot.Tree, error) {
	before, after := map[string][]realization.Registration{}, map[string][]realization.Registration{}
	owners := map[string]realization.Owner{}
	for _, r := range capturedRegistry.Entries() {
		before[ownerKey(r.Owner())] = append(before[ownerKey(r.Owner())], r)
		owners[ownerKey(r.Owner())] = r.Owner()
	}
	for _, r := range registry.Entries() {
		after[ownerKey(r.Owner())] = append(after[ownerKey(r.Owner())], r)
		owners[ownerKey(r.Owner())] = r.Owner()
	}
	files := map[snapshot.Path]snapshot.Entry{}
	for _, e := range aggregate.Entries() {
		files[e.Path()] = e
	}
	keys := make([]string, 0, len(owners))
	for key := range owners {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	for _, key := range keys {
		if sameRecords(before[key], after[key]) {
			continue
		}
		path, err := registryShardPath(owners[key])
		if err != nil {
			return snapshot.Tree{}, err
		}
		var order []string
		if shard, ok := captured.Lookup(path); ok {
			order = shardOrder(shard.Bytes())
		}
		data, err := encodeShard(after[key], order)
		if err != nil {
			return snapshot.Tree{}, err
		}
		mode := snapshot.Regular
		if existing, ok := files[path]; ok {
			mode = existing.Mode()
		}
		entry, err := snapshot.NewEntry(path, mode, data)
		if err != nil {
			return snapshot.Tree{}, err
		}
		files[path] = entry
	}
	entries := make([]snapshot.Entry, 0, len(files))
	for _, e := range files {
		entries = append(entries, e)
	}
	return snapshot.NewTree(entries)
}

func sameRecords(a, b []realization.Registration) bool {
	key := func(r realization.Registration) string {
		art := r.Artifact()
		var basis []string
		for _, id := range art.Basis() {
			basis = append(basis, id.Package+"/"+id.ID)
		}
		return art.Path().String() + "\x00" + string(art.Form()) + "\x00" + strings.Join(basis, ",") + "\x00" + art.Why()
	}
	if len(a) != len(b) {
		return false
	}
	seen := map[string]bool{}
	for _, r := range a {
		seen[key(r)] = true
	}
	for _, r := range b {
		if !seen[key(r)] {
			return false
		}
	}
	return true
}

// shardOrder returns the record paths of a captured shard in file order.
func shardOrder(data []byte) []string {
	var doc struct {
		Artifacts []struct {
			Path string `yaml:"path"`
		} `yaml:"artifacts"`
	}
	if yaml.Unmarshal(data, &doc) != nil {
		return nil
	}
	var order []string
	for _, a := range doc.Artifacts {
		order = append(order, a.Path)
	}
	return order
}

// encodeShard writes one owner's records in the registry shard format.
func encodeShard(records []realization.Registration, order []string) ([]byte, error) {
	rank := map[string]int{}
	for i, p := range order {
		rank[p] = i
	}
	records = slices.Clone(records)
	slices.SortFunc(records, func(a, b realization.Registration) int {
		pa, pb := a.Artifact().Path().String(), b.Artifact().Path().String()
		ra, oka := rank[pa]
		rb, okb := rank[pb]
		switch {
		case oka && okb:
			return ra - rb
		case oka:
			return -1
		case okb:
			return 1
		}
		return strings.Compare(pa, pb)
	})
	str := func(value string, style yaml.Style) *yaml.Node {
		return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: value, Style: style}
	}
	artifacts := &yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq"}
	for _, r := range records {
		art := r.Artifact()
		realizes := &yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq", Style: yaml.FlowStyle}
		for _, id := range art.Basis() {
			realizes.Content = append(realizes.Content, str(id.Package+"/"+id.ID, 0))
		}
		artifacts.Content = append(artifacts.Content, &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map", Content: []*yaml.Node{
			str("path", 0), str(art.Path().String(), yaml.DoubleQuotedStyle),
			str("form", 0), str(string(art.Form()), 0),
			str("realizes", 0), realizes,
			str("why", 0), str(art.Why(), 0),
		}})
	}
	if len(records) == 0 {
		artifacts.Style = yaml.FlowStyle
	}
	doc := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map", Content: []*yaml.Node{str("apiVersion", 0), str(registryAPIVersion, 0), str("artifacts", 0), artifacts}}
	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(doc); err != nil {
		return nil, err
	}
	if err := enc.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
