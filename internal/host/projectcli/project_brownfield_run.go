package projectcli

import (
	"context"
	"errors"
	"fmt"
	"github.com/Glacius-Labs/Markitect/internal/host/projectadoption"
	"github.com/Glacius-Labs/Markitect/internal/host/projectapp"
	"io"
	"os"
	"os/signal"
)

type brownfieldManagerRunInput = projectapp.BrownfieldManagerRunInput
type brownfieldManagerRunOutput = projectapp.BrownfieldManagerRunOutput
type brownfieldManagerAttempt = projectapp.BrownfieldManagerAttempt

func runBrownfieldManagerStage(opts options, out io.Writer, invoker projectadoption.ManagerRunInvoker) error {
	if opts.input == "" {
		return errors.New("Brownfield run requires --input")
	}
	sourceRoot := opts.sourceRepo
	if sourceRoot == "" {
		sourceRoot = opts.repo
	}
	data, err := readRecord(sourceRoot, opts.input)
	if err != nil {
		return err
	}
	var request brownfieldManagerRunInput
	if err := decodeClosedProjectJSON(data, &request); err != nil {
		return fmt.Errorf("decode Brownfield manager-run input: %w", err)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	output, runErr := (projectapp.Operations{}).BrownfieldRun(ctx, projectapp.BrownfieldRunOperation{Root: opts.repo, SourceRoot: sourceRoot, Revision: opts.revision, SessionID: opts.sessionID, Write: opts.write, ExpectedDigest: opts.expect, Request: request}, invoker)

	if runErr != nil {
		if output.Attempt == nil {
			return runErr
		}
		if err := writeJSON(out, output); err != nil {
			return fmt.Errorf("manager attempt %s is durable but its CLI result could not be emitted; inspect the Brownfield run journal and do not replay automatically: %w", output.Attempt.ID, err)
		}
		return &projectOutcomeError{code: 1, message: fmt.Sprintf("Brownfield manager stage did not complete; attempt %s is retained with status %s and will not be replayed automatically", output.Attempt.ID, output.Attempt.Status)}
	}
	if err := writeJSON(out, output); err != nil {
		if output.Attempt != nil {
			return fmt.Errorf("Brownfield manager attempt %s is durable but its CLI result could not be emitted; inspect the run journal before continuing: %w", output.Attempt.ID, err)
		}
		return err
	}
	return nil
}
