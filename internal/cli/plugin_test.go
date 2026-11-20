package cli

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/library"
	"github.com/Glacius-Labs/Markitect/internal/state"
	"go.yaml.in/yaml/v3"
)

// pluginDir is the Claude Code plugin of the rebuilt tool. Its skills and
// agents name the tool's commands, and the guide describes them.
const pluginDir = "../../plugin-next"

var (
	// toolWord is the word after the tool's name anywhere in a file, front
	// matter and prose included: it must be a command.
	toolWord = regexp.MustCompile(`markitect-next[ \t]+([A-Za-z][A-Za-z-]*)`)
	// pluginName is a skill or an agent named with the plugin's scope.
	pluginName = regexp.MustCompile(`markitect-next:([a-z0-9-]+)`)
	codeSpan   = regexp.MustCompile("`([^`\n]+)`")
	// relativeLink is the target of a Markdown link, which must lead to a
	// file when it is relative.
	relativeLink = regexp.MustCompile(`\]\(([^)\s]+)\)`)
	// rootPath is a file of the plugin as Claude Code substitutes its root
	// in a skill.
	rootPath = regexp.MustCompile(`\$\{CLAUDE_PLUGIN_ROOT\}/([^\s` + "`" + `]+)`)
	// invocation is the tool's name with what follows it in a piece of code,
	// up to a shell separator.
	invocation = regexp.MustCompile(`(?:^|[\s(])markitect-next((?:[ \t][^;|&>)\n]*)?)`)
	agentTools = []string{"Read", "Write", "Edit", "Glob", "Grep", "Bash", "WebFetch"}
	// jsonValue is a value of a command's JSON document, written
	// COMMAND.KEY in a code span, such as settings.tests.
	jsonValue = regexp.MustCompile(`^(status|settings|figures|reports)((?:\.[a-z_]+)+)$`)
	// placeholder is a word that stands for a value, such as ROLE.
	placeholder = regexp.MustCompile(`^[A-Z][A-Z_]*$`)
)

// jsonDocuments are the JSON documents of the commands whose values the
// plugin reads, by command.
var jsonDocuments = map[string][]any{
	"status":   {statusDocument{}, checksDocument{}},
	"settings": {settingsDocument{}},
	"figures":  {figuresDocument{}},
	"reports":  {reportsDocument{}},
}

// jsonKeys adds the key paths of a JSON document's type, such as
// .waiting.counts; a map takes any key below it, which "*" marks.
func jsonKeys(t reflect.Type, prefix string, keys map[string]bool) {
	for t.Kind() == reflect.Slice || t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	switch t.Kind() {
	case reflect.Map:
		keys[prefix+".*"] = true
		return
	case reflect.Struct:
	default:
		return
	}
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		name, _, _ := strings.Cut(f.Tag.Get("json"), ",")
		switch {
		case f.Anonymous && name == "":
			jsonKeys(f.Type, prefix, keys)
		case name != "" && name != "-":
			keys[prefix+"."+name] = true
			jsonKeys(f.Type, prefix+"."+name, keys)
		}
	}
}

// knownValue says whether a key path is in a command's JSON document.
func knownValue(command, path string) bool {
	keys := map[string]bool{}
	for _, doc := range jsonDocuments[command] {
		jsonKeys(reflect.TypeOf(doc), "", keys)
	}
	if keys[path] {
		return true
	}
	for p := path; strings.Contains(p, "."); {
		p = p[:strings.LastIndex(p, ".")]
		if keys[p+".*"] {
			return true
		}
	}
	return false
}

// pluginFiles are the files the test reads: every skill, every agent, the
// frame of the jobs and the guide to the plugin.
func pluginFiles(t *testing.T) (skills, agents, all []string) {
	t.Helper()
	var err error
	if skills, err = filepath.Glob(filepath.Join(pluginDir, "skills", "*", "SKILL.md")); err != nil {
		t.Fatal(err)
	}
	if agents, err = filepath.Glob(filepath.Join(pluginDir, "agents", "*.md")); err != nil {
		t.Fatal(err)
	}
	all = append(append(append(all, skills...), agents...), filepath.Join(pluginDir, "jobs.md"), "../../docs/guides/plugin.md")
	return skills, agents, all
}

