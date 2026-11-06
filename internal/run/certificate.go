package run

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/analysis"
	"github.com/Glacius-Labs/Markitect/internal/project"
)

// The certificate (DEC-097). When gate passes, it writes one under the results
// folder: the SHA-256 of every accounted file and every Definition, and the text
// of the spec it certified. The newest certificate in the folder is the base of
// the next change: "changed" means a different hash, and a Definition's old text
// comes from the copy.

// CertificateFormat is the version of the certificate's form.
const CertificateFormat = 1

// certificateKind marks a certificate among the results folder's JSON files.
const certificateKind = "markitect-certificate"

// Certificate is what gate writes when the work passes.
type Certificate struct {
	Kind    string    `json:"kind"`
	Format  int       `json:"format"`
	Created time.Time `json:"created"`
	// Base names the certificate the change started from, empty when there was
	// none; Full says every Definition was on the list.
	Base string `json:"base,omitempty"`
	Full bool   `json:"full,omitempty"`
	// Files holds the SHA-256 of every accounted file outside the results folder,
	// line endings aside.
	Files map[string]string `json:"files"`
	// Definitions holds the SHA-256 of every Definition's text.
	Definitions map[string]string `json:"definitions"`
	// Spec holds the text of every file in Markitect's folder outside the results
	// folder: the spec and its registry, as certified.
	Spec map[string]string `json:"spec"`
	// Items holds each checklist item with its answer.
	Items []ResultItem `json:"items"`
}

// gatePrefix starts the name of every page gate writes.
const gatePrefix = "gate-"

// gateStamp names what gate writes for a state: when, and the start of the
// state's fingerprint.
func gateStamp(at time.Time, fingerprint string) string {
	return gatePrefix + at.UTC().Format("20060102-150405") + "-" + fingerprint[:8]
}

// isResult says whether a file lies in the results folder, which no certificate
// counts: it holds the records, not the work.
func isResult(file string) bool {
	return strings.HasPrefix(file, project.ResultsDir+"/")
}

// fileHash is the SHA-256 of a file's content; in text, without a NUL byte, line
// endings do not count, so a checkout on another system has the same hashes.
func fileHash(data []byte) string {
	if bytes.IndexByte(data, 0) < 0 {
		data = project.Normalize(data)
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// hashFiles hashes every accounted file outside the results folder.
func hashFiles(repo project.Repository) (map[string]string, error) {
	files, err := repo.Files()
	if err != nil {
		return nil, err
	}
	out := map[string]string{}
	for _, f := range files {
		if isResult(f) {
			continue
		}
		data, err := repo.Read(f)
		if err != nil {
			return nil, err
		}
		out[f] = fileHash(data)
	}
	return out, nil
}

// fingerprint is one hash for a whole state of the folder.
func fingerprint(hashes map[string]string) string {
	h := sha256.New()
	for _, f := range sortedKeys(hashes) {
		fmt.Fprintf(h, "%s\x00%s\n", f, hashes[f])
	}
	return hex.EncodeToString(h.Sum(nil))
}

// newCertificate is the certificate of a state of the folder.
func newCertificate(repo project.Repository, hashes map[string]string, p *project.Project, created time.Time) (*Certificate, error) {
	c := &Certificate{Kind: certificateKind, Format: CertificateFormat, Created: created.UTC(), Files: hashes, Definitions: map[string]string{}, Spec: map[string]string{}}
	for id, d := range p.Model.Definitions {
		c.Definitions[id] = fileHash(d.Data)
	}
	for f := range hashes {
		if !strings.HasPrefix(f, project.Dir+"/") {
			continue
		}
		data, err := repo.Read(f)
		if err != nil {
			return nil, err
		}
		c.Spec[f] = string(project.Normalize(data))
	}
	return c, nil
}

// ReadCertificate reads a certificate file. The Result page beside it names it
// too.
func ReadCertificate(file string) (*Certificate, error) {
	if strings.HasSuffix(file, ".md") {
		file = strings.TrimSuffix(file, ".md") + ".json"
	}
	data, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}
	var c Certificate
	if err := json.Unmarshal(data, &c); err != nil || c.Kind != certificateKind || c.Files == nil {
		return nil, fmt.Errorf("%s is no certificate of markitect gate", file)
	}
	if c.Format != CertificateFormat {
		return nil, fmt.Errorf("%s has certificate format %d, and this markitect reads %d", file, c.Format, CertificateFormat)
	}
	return &c, nil
}

// newestCertificate finds the newest certificate in the results folder of the
// project at root: the latest one made, and of two made in the same second, the
// one that names the other as its base. It returns its project-relative path, or
// "" when there is none. A file there that is no certificate is not one.
func newestCertificate(root string) (string, *Certificate, error) {
	entries, err := os.ReadDir(filepath.Join(root, filepath.FromSlash(project.ResultsDir)))
	if errors.Is(err, os.ErrNotExist) {
		return "", nil, nil
	}
	if err != nil {
		return "", nil, err
	}
	var best string
	var bestCert *Certificate
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		file := path.Join(project.ResultsDir, e.Name())
		c, err := ReadCertificate(filepath.Join(root, filepath.FromSlash(file)))
		if err != nil {
			continue
		}
		if bestCert == nil || newer(c, file, bestCert, best) {
			best, bestCert = file, c
		}
	}
	return best, bestCert, nil
}

func newer(a *Certificate, aFile string, b *Certificate, bFile string) bool {
	switch {
	case !a.Created.Equal(b.Created):
		return a.Created.After(b.Created)
	case a.Base == bFile:
		return true
	case b.Base == aFile:
		return false
	}
	return aFile > bFile
}

// Project loads the project as the certificate holds it: its spec and registry,
// and the list of its files.
func (c *Certificate) Project() (*project.Project, error) {
	return project.Load(certified{c})
}

// Changed lists the files outside Markitect's folder whose hashes differ between
// the certificate and hashes, which is the folder now.
func (c *Certificate) Changed(hashes map[string]string) []analysis.Change {
	var out []analysis.Change
	for f, h := range hashes {
		switch was, ok := c.Files[f]; {
		case strings.HasPrefix(f, project.Dir+"/"):
		case !ok:
			out = append(out, analysis.Change{Path: f, Change: analysis.Added})
		case was != h:
			out = append(out, analysis.Change{Path: f, Change: analysis.Modified})
		}
	}
	for f := range c.Files {
		if _, ok := hashes[f]; !ok && !strings.HasPrefix(f, project.Dir+"/") {
			out = append(out, analysis.Change{Path: f, Change: analysis.Removed})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out
}

// Differences lists every file, Markitect's folder included, whose hash differs
// between the certificate and hashes.
func (c *Certificate) Differences(hashes map[string]string) []string {
	var out []string
	for f, h := range hashes {
		if c.Files[f] != h {
			out = append(out, f)
		}
	}
	for f := range c.Files {
		if _, ok := hashes[f]; !ok {
			out = append(out, f)
		}
	}
	sort.Strings(out)
	return out
}

// HashFiles hashes every accounted file of a repository state outside the results
// folder, as a certificate does.
func HashFiles(repo project.Repository) (map[string]string, error) { return hashFiles(repo) }

// certified is a certificate as a repository state: its files and the text of
// its spec.
type certified struct{ c *Certificate }

func (s certified) Files() ([]string, error) { return sortedKeys(s.c.Files), nil }

func (s certified) Read(file string) ([]byte, error) {
	if text, ok := s.c.Spec[file]; ok {
		return []byte(text), nil
	}
	return nil, fmt.Errorf("the certificate holds no text of %s", file)
}

func (s certified) GitIgnored(string) bool { return false }
