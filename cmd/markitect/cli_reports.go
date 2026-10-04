package main

import (
	"os"

	"github.com/Glacius-Labs/Markitect/internal/host"
	"github.com/Glacius-Labs/Markitect/internal/core"
)

type report struct {
	Tool          string              `yaml:"tool"`
	Version       string              `yaml:"version"`
	ToolDigest    string              `yaml:"toolDigest,omitempty"`
	Revision      string              `yaml:"revision,omitempty"`
	Provisional   bool                `yaml:"provisional"`
	Digest        string              `yaml:"digest,omitempty"`
	Status        string              `yaml:"status"`
	Coverage      string              `yaml:"coverage"`
	Inventory     []host.Entry         `yaml:"inventory,omitempty"`
	Diagnostics   []core.Diagnostic   `yaml:"diagnostics,omitempty"`
	PolicyResults []core.PolicyResult `yaml:"policyResults,omitempty"`
	Files         []string            `yaml:"files,omitempty"`
	Gates         []host.GateResult    `yaml:"gates,omitempty"`
}

type queryEnvelope struct {
	Version        string `yaml:"version"`
	ToolDigest     string `yaml:"toolDigest"`
	Revision       string `yaml:"revision,omitempty"`
	Provisional    bool   `yaml:"provisional"`
	SnapshotDigest string `yaml:"snapshotDigest"`
	Result         any    `yaml:"result"`
}

func currentToolDigest() (string, error) {
	executable, err := os.Executable()
	if err != nil {
		return "", err
	}
	executableBytes, err := os.ReadFile(executable)
	if err != nil {
		return "", err
	}
	return host.Hash(executableBytes), nil
}
