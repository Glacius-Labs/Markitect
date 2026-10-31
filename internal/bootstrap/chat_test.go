package bootstrap

import (
	"context"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/adapters/desktop"
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

func TestAChatNeedsTheAgentsSettings(t *testing.T) {
	t.Setenv("MARKITECT_MANAGER_IMAGE", "")
	_, err := desktopChats{}.Start(context.Background(), desktop.ChatStart{Name: "markitect-chat-a"})
	if err == nil || !strings.Contains(err.Error(), "MARKITECT_MANAGER_IMAGE") {
		t.Fatalf("a chat without a Manager image: %v", err)
	}
}
