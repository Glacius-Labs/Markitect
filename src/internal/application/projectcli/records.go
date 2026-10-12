package projectcli

import "github.com/Glacius-Labs/Markitect/src/internal/application/projectapp"

func readRecord(root, rawPath string) ([]byte, error) {
	return projectOperations().ReadRecord(root, rawPath)
}
func recordPath(raw string) (string, error) { return projectapp.RecordPath(raw) }
