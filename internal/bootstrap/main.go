// Package bootstrap constructs concrete adapters for the implemented product.
// Metadata preparation never constructs an Agent, Workspace or execution port.
package bootstrap

import (
	"context"
	"fmt"
	"io"
	"runtime"

	"github.com/Glacius-Labs/Markitect/internal/adapters/checks"
	"github.com/Glacius-Labs/Markitect/internal/adapters/cli"
	"github.com/Glacius-Labs/Markitect/internal/adapters/codexappserver"
	"github.com/Glacius-Labs/Markitect/internal/adapters/documents"
	"github.com/Glacius-Labs/Markitect/internal/adapters/gitrepo"
	"github.com/Glacius-Labs/Markitect/internal/modeldocs"
	"github.com/Glacius-Labs/Markitect/internal/project"
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
	agents := codexappserver.NewMetadataPreparer(images)
	checker := checks.NewMetadataPreparer(checks.NewEngineImageReaders().Reader)
	execution := applyExecution(goos, preparation, repository, agents, checker)
	return cli.New(preparation, agents, checker, repository, views, execution).Run(ctx, args, stdout, stderr)
}