// frontMatter parses a file's YAML front matter into its fields, and returns
// the text after it.
func frontMatter(t *testing.T, path string) (map[string]any, string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	text := strings.ReplaceAll(string(data), "\r\n", "\n")
	rest, ok := strings.CutPrefix(text, "---\n")
	head, body, closed := strings.Cut(rest, "\n---\n")
	if !ok || !closed {
		t.Errorf("%s: no front matter between --- lines", path)
		return nil, text
	}
	fields := map[string]any{}
	if err := yaml.Unmarshal([]byte(head), &fields); err != nil {
		t.Errorf("%s: front matter: %v", path, err)
	}
	return fields, body
}

func field(t *testing.T, path string, fields map[string]any, key string) string {
	t.Helper()
	v, ok := fields[key].(string)
	if !ok || strings.TrimSpace(v) == "" {
		t.Errorf("%s: front matter has no %s", path, key)
	}
	return v
}

func onlyKeys(t *testing.T, path string, fields map[string]any, allowed ...string) {
	t.Helper()
	for k := range fields {
		if !slices.Contains(allowed, k) {
			t.Errorf("%s: front matter has %s, which is none of %s", path, k, strings.Join(allowed, ", "))
		}
	}
}

// The plugin holds an agent for each role of the tool and a skill per job and
// reading entry, each with valid front matter.
func TestPluginFrontMatter(t *testing.T) {
	data, err := os.ReadFile(filepath.Join(pluginDir, ".claude-plugin", "plugin.json"))
	if err != nil {
		t.Fatal(err)
	}
	var manifest struct {
		Name, Version, Description string
	}
	if err := json.Unmarshal(data, &manifest); err != nil || manifest.Name != "markitect-next" || manifest.Version == "" || manifest.Description == "" {
		t.Errorf("plugin.json: %v, %+v", err, manifest)
	}
	skills, agents, _ := pluginFiles(t)
	if len(skills) < 13 {
		t.Errorf("found %d skills; the plugin has a skill per job and reading entry", len(skills))
	}
	roles := map[string]bool{}
	for _, path := range agents {
		fields, body := frontMatter(t, path)
		onlyKeys(t, path, fields, "name", "description", "tools", "model")
		name := field(t, path, fields, "name")
		if want := strings.TrimSuffix(filepath.Base(path), ".md"); name != want {
			t.Errorf("%s: name %q, not %q", path, name, want)
		}
		if !slices.Contains(state.Roles, name) {
			t.Errorf("%s: %s is no role of the tool", path, name)
		}
		roles[name] = true
		field(t, path, fields, "description")
		for _, tool := range strings.Split(field(t, path, fields, "tools"), ",") {
			if !slices.Contains(agentTools, strings.TrimSpace(tool)) {
				t.Errorf("%s: tool %q is none of %s", path, strings.TrimSpace(tool), strings.Join(agentTools, ", "))
			}
		}
		if m := field(t, path, fields, "model"); !slices.Contains([]string{"opus", "sonnet", "haiku"}, m) {
			t.Errorf("%s: model %q", path, m)
		}
		if strings.TrimSpace(body) == "" {
			t.Errorf("%s: no standing text", path)
		}
	}
	for _, role := range state.Roles {
		if !roles[role] {
			t.Errorf("the plugin has no agent for the role %s", role)
		}
	}
	for _, path := range skills {
		fields, body := frontMatter(t, path)
		onlyKeys(t, path, fields, "name", "description", "allowed-tools", "disable-model-invocation", "argument-hint")
		if name, want := field(t, path, fields, "name"), filepath.Base(filepath.Dir(path)); name != want {
			t.Errorf("%s: name %q, not %q", path, name, want)
		}
		field(t, path, fields, "description")
		if v, ok := fields["disable-model-invocation"]; ok {
			if _, ok := v.(bool); !ok {
				t.Errorf("%s: disable-model-invocation is no boolean", path)
			}
		}
		if strings.TrimSpace(body) == "" {
			t.Errorf("%s: no text", path)
		}
	}
}

// code returns the pieces of code in a Markdown text: each line of a fenced
// block and each code span.
func code(text string) []string {
	var pieces []string
	fence := 0 // the length of the open fence, 0 outside a block
	for _, line := range strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n") {
		trimmed := strings.TrimSpace(line)
		ticks := len(trimmed) - len(strings.TrimLeft(trimmed, "`"))
		switch {
		case fence == 0 && ticks >= 3:
			fence = ticks
			continue
		case fence > 0 && ticks >= fence && strings.Trim(trimmed, "`") == "":
			fence = 0
			continue
		case fence > 0:
			pieces = append(pieces, line)
			continue
		}
		for _, m := range codeSpan.FindAllStringSubmatch(line, -1) {
			pieces = append(pieces, m[1])
		}
	}
	return pieces
}

