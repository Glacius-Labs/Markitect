package bootstrap

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/adapters/desktop"
	"github.com/Glacius-Labs/Markitect/internal/adapters/repo"
	"github.com/Glacius-Labs/Markitect/internal/testkit"
)

// sweptEngine records what a sweep removes.
type sweptEngine struct {
	containers, volumes map[string][]string // by label value
	removed             []string
}

func (e *sweptEngine) Containers(_ context.Context, key, value string) ([]string, error) {
	if key != chatLabel {
		return nil, nil
	}
	return e.containers[value], nil
}

func (e *sweptEngine) Volumes(_ context.Context, key, value string) ([]string, error) {
	if key != chatLabel {
		return nil, nil
	}
	return e.volumes[value], nil
}

func (e *sweptEngine) RemoveContainer(_ context.Context, name string) error {
	e.removed = append(e.removed, "container "+name)
	return nil
}

func (e *sweptEngine) RemoveVolume(_ context.Context, name string) error {
	e.removed = append(e.removed, "volume "+name)
	return nil
}

func TestAChatStartSweepsWhatAnEarlierAgentOfItsConversationLeft(t *testing.T) {
	e := &sweptEngine{
		containers: map[string][]string{"markitect-chat-a": {"markitect-chat-a"}, "markitect-chat-b": {"markitect-chat-b"}},
		volumes:    map[string][]string{"markitect-chat-a": {"markitect-chat-a-home", "markitect-chat-a-work"}},
	}
	if err := sweepChat(context.Background(), e, "markitect-chat-a"); err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(e.removed, ","); got != "container markitect-chat-a,volume markitect-chat-a-home,volume markitect-chat-a-work" {
		t.Errorf("removed %s", got)
	}
}

// A chat's copy holds regular files only: a symbolic link could lead out of
// the repository.
func TestAChatCopiesOnlyRegularFiles(t *testing.T) {
	dir := testkit.TempDir(t)
	if err := os.WriteFile(filepath.Join(dir, "a.md"), []byte("a\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	read, err := regularFiles(repo.WorkingTree{Dir: dir, Ctx: context.Background()})
	if err != nil {
		t.Fatal(err)
	}
	if data, err := read("a.md"); err != nil || string(data) != "a\n" {
		t.Fatalf("a regular file: %q %v", data, err)
	}
	if _, err := regularFiles(memTree{}); err == nil {
		t.Error("a tree without a root was read")
	}
	outside := filepath.Join(testkit.TempDir(t), "secret")
	os.WriteFile(outside, []byte("secret\n"), 0o600)
	if err := os.Symlink(outside, filepath.Join(dir, "link.md")); err != nil {
		t.Logf("no symbolic link here (%v); the refusal is not checked", err)
		return
	}
	if data, err := read("link.md"); err == nil {
		t.Errorf("a symbolic link was followed: %q", data)
	}
}

// memTree is a repository state with no root.
type memTree struct{}

func (memTree) Files() ([]string, error)    { return nil, nil }
func (memTree) Read(string) ([]byte, error) { return nil, os.ErrNotExist }
func (memTree) GitIgnored(string) bool      { return false }

func TestAChatNeedsTheAgentsSettings(t *testing.T) {
	t.Setenv("MARKITECT_MANAGER_IMAGE", "")
	_, err := desktopChats{}.Start(context.Background(), desktop.ChatStart{Name: "markitect-chat-a"})
	if err == nil || !strings.Contains(err.Error(), "MARKITECT_MANAGER_IMAGE") {
		t.Fatalf("a chat without a Manager image: %v", err)
	}
}
