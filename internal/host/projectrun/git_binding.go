package projectrun

import (
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/infrastructure/source"
)

func resolveGitHead(root string) (string, error) {
	identity, err := source.IdentifyGit(root)
	if err != nil {
		return "", err
	}
	output, err := source.GitOutput(root, "rev-parse", "--verify", "HEAD^{commit}")
	if err != nil {
		return "", fmt.Errorf("resolve repository HEAD: %w", err)
	}
	head := strings.TrimSpace(string(output))
	want := 40
	if identity.ObjectFormat == "sha256" {
		want = 64
	} else if identity.ObjectFormat != "sha1" {
		return "", fmt.Errorf("unsupported Git object format %q", identity.ObjectFormat)
	}
	if len(head) != want || head != strings.ToLower(head) {
		return "", fmt.Errorf("Git HEAD is not a full lowercase %d-character commit ID", want)
	}
	if _, err := hex.DecodeString(head); err != nil {
		return "", fmt.Errorf("Git HEAD is not hexadecimal: %w", err)
	}
	return head, nil
}

func resolveGitBranch(root string) (string, error) {
	output, err := source.GitOutput(root, "branch", "--show-current")
	if err != nil {
		return "", fmt.Errorf("resolve repository branch: %w", err)
	}
	branch := strings.TrimSpace(string(output))
	if strings.ContainsAny(branch, "\r\n\x00") {
		return "", fmt.Errorf("Git returned an invalid branch name")
	}
	return branch, nil
}
