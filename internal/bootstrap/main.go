// Package bootstrap constructs concrete adapters for the implemented product.
// Metadata preparation never constructs an Agent, Workspace or execution port.
package bootstrap

import (
	"context"
	"fmt"
	"io"

	"github.com/Glacius-Labs/Markitect/internal/adapters/checks"
	"github.com/Glacius-Labs/Markitect/internal/adapters/claudecode"
	"github.com/Glacius-Labs/Markitect/internal/adapters/cli"
	"github.com/Glacius-Labs/Markitect/internal/adapters/codexappserver"
	"github.com/Glacius-Labs/Markitect/internal/adapters/documents"
	"github.com/Glacius-Labs/Markitect/internal/adapters/gitrepo"
	"github.com/Glacius-Labs/Markitect/internal/apply"
	"github.com/Glacius-Labs/Markitect/internal/modeldocs"
)

func Run(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	return run(ctx, args, stdout, stderr, codexappserver.NewEngineImageReader)
}

func run(ctx context.Context, args []string, stdout, stderr io.Writer, images codexappserver.ImageReaderFactory) int {
	repository, err := gitrepo.New(gitrepo.Limits{})
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 3
	}
	models := apply.NewModelService(repository, repository)
	preparation := apply.NewPreparationService(models)
	views := modeldocs.New(models, preparation, documents.Writer{})
	agents := agentMetadataRouter{codex: codexappserver.NewMetadataPreparer(images), claude: claudecode.NewMetadataPreparer(claudecode.NewEngineImageReader)}
	checker := checks.NewMetadataPreparer(checks.NewEngineImageReaders().Reader)
	execution := newExecution(preparation, repository, agents, checker, stderr)
	return cli.New(preparation, agents, checker, repository, views, execution.Execute, answer).WithExtract(execution.Extract).WithSetup(newSetup(preparation, containerSetupEngine).Setup).WithCockpit(cockpitService{preparation: preparation, repository: repository, agents: agents, checks: checker}.Serve).Run(ctx, args, stdout, stderr)
}
