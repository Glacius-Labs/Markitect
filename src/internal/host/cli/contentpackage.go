package cli

import (
	"fmt"
	"os"

	"github.com/Glacius-Labs/Markitect/src/internal/host"
	"github.com/Glacius-Labs/Markitect/src/internal/infrastructure/source"
)

func runPack(root, revision, output string, emit func(any) int, fail func(error) int) int {
	if revision == "" || output == "" {
		return fail(fmt.Errorf("pack requires --revision and --output pointing to an absent ZIP file"))
	}
	snapshot, err := source.Load(root, revision)
	if err != nil {
		return fail(err)
	}
	archive, pin, err := host.PackContent(snapshot, "git:"+snapshot.ID)
	if err != nil {
		return fail(err)
	}
	f, err := os.OpenFile(output, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if err != nil {
		return fail(err)
	}
	_, writeErr := f.Write(archive)
	if writeErr == nil {
		writeErr = f.Sync()
	}
	closeErr := f.Close()
	if writeErr != nil || closeErr != nil {
		_ = os.Remove(output) // Only the incomplete file exclusively created here.
		if writeErr != nil {
			return fail(writeErr)
		}
		return fail(closeErr)
	}
	return emit(map[string]any{
		"status": "packed", "sourceCommit": snapshot.ID, "output": output,
		"pin": pin, "notice": "Archive integrity does not authenticate a publisher. Review the package, vendor these exact bytes, and set the chosen archive path and provenance in the Project pin.",
	})
}
