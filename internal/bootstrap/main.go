// Package bootstrap constructs concrete adapters for the implemented product.
// Metadata preparation never constructs an Agent, Workspace or execution port.
package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"io"
	"runtime"

	"github.com/Glacius-Labs/Markitect/internal/adapters/checks"
	"github.com/Glacius-Labs/Markitect/internal/adapters/claudecode"
	"github.com/Glacius-Labs/Markitect/internal/adapters/cli"
	"github.com/Glacius-Labs/Markitect/internal/adapters/codexappserver"
	"github.com/Glacius-Labs/Markitect/internal/adapters/documents"
	"github.com/Glacius-Labs/Markitect/internal/adapters/gitrepo"
	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/modeldocs"
	"github.com/Glacius-Labs/Markitect/internal/project"
	"github.com/Glacius-Labs/Markitect/internal/runtimeconfig"
)

func Run(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	return run(ctx, runtime.GOOS, args, stdout, stderr, codexappserver.NewEngineImageReader)
}

// run composes the product for the host operating system goos.
func run(ctx context.Context, goos string, args []string, stdout, stderr io.Writer, images codexappserver.ImageReaderFactory) int {
	repository, err := gitrepo.New(gitrepo.Limits{})
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 3
	}
	models := project.NewModelService(repository, repository)
	preparation := project.NewPreparationService(models)
	views := modeldocs.New(models, preparation, documents.Writer{})
	agents := agentMetadata{
		"codexAppServer": codexappserver.NewMetadataPreparer(images),
		// One injected engine reader serves both adapters' preparations.
		"claudeCode": claudecode.NewMetadataPreparer(func(c claudecode.EngineConfig) (claudecode.ImageReader, error) {
			return images(codexappserver.EngineConfig{Executable: c.Executable})
		}),
	}
	checker := checks.NewMetadataPreparer(checks.NewEngineImageReaders().Reader)
	execution := applyExecution(goos, preparation, repository, agents, checker)
	return cli.New(preparation, agents, checker, repository, views, execution).Run(ctx, args, stdout, stderr)
}

// agentMetadata routes each Agent profile to the metadata preparer of its
// section. The Runtime profile decoder admits exactly one known section per
// profile, so a missing entry is an incomplete composition.
type agentMetadata map[string]runtimeconfig.AgentProfilePreparer

func (a agentMetadata) PrepareAgentProfile(ctx context.Context, input runtimeconfig.ProfileInput) (runtimeconfig.ProfileObservation, error) {
	if preparer := a[input.Section()]; preparer != nil {
		return preparer.PrepareAgentProfile(ctx, input)
	}
	return runtimeconfig.ProfileObservation{InputBinding: input.Binding(), Status: model.NotCompleted}, errors.New("no metadata preparer is composed for the selected Agent section")
}
