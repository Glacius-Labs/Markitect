package library

import (
	"io/fs"
	"path"
	"sort"
	"strings"
	"testing"

	"go.yaml.in/yaml/v3"
)

func TestEmbedsPackagesAndTemplates(t *testing.T) {
	folders := func(dir string) []string {
		entries, err := fs.ReadDir(FS(), dir)
		if err != nil {
			t.Fatal(err)
		}
		var out []string
		for _, e := range entries {
			if e.IsDir() {
				out = append(out, e.Name())
			}
		}
		sort.Strings(out)
		return out
	}
	if got := strings.Join(folders("packages"), " "); got != "company core factory software" {
		t.Errorf("packages: %s", got)
	}
	if got := strings.Join(folders("templates"), " "); got != "architecture company factory product project software" {
		t.Errorf("templates: %s", got)
	}
	if got := strings.Join(folders("."), " "); got != "packages questions references templates" {
		t.Errorf("the library holds %s", got)
	}
}

// The skills lie apart from the library: they are plain files, and a skill
// starts with front matter that no document of the language has.
func TestSkillsAreFoldersWithASkillFileAndItsName(t *testing.T) {
	names, err := SkillNames()
	if err != nil {
		t.Fatal(err)
	}
	if len(names) == 0 {
		t.Fatal("no skills")
	}
	for _, name := range names {
		data, err := fs.ReadFile(Skills(), name+"/SKILL.md")
		if err != nil {
			t.Errorf("%s: %v", name, err)
			continue
		}
		text := strings.ReplaceAll(string(data), "\r\n", "\n")
		rest, ok := strings.CutPrefix(text, "---\n")
		head, _, closed := strings.Cut(rest, "\n---\n")
		if !ok || !closed {
			t.Errorf("%s/SKILL.md: no front matter between --- lines", name)
			continue
		}
		var fields struct {
			Name        string `yaml:"name"`
			Description string `yaml:"description"`
		}
		if err := yaml.Unmarshal([]byte(head), &fields); err != nil {
			t.Errorf("%s/SKILL.md: front matter: %v", name, err)
		}
		if fields.Name != name {
			t.Errorf("%s/SKILL.md: name is %q, not the folder's name", name, fields.Name)
		}
		if strings.TrimSpace(fields.Description) == "" {
			t.Errorf("%s/SKILL.md: no description", name)
		}
	}
}

// There is one skill for each Agent document of the factory template, named
// markitect- and the file name of the document, no more and no fewer.
func TestSkillsAreTheAgentsOfTheFactoryTemplate(t *testing.T) {
	entries, err := fs.ReadDir(FS(), "templates/factory/agents")
	if err != nil {
		t.Fatal(err)
	}
	var agents []string
	for _, e := range entries {
		if name, ok := strings.CutSuffix(e.Name(), ".md"); ok && name != "README" {
			agents = append(agents, "markitect-"+name)
		}
	}
	sort.Strings(agents)
	skills, err := SkillNames()
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(skills, " ") != strings.Join(agents, " ") {
		t.Errorf("skills %v, agents %v", skills, agents)
	}
}

func TestEveryFileIsADocumentWithATitleAndLineFeeds(t *testing.T) {
	n := 0
	err := fs.WalkDir(FS(), ".", func(p string, e fs.DirEntry, err error) error {
		if err != nil || e.IsDir() {
			return err
		}
		n++
		data, err := fs.ReadFile(FS(), p)
		if err != nil {
			return err
		}
		text := string(data)
		if path.Ext(p) != ".md" || !strings.HasPrefix(text, "# ") || strings.Contains(text, "\r") || !strings.HasSuffix(text, "\n") {
			t.Errorf("%s is not a document with a title, LF line ends and a final line break", p)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if n < 40 {
		t.Errorf("only %d files", n)
	}
}

func TestEveryPackageHasReferencesAndQuestions(t *testing.T) {
	entries, err := fs.ReadDir(FS(), "packages")
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		refs, err := fs.ReadFile(FS(), "references/"+e.Name()+".md")
		if err != nil {
			t.Fatal(err)
		}
		types, _ := fs.Glob(FS(), "packages/"+e.Name()+"/*.md")
		if rows := strings.Count(string(refs), "\n| ") - 1; rows != len(types) {
			t.Errorf("%s: %d reference rows for %d types", e.Name(), rows, len(types))
		}
		questions, err := fs.ReadFile(FS(), "questions/"+e.Name()+".md")
		if err != nil {
			t.Fatal(err)
		}
		if n := strings.Count(string(questions), "\n- "); n < 5 || n > 10 {
			t.Errorf("%s: %d questions", e.Name(), n)
		}
	}
}
