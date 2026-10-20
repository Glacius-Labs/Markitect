package bootstrap

import (
	"context"
	"errors"
	"path/filepath"

	"github.com/Glacius-Labs/Markitect/internal/adapters/cli"
	"github.com/Glacius-Labs/Markitect/internal/adapters/cockpit"
	"github.com/Glacius-Labs/Markitect/internal/apply"
	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

// cockpitService serves the cockpit over the selected state directories. A run
// configuration contributes its state directory; nothing else of it is used.
// Live views probe the state lock the way the answer command does, and an
// owner's answer goes through that command's owner channel operation.
type cockpitService struct {
	preparation *apply.PreparationService
}

func (c cockpitService) Serve(ctx context.Context, request cli.CockpitRequest, ready func(url string)) error {
	states := append([]string(nil), request.States...)
	if request.RunConfig != "" {
		cfg, err := loadRunConfig(ctx, request.RunConfig)
		if err != nil {
			return err
		}
		states = append(states, cfg.StateDirectory)
	}
	cfg := cockpit.Config{States: states, Liveness: stateHeld, Answer: writeAnswer}
	if request.Repository != "" {
		repository, err := filepath.Abs(request.Repository)
		if err != nil {
			return err
		}
		cfg.Project = func(ctx context.Context) (cockpit.ProjectInfo, error) {
			return c.project(ctx, repository)
		}
	}
	server, err := cockpit.New(cfg)
	if err != nil {
		return err
	}
	return server.Serve(ctx, request.Listen, ready)
}

// project observes the repository's identity by structural preparation of its
// working tree, as markitect check does: model, canonical digest, head and
// Packages. It writes nothing.
func (c cockpitService) project(ctx context.Context, repository string) (cockpit.ProjectInfo, error) {
	info := cockpit.ProjectInfo{Repository: repository}
	manifest, err := snapshot.NewPath(".markitect/model.yaml")
	if err != nil {
		return info, err
	}
	selection := apply.InputSelection{Repository: repository, Local: apply.ModelSelection{Boundary: ".", ManifestPath: manifest}}
	result, err := c.preparation.Preview(ctx, apply.PreviewRequest{Selection: selection}, model.Limits{})
	info.Status = string(result.Status())
	attempts := result.Attempts()
	if len(attempts) == 0 {
		return info, errors.Join(err, errors.New("structural preparation made no attempt"))
	}
	final := attempts[len(attempts)-1]
	info.Head = final.Model().Inputs().Head
	if selections, complete := final.Contexts().Selections(); complete {
		info.Packages = len(selections)
		if len(selections) > 0 {
			info.ModelID, info.ModelDigest = selections[0].ModelID(), selections[0].ModelDigest()
		}
	}
	return info, err
}
