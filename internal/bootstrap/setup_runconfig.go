package bootstrap

import (
	"bytes"
	"encoding/json"
)

// defaultOwnerRequest is the neutral owner request when none is given.
const defaultOwnerRequest = "Reconcile the repository with its canonical model."

// setupCredentialRef is the reference that setup's Agent profiles use for the
// runtime token; run.json maps it to the credential file.
const setupCredentialRef = "markitect-claude-token"

// runConfigDocument is the formatVersion 2 run configuration as setup writes
// it. Its field names are those of the decoder's closed contract; the optional
// ownerDecisions and deadline are omitted when absent, because the decoder
// refuses an empty list or null.
type runConfigDocument struct {
	FormatVersion      int                         `json:"formatVersion"`
	RepositoryIdentity string                      `json:"repositoryIdentity"`
	ProductBuild       string                      `json:"productBuild"`
	OwnerRequest       string                      `json:"ownerRequest"`
	OwnerDecisions     []string                    `json:"ownerDecisions,omitempty"`
	StateDirectory     string                      `json:"stateDirectory"`
	Deadline           string                      `json:"deadline,omitempty"`
	EngineExecutable   string                      `json:"engineExecutable"`
	WorkspaceHelper    workspaceHelperDocument     `json:"workspaceHelper"`
	Harvest            harvestLimits               `json:"harvest"`
	Credentials        map[string]credentialSource `json:"credentials"`
}

// workspaceHelperDocument is the helper container of the shared workspace.
type workspaceHelperDocument struct {
	Name        string            `json:"name"`
	Image       string            `json:"image"`
	Labels      map[string]string `json:"labels"`
	User        string            `json:"user"`
	CPUs        float64           `json:"cpus"`
	MemoryBytes int64             `json:"memoryBytes"`
	Pids        int               `json:"pids"`
	TmpBytes    int64             `json:"tmpBytes"`
	Network     string            `json:"network"`
}

// encodeRunConfig renders the document deterministically (indented, sorted map
// keys, no HTML escaping, one final newline) and checks the result with the
// decoder that Apply uses.
func encodeRunConfig(document runConfigDocument) ([]byte, error) {
	var out bytes.Buffer
	encoder := json.NewEncoder(&out)
	encoder.SetIndent("", "  ")
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(document); err != nil {
		return nil, err
	}
	if _, err := decodeRunConfig(out.Bytes()); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}
