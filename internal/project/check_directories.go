package project

import (
	"path"
	"slices"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

// CheckDirectories retains the fixed model's directory intent separately from
// the file-only Census. External model sources remain captured inputs outside
// the product tree and are never vendored into a command copy.
func CheckDirectories(raw RawInputs) ([]snapshot.Path, error) {
	set := map[string]bool{}
	for _, m := range append([]ModelSource{raw.Local}, raw.Dependencies...) {
		if m.RepositoryArea.String() == "" {
			continue
		}
		for _, e := range m.Source.Entries {
			if e.Kind == model.Directory {
				name := path.Join(m.RepositoryArea.String(), e.Path)
				if name != "." && name != "" {
					set[name] = true
				}
			}
		}
	}
	var names []string
	for name := range set {
		names = append(names, name)
	}
	slices.SortFunc(names, strings.Compare)
	var result []snapshot.Path
	for _, name := range names {
		p, err := snapshot.NewPath(name)
		if err != nil {
			return nil, err
		}
		result = append(result, p)
	}
	return result, nil
}
