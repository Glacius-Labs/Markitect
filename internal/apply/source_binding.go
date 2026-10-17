package apply

import (
	"bytes"
	"fmt"

	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

// Match the two raw representations without collapsing the compiler's source
// slice: duplicate paths, redirects and malformed source entries still reach
// Compile unchanged and receive its language diagnostics.
func validateModelFiles(source ModelSource) error {
	regular := make(map[string]bool)
	for _, entry := range source.Source.Entries {
		if entry.Kind != model.Regular {
			continue
		}
		p, err := snapshot.NewPath(entry.Path)
		if err != nil {
			continue // The compiler diagnoses malformed language input paths.
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
	}
	return nil
}

func sameCensusFile(raw RawInputs, p string, file snapshot.Entry) bool {
	path, err := snapshot.NewPath(p)
	if err != nil {
		return false
	}
	actual, found := raw.Census.Lookup(path)
	return found && actual.Mode() == file.Mode() && bytes.Equal(actual.Bytes(), file.Bytes())
}
