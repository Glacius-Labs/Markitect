package codexappserver

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"os"
	"sort"

	"github.com/Glacius-Labs/Markitect/src/internal/host/agentexec"
)

func assetDigest(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Size() > agentexec.MaxRuntimeAssetBytes {
		return "", errors.New("runtime asset exceeds bound")
	}
	h := sha256.New()
	n, err := io.Copy(h, io.LimitReader(f, agentexec.MaxRuntimeAssetBytes+1))
	if err != nil || n > agentexec.MaxRuntimeAssetBytes {
		return "", errors.New("runtime asset digest failed")
	}
	return "sha256:" + hex.EncodeToString(h.Sum(nil)), nil
}
func bindReceipt(receipt *agentexec.Receipt, cfg agentexec.Config, env []string) error {
	var err error
	receipt.ExecutableDigest, err = assetDigest(cfg.Command)
	if err != nil {
		return err
	}
	command, _ := json.Marshal(struct {
		Command string
		Args    []string
	}{cfg.Command, []string{"app-server", "--listen", "stdio://"}})
	receipt.CommandDigest = digest(command)
	files := append([]agentexec.RuntimeFile(nil), cfg.RuntimeFiles...)
	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
	wire, _ := json.Marshal(files)
	receipt.RuntimeFilesDigest = digest(wire)
	env = append([]string(nil), env...)
	sort.Strings(env)
	wire, _ = json.Marshal(env)
	receipt.EnvironmentDigest = digest(wire)
	return nil
}