// fields splits a command line into its words, keeping what stands between
// double quotes together.
func words(line string) []string {
	var out []string
	var b strings.Builder
	quoted, started := false, false
	for _, r := range line {
		switch {
		case r == '"':
			quoted, started = !quoted, true
		case (r == ' ' || r == '\t') && !quoted:
			if started {
				out = append(out, b.String())
				b.Reset()
				started = false
			}
		default:
			b.WriteRune(r)
			started = true
		}
	}
	if started {
		out = append(out, b.String())
	}
	return out
}

// Every command line of the tool that the plugin and its guide name parses as
// the tool parses it, with a command the tool has and only the flags of that
// command; every word after the tool's name is a command; and every skill or
// agent named with the plugin's scope exists, and so does every file a
// relative link or path below the plugin's root leads to. So a skill cannot
// drift from
// the command line it drives.
func TestPluginNamesOnlyWhatExists(t *testing.T) {
	skills, agents, all := pluginFiles(t)
	known := map[string]bool{}
	for _, path := range skills {
		known[filepath.Base(filepath.Dir(path))] = true
	}
	for _, path := range agents {
		known[strings.TrimSuffix(filepath.Base(path), ".md")] = true
	}
	lines, names, values := 0, 0, 0
	for _, path := range all {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		text := string(data)
		for _, m := range toolWord.FindAllStringSubmatch(text, -1) {
			if _, ok := commands[m[1]]; !ok && m[1] != "help" {
				t.Errorf("%s: markitect-next has no command %q", path, m[1])
			}
		}
		for _, m := range relativeLink.FindAllStringSubmatch(text, -1) {
			target, _, _ := strings.Cut(m[1], "#")
			if strings.Contains(target, ":") || target == "" {
				continue
			}
			if _, err := os.Stat(filepath.Join(filepath.Dir(path), target)); err != nil {
				t.Errorf("%s: the link to %s leads nowhere", path, m[1])
			}
		}
		for _, m := range rootPath.FindAllStringSubmatch(text, -1) {
			if _, err := os.Stat(filepath.Join(pluginDir, m[1])); err != nil {
				t.Errorf("%s: the plugin has no file %s", path, m[1])
			}
		}
		for _, m := range pluginName.FindAllStringSubmatch(text, -1) {
			names++
			if !known[m[1]] {
				t.Errorf("%s: the plugin has no skill or agent %q", path, m[1])
			}
		}
		for _, piece := range code(text) {
			if m := jsonValue.FindStringSubmatch(piece); m != nil && !strings.HasSuffix(piece, ".md") && !strings.HasSuffix(piece, ".jsonl") {
				values++
				if !knownValue(m[1], m[2]) {
					t.Errorf("%s: `%s`: the JSON of %s has no such value", path, piece, m[1])
				}
			}
			for _, m := range invocation.FindAllStringSubmatch(piece, -1) {
				args := words(m[1])
				if len(args) == 0 || len(args) == 1 && args[0] == "help" {
					continue
				}
				lines++
				c, ok := commands[args[0]]
				if !ok {
					t.Errorf("%s: `markitect-next%s`: no such command", path, m[1])
					continue
				}
				o, err := parse(args[1:])
				if err == nil {
					err = validate(args[0], c, o)
				}
				if err == nil && o.step != "" && !placeholder.MatchString(o.step) && !slices.Contains(state.StepRoles, o.step) {
					err = errors.New("--step " + o.step + " is no role of a step")
				}
				if err == nil && args[0] == "apply" && !placeholder.MatchString(o.positional[0]) {
					if info, statErr := fs.Stat(library.FS(), "templates/"+o.positional[0]); statErr != nil || !info.IsDir() {
						err = errors.New(o.positional[0] + " is no template of the library")
					}
				}
				if err != nil {
					first, _, _ := strings.Cut(err.Error(), "\n")
					t.Errorf("%s: `markitect-next%s`: %s", path, m[1], first)
				}
			}
		}
	}
	if lines < 40 || names < 10 || values < 10 {
		t.Errorf("found %d command lines, %d scoped names and %d JSON values; the search no longer fits the plugin", lines, names, values)
	}
}
