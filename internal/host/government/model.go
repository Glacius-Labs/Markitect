package government

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path"
	"sort"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/core"
	"github.com/Glacius-Labs/Markitect/internal/host/government/inventory"
	"go.yaml.in/yaml/v3"
)

type Source struct {
	APIVersion   string                  `json:"apiVersion" yaml:"apiVersion"`
	Kind         string                  `json:"kind" yaml:"kind"`
	Constitution core.DefinitionIdentity `json:"constitution" yaml:"constitution"`
	Schemas      []core.Schema           `json:"schemas" yaml:"schemas"`
	Definitions  []core.Definition       `json:"definitions" yaml:"definitions"`
	Observation  inventory.Options       `json:"observation" yaml:"observation"`
}

type Finding struct {
	Code    string `json:"code" yaml:"code"`
	Subject string `json:"subject,omitempty" yaml:"subject,omitempty"`
	Detail  string `json:"detail" yaml:"detail"`
}

type Model struct {
	Digest       string                    `json:"digest" yaml:"digest"`
	Canonical    core.Model                `json:"canonical" yaml:"canonical"`
	Constitution core.DefinitionIdentity   `json:"constitution" yaml:"constitution"`
	Cabinet      []core.DefinitionIdentity `json:"cabinet" yaml:"cabinet"`
	Root         core.DefinitionIdentity   `json:"root" yaml:"root"`
	Findings     []Finding                 `json:"findings" yaml:"findings"`
	byID         map[string]core.Definition
}

// Decode rejects additional documents, aliases, unknown fields and unbounded
// input. Canonical provenance is assigned by the loader, never accepted as an
// authority assertion from YAML.
func Decode(data []byte, value any) error {
	if len(data) == 0 || len(data) > 8<<20 {
		return errors.New("Government input must contain 1..8388608 bytes")
	}
	var node yaml.Node
	if err := yaml.Unmarshal(data, &node); err != nil {
		return err
	}
	var visit func(*yaml.Node) error
	visit = func(n *yaml.Node) error {
		if n.Kind == yaml.AliasNode || n.Anchor != "" {
			return errors.New("Government input cannot contain YAML aliases or anchors")
		}
		for _, child := range n.Content {
			if err := visit(child); err != nil {
				return err
			}
		}
		return nil
	}
	if err := visit(&node); err != nil {
		return err
	}
	d := yaml.NewDecoder(bytes.NewReader(data))
	d.KnownFields(true)
	if err := d.Decode(value); err != nil {
		return err
	}
	var extra any
	if err := d.Decode(&extra); err != io.EOF {
		return errors.New("Government input must contain exactly one document")
	}
	return nil
}

func Digest(value any) string {
	data, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:])
}
func BytesDigest(data []byte) string {
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:])
}

