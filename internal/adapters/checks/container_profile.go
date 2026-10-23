package checks

import (
	"fmt"
	"path"
	"slices"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/project"
	"github.com/Glacius-Labs/Markitect/internal/run"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

// ContainerProfile holds explicitly supplied Runtime execution values for a
// selected check profile. It neither decodes nor certifies profile metadata.
type ContainerProfile struct {
	ImageID        string
	Executable     string
	ArgumentPrefix []string
	Env            []string
	Limits         run.CheckLimits
	MaxOutputBytes int64
}

// Request combines the selected profile with a declared check and the actual
// immutable tree. The caller owns selecting this profile for check.Profile.
// Arguments remain literal, with the fixed prefix before project arguments.
// No defaults, environment lookup, process execution or metadata observation
// occurs here. Returned slices do not share the caller's backing arrays.
func (p ContainerProfile) Request(check project.ExecutionCheck, tree snapshot.Tree) (run.CheckRequest, error) {
	req := run.CheckRequest{
		ID: run.CheckID(check.ID.Value), Tree: tree,
		ImageID: p.ImageID, Executable: p.Executable,
		Args: slices.Concat(p.ArgumentPrefix, check.Args.Value), Env: slices.Clone(p.Env),
		Directory: check.Directory.Value, Limits: p.Limits,
		Timeout: check.Timeout.Value, MaxOutputBytes: p.MaxOutputBytes,
	}
	if !req.Valid() {
		return run.CheckRequest{}, fmt.Errorf("container check request is incomplete or invalid")
	}
	if path.Clean(req.Executable) != req.Executable || strings.ContainsRune(req.Executable, 0) {
		return run.CheckRequest{}, fmt.Errorf("check executable must be a clean absolute container path")
	}
	for _, arg := range req.Args {
		if strings.ContainsRune(arg, 0) {
			return run.CheckRequest{}, fmt.Errorf("check argument contains NUL")
		}
	}
	if req.Limits.Processes > int64(int(^uint(0)>>1)) {
		return run.CheckRequest{}, fmt.Errorf("check process limit cannot be represented")
	}
	return req, nil
}
