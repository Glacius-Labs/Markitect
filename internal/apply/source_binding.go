package apply

import (
	"bytes"
	"fmt"

	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

// Match the two raw representations without collapsing the Model check's source
// slice: duplicate paths, redirects and malformed source entries still reach
// Check unchanged and receive its language diagnostics.
func validateModelFiles(source ModelSource, census snapshot.Tree) error {
	regular := make(map[string]bool)
	for _, entry := range source.Source.Entries {
		if entry.Kind != model.Regular {
			continue
		}
		p, err := snapshot.NewPath(entry.Path)
		if err != nil {
			continue // The Model check diagnoses malformed language input paths.
		}
		file, found := source.Files.Lookup(p)
		if !found || !bytes.Equal(file.Bytes(), entry.Bytes) {
			return fmt.Errorf("%w: model bytes differ from fixed files at %q", ErrIncompleteInputs, entry.Path)
		}
		regular[entry.Path] = true
	}
	for _, file := range source.Files.Entries() {
		if !regular[file.Path().String()] {
			return fmt.Errorf("%w: fixed model file absent from source at %q", ErrIncompleteInputs, file.Path())
		}
		if source.RepositoryArea.String() != "" {
			p := file.Path().String()
			if !source.RepositoryArea.IsRoot() {
				p = source.RepositoryArea.String() + "/" + p
			}
			if !sameCensusFile(census, p, file) {
				return fmt.Errorf("%w: model files differ from repository census at %q", ErrIncompleteInputs, p)
			}
		}
	}
	return nil
}

func sameCensusFile(census snapshot.Tree, p string, file snapshot.Entry) bool {
	path, err := snapshot.NewPath(p)
	if err != nil {
		return false
	}
	actual, found := census.Lookup(path)
	return found && actual.Mode() == file.Mode() && bytes.Equal(actual.Bytes(), file.Bytes())
}
