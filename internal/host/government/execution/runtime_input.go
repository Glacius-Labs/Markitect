package execution

import (
	"errors"
	"path/filepath"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/host/government/inventory"
	"github.com/Glacius-Labs/Markitect/internal/infrastructure/source"
)

// ReadRuntime acquires the caller-selected trusted runtime outside source and
// Git metadata. DecodeRuntime binds its complete values, not a mutable path.
func ReadRuntime(repo, path string) (Runtime, error) {
	if !filepath.IsAbs(path) {
		return Runtime{}, errors.New("runtime path must be absolute and external")
	}
	repo, err := filepath.Abs(repo)
	if err != nil {
		return Runtime{}, err
	}
	repo, err = realDirectory(repo)
	if err != nil {
		return Runtime{}, err
	}
	identity, err := source.IdentifyGit(repo)
	if err != nil {
		return Runtime{}, err
	}
	directory, err := realDirectory(filepath.Dir(path))
	if err != nil {
		return Runtime{}, err
	}
	path = filepath.Join(directory, filepath.Base(path))
	for _, root := range []string{repo, identity.CommonDir, identity.GitDir} {
		root, err = realDirectory(root)
		if err != nil {
			return Runtime{}, err
		}
		rel, err := filepath.Rel(root, path)
		if err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			return Runtime{}, errors.New("runtime file must be outside source and Git metadata")
		}
	}
	data, err := inventory.ReadInput(filepath.Dir(path), filepath.Base(path), maxRuntimeJSONBytes)
	if err != nil {
		return Runtime{}, err
	}
	return DecodeRuntime(data)
}
