package edit

import (
	"io/fs"
	"path"
	"sort"
	"strconv"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/lang"
)

// Applied is what Apply computed: the documents of the template at their paths
// below the spec folder, the packages it copies and the change set.
type Applied struct {
	Documents []string
	Packages  []string
	Changes   Changes
}

// Apply copies the template templates/TEMPLATE of the library lib into the
// spec at target, a folder below it, "" for its root. Every package a copied
// document links to, and every package those link to, is copied whole to
// packages/NAME/ unless the spec has that folder already, which is never
// overwritten. Links into the library's packages are rewritten to the spec's
// copy; links between the copied documents stay. A link elsewhere in the
// library, a missing package or a path that exists refuses the whole apply.
func Apply(spec, lib fs.FS, templateArg, targetArg string) (Applied, error) {
	tpl, ok := folderPath(templateArg)
	if !ok || tpl == "" || !isDir(lib, "templates/"+tpl) {
		return Applied{}, refuse("unknown template: " + templateArg)
	}
	target, ok := folderPath(targetArg)
	if !ok {
		return Applied{}, refuse("invalid target: " + strconv.Quote(targetArg))
	}
	base := "templates/" + tpl
	sources, err := documents(lib, base)
	if err != nil {
		return Applied{}, err
	}
	copies := map[string]string{} // library path to spec path
	for _, src := range sources {
		copies[src] = path.Join(target, strings.TrimPrefix(src, base+"/"))
	}
	var problems, wanted []string
	retarget := func(src, dst string, data []byte) []byte {
		d := lang.Parse(src, data)
		fresh, err := rewrite(d, func(r lang.Ref) (string, bool) {
			if strings.HasPrefix(r.Target, "packages/") {
				wanted = append(wanted, strings.Split(r.Target, "/")[1])
				if dst == "" {
					return "", false
				}
				return rel(r.Target, dir(dst)), true
			}
			if _, ok := copies[r.Target]; !ok || dst == "" {
				problems = append(problems, src+": link leaves the template: "+r.Raw)
			}
			return "", false
		})
		if err != nil {
			problems = append(problems, err.Error())
		}
		return fresh
	}
	if len(sources) == 0 {
		problems = append(problems, base+"/: no documents")
	}
	a := Applied{}
	var writes []Write
	for _, src := range sources {
		data, err := fs.ReadFile(lib, src)
		if err != nil {
			return Applied{}, err
		}
		dst := copies[src]
		a.Documents = append(a.Documents, dst)
		writes = append(writes, Write{Path: dst, Data: retarget(src, dst, data), Create: true})
	}
	for n := 0; n < len(wanted); n++ {
		name := wanted[n]
		if contains(a.Packages, name) || exists(spec, "packages/"+name) {
			continue // the spec owns its copy
		}
		if !isDir(lib, "packages/"+name) {
			problems = append(problems, "packages/"+name+": no such package in the library")
			continue
		}
		a.Packages = append(a.Packages, name)
		files, err := allFiles(lib, "packages/"+name)
		if err != nil {
			return Applied{}, err
		}
		for _, f := range files {
			data, err := fs.ReadFile(lib, f)
			if err != nil {
				return Applied{}, err
			}
			if lang.IsDocumentName(path.Base(f)) {
				retarget(f, "", data)
			}
			writes = append(writes, Write{Path: f, Data: data, Create: true})
		}
	}
	for _, dst := range a.Documents {
		if exists(spec, dst) {
			problems = append(problems, dst+": already exists")
		}
	}
	if target != "" && exists(spec, target) && !isDir(spec, target) {
		problems = append(problems, "not a folder: "+target)
	}
	if len(problems) > 0 {
		return Applied{}, refuse(unique(problems)...)
	}
	a.Changes.Writes = writes
	return a, nil
}

func isDir(fsys fs.FS, p string) bool {
	info, err := fs.Stat(fsys, p)
	return err == nil && info.IsDir()
}

// documents returns the paths of the documents below a folder, sorted.
func documents(fsys fs.FS, root string) ([]string, error) {
	files, err := allFiles(fsys, root)
	var out []string
	for _, f := range files {
		if lang.IsDocumentName(path.Base(f)) {
			out = append(out, f)
		}
	}
	return out, err
}

// allFiles returns the paths of the files below a folder, sorted.
func allFiles(fsys fs.FS, root string) ([]string, error) {
	var out []string
	err := fs.WalkDir(fsys, root, func(p string, e fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !e.IsDir() {
			out = append(out, p)
		}
		return nil
	})
	sort.Strings(out)
	return out, err
}

func contains(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}

func unique(list []string) []string {
	var out []string
	for _, s := range list {
		if !contains(out, s) {
			out = append(out, s)
		}
	}
	return out
}