// Compile interprets only explicit Government contracts, after ordinary Core
// structural compilation. It never analyzes adopting-project source semantics.
func Compile(s Source) Model {
	m := Model{Constitution: s.Constitution, byID: map[string]core.Definition{}}
	add := func(code, subject, detail string) { m.Findings = append(m.Findings, Finding{code, subject, detail}) }
	if s.APIVersion != SourceVersion || s.Kind != "GovernmentSource" {
		add("source.version", "", "expected "+SourceVersion+" GovernmentSource")
		return m
	}
	schemas := append([]core.Schema{Schema()}, s.Schemas...)
	var diagnostics []core.Diagnostic
	m.Canonical, diagnostics = core.Compile(schemas, s.Definitions, "")
	for _, d := range diagnostics {
		add(d.Code, d.Identity, d.Property+": "+d.Message)
	}
	if len(diagnostics) > 0 {
		sortFindings(m.Findings)
		return m
	}
	for _, d := range m.Canonical.Definitions {
		m.byID[d.Identity().Key()] = d
	}
	root, ok := m.byID[s.Constitution.Key()]
	if !ok || root.APIVersion != APIVersion || root.Kind != "Constitution" {
		add("constitution.identity", s.Constitution.Key(), "selected Constitution must resolve exactly")
		return m
	}
	m.Root = identity(root.Spec["root"])
	m.Cabinet = identities(root.Spec["cabinet"])
	sortIDs(m.Cabinet)
	parents := map[string]string{}
	pathOwners := map[string]string{}
	responsible := map[string]string{}
	for _, d := range m.Canonical.Definitions {
		if d.APIVersion != APIVersion {
			continue
		}
		key := d.Identity().Key()
		switch d.Kind {
		case "Constitution":
			if key != s.Constitution.Key() {
				add("constitution.multiple", key, "one active Constitution per source")
			}
		case "Area":
			parent := optionalIdentity(d.Spec["parent"])
			parents[key] = parent
			if key == m.Root.Key() && parent != "" {
				add("area.root-parent", key, "root cannot have a parent")
			}
			if key != m.Root.Key() && parent == "" {
				add("area.disconnected", key, "non-root Area requires a parent")
			}
		case "Mandate":
			for _, id := range identities(d.Spec["scope"]) {
				m.checkSubject(id, key, add)
			}
			parent := optionalIdentity(d.Spec["parent"])
			area := identity(d.Spec["area"])
			if parent == "" && area.Key() != m.Root.Key() {
				add("mandate.undelegated", key, "non-root mandate requires prior parent delegation")
			}
			if parent != "" {
				p := m.byID[parent]
				if !subsetIDs(identities(d.Spec["scope"]), identities(p.Spec["scope"])) || !subsetStrings(stringsList(d.Spec["actions"]), stringsList(p.Spec["actions"])) {
					add("mandate.expansion", key, "child scope/actions exceed higher delegation")
				}
				if !m.isDescendant(area.Key(), identity(p.Spec["area"]).Key()) {
					add("mandate.area", key, "delegation must remain within the parent Area hierarchy")
				}
			}
		case "Responsibility":
			subject := identity(d.Spec["subject"])
			m.checkSubject(subject, key, add)
			if owner, exists := responsible[subject.Key()]; exists {
				add("responsibility.conflict", subject.Key(), "multiple claims: "+owner+" and "+key)
			}
			responsible[subject.Key()] = key
		case "Realization":
			m.checkSubject(identity(d.Spec["subject"]), key, add)
		case "Artifact":
			p, _ := d.Spec["path"].(string)
			if !SafePath(p) {
				add("artifact.path", key, "expected exact normalized repository-relative path")
			}
			if prior, exists := pathOwners[strings.ToLower(p)]; exists {
				add("artifact.conflict", p, "multiple path declarations: "+prior+" and "+key)
			}
			pathOwners[strings.ToLower(p)] = key
			writer := optionalIdentity(d.Spec["writer"])
			if d.Spec["class"] == "foreign" && writer != "" {
				add("artifact.foreign-writer", p, "foreign material cannot grant a writer")
			}
			if d.Spec["class"] != "foreign" && writer == "" {
				add("artifact.writer-missing", p, "managed material needs one explicit Area writer")
			}
		case "Capability":
			if !validDigest(fmt.Sprint(d.Spec["digest"])) {
				add("capability.digest", key, "exact sha256 digest required")
			}
		}
	}
	for _, d := range m.Canonical.Definitions {
		if d.APIVersion != APIVersion {
			continue
		}
		key := d.Identity().Key()
		if d.Kind == "Area" && !reaches(key, m.Root.Key(), parents) {
			add("area.cycle", key, "Area hierarchy must reach the root without a cycle")
		}
		if d.Kind == "Mandate" {
			seen := map[string]bool{}
			cur := key
			for cur != "" {
				if seen[cur] {
					add("mandate.cycle", key, "delegation cannot cycle")
					break
				}
				seen[cur] = true
				cur = optionalIdentity(m.byID[cur].Spec["parent"])
			}
		}
		if d.Kind == "Ressort" {
			mandate := m.byID[identity(d.Spec["mandate"]).Key()]
			if !contains(stringsList(mandate.Spec["actions"]), "review") {
				add("ressort.mandate", key, "Ressort mandate must permit review")
			}
		}
	}
	m.Digest = Digest(struct {
		Version      string
		Constitution core.DefinitionIdentity
		Model        string
	}{APIVersion, m.Constitution, m.Canonical.Digest})
	sortFindings(m.Findings)
	return m
}

