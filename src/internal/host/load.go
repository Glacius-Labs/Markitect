package host

import "github.com/Glacius-Labs/Markitect/src/internal/infrastructure/source"

// Load resolves the CLI's Git revision or provisional directory through the
// source adapter. Parse consumes the resulting snapshot independently.
func Load(root, revision string) (*Project, error) {
	snap, err := source.Load(root, revision)
	if err != nil {
		return nil, err
	}
	return Parse(snap)
}