func (m Model) checkSubject(id core.DefinitionIdentity, relation string, add func(string, string, string)) {
	if _, ok := m.byID[id.Key()]; !ok {
		add("subject.unresolved", relation, "unknown exact identity "+id.Key())
	}
}
func SafePath(p string) bool {
	if p == "" || p == "." || strings.ContainsAny(p, "\\:\x00") || strings.HasPrefix(p, "/") || path.Clean(p) != p || p == ".." || strings.HasPrefix(p, "../") {
		return false
	}
	for _, part := range strings.Split(p, "/") {
		if strings.EqualFold(part, ".git") || strings.HasSuffix(part, ".") || strings.HasSuffix(part, " ") {
			return false
		}
		stem := strings.ToUpper(strings.SplitN(part, ".", 2)[0])
		if stem == "CON" || stem == "PRN" || stem == "AUX" || stem == "NUL" || len(stem) == 4 && (strings.HasPrefix(stem, "COM") || strings.HasPrefix(stem, "LPT")) && stem[3] >= '1' && stem[3] <= '9' {
			return false
		}
	}
	return true
}
func identity(v any) core.DefinitionIdentity {
	b, _ := json.Marshal(v)
	var id core.DefinitionIdentity
	_ = json.Unmarshal(b, &id)
	return id
}
func identities(v any) []core.DefinitionIdentity {
	values, _ := v.([]any)
	out := make([]core.DefinitionIdentity, 0, len(values))
	for _, v := range values {
		out = append(out, identity(v))
	}
	return out
}
func optionalIdentity(v any) string {
	if v == nil {
		return ""
	}
	return identity(v).Key()
}
func stringsList(v any) []string {
	values, _ := v.([]any)
	out := make([]string, 0, len(values))
	for _, v := range values {
		out = append(out, fmt.Sprint(v))
	}
	return out
}
func contains(values []string, v string) bool {
	for _, x := range values {
		if x == v {
			return true
		}
	}
	return false
}
func subsetStrings(a, b []string) bool {
	for _, v := range a {
		if !contains(b, v) {
			return false
		}
	}
	return true
}
func subsetIDs(a, b []core.DefinitionIdentity) bool {
	keys := []string{}
	for _, id := range b {
		keys = append(keys, id.Key())
	}
	for _, id := range a {
		if !contains(keys, id.Key()) {
			return false
		}
	}
	return true
}
func sortIDs(ids []core.DefinitionIdentity) {
	sort.Slice(ids, func(i, j int) bool { return ids[i].Key() < ids[j].Key() })
}
func sortFindings(f []Finding) {
	sort.Slice(f, func(i, j int) bool {
		a, b := f[i], f[j]
		if a.Code != b.Code {
			return a.Code < b.Code
		}
		if a.Subject != b.Subject {
			return a.Subject < b.Subject
		}
		return a.Detail < b.Detail
	})
}
func reaches(from, to string, parents map[string]string) bool {
	seen := map[string]bool{}
	for from != "" {
		if from == to {
			return true
		}
		if seen[from] {
			return false
		}
		seen[from] = true
		from = parents[from]
	}
	return false
}
func (m Model) isDescendant(child, parent string) bool {
	parents := map[string]string{}
	for k, d := range m.byID {
		if d.Kind == "Area" && d.APIVersion == APIVersion {
			parents[k] = optionalIdentity(d.Spec["parent"])
		}
	}
	return reaches(child, parent, parents)
}
func validDigest(s string) bool {
	if len(s) != 71 || !strings.HasPrefix(s, "sha256:") {
		return false
	}
	_, err := hex.DecodeString(s[7:])
	return err == nil && s == strings.ToLower(s)
}
