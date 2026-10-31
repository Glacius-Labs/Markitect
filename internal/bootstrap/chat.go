package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/adapters/desktop"
	"github.com/Glacius-Labs/Markitect/internal/agents"
)

// chatLabel marks a chat agent's container and volumes with its name.
const chatLabel = "markitect.chat"

// chatTurn bounds one turn of a chat.
const chatTurn = 30 * time.Minute

// desktopChats starts Markitect Desktop's chat agents with the engine,
// Manager image, provider, login, model and effort of apply and verify
// (agentConfig), read at every start, so a login refreshed on the host
// reaches the next agent.
type desktopChats struct{}

func (desktopChats) Start(ctx context.Context, s desktop.ChatStart) (desktop.ChatAgent, error) {
	cfg, engine, err := agentConfig(ctx)
	if err != nil {
		return nil, err
	}
	login, err := os.ReadFile(cfg.LoginFile)
	if err != nil {
		return nil, fmt.Errorf("the %s login: %w", cfg.Provider.Name(), err)
	}
	// A chat agent of an earlier app that ended without cleaning up holds
	// this conversation's container name; it goes first.
	if err := sweepChat(ctx, engine, s.Name); err != nil {
		return nil, fmt.Errorf("an earlier chat agent of this conversation: %w", err)
	}
	pending := make([]agents.ChatFile, len(s.Pending))
	for i, f := range s.Pending {
		pending[i] = agents.ChatFile{Path: f.Path, Data: f.Data, Removed: f.Removed}
	}
	chat, err := agents.StartChat(ctx, cfg.Runtime, cfg.Provider, agents.ChatSpec{
		Name: s.Name, Labels: map[string]string{chatLabel: s.Name}, Image: cfg.Image, Limits: cfg.Limits,
		Network: cfg.Network, Hosts: cfg.Hosts, DNS: cfg.DNS, Login: login, Model: cfg.Model, Effort: cfg.Effort,
		Skill: s.Skill, Earlier: s.Earlier, Files: s.Files, Read: s.Tree.Read, Pending: pending, Turn: chatTurn,
	})
	if err != nil {
		return nil, err
	}
	return desktopChat{chat}, nil
}

// sweepChat removes the containers and volumes of a chat agent by its name.
func sweepChat(ctx context.Context, engine interface {
	Containers(ctx context.Context, key, value string) ([]string, error)
	Volumes(ctx context.Context, key, value string) ([]string, error)
	RemoveContainer(ctx context.Context, name string) error
	RemoveVolume(ctx context.Context, name string) error
}, name string) error {
	containers, err := engine.Containers(ctx, chatLabel, name)
	if err != nil {
		return err
	}
	var errs []error
	for _, c := range containers {
		errs = append(errs, engine.RemoveContainer(ctx, c))
	}
	volumes, err := engine.Volumes(ctx, chatLabel, name)
	if err != nil {
		return errors.Join(append(errs, err)...)
	}
	for _, v := range volumes {
		errs = append(errs, engine.RemoveVolume(ctx, v))
	}
	return errors.Join(errs...)
}

// desktopChat is one chat agent as the desktop holds it.
type desktopChat struct{ c *agents.Chat }

func (d desktopChat) Send(ctx context.Context, message string, say func(string)) (desktop.ChatEnd, error) {
	out, err := d.c.Send(ctx, message)
	if err != nil {
		return desktop.ChatEnd{}, err
	}
	if out.Text != "" {
		say(out.Text)
	}
	detail := out.Detail
	if out.Class != "" {
		detail = fmt.Sprintf("%s: %s", out.Class, detail)
	}
	return desktop.ChatEnd{End: string(out.End), Detail: detail}, nil
}

func (d desktopChat) Changes(ctx context.Context) ([]desktop.ChatFile, []string, error) {
	model, outside, err := d.c.Changes(ctx)
	if err != nil {
		return nil, nil, err
	}
	files := make([]desktop.ChatFile, len(model))
	for i, f := range model {
		files[i] = desktop.ChatFile{Path: f.Path, Data: f.Data, Removed: f.Removed}
	}
	return files, outside, nil
}

func (d desktopChat) Settle(ctx context.Context, keep bool) error { return d.c.Settle(ctx, keep) }
func (d desktopChat) Close(ctx context.Context) error             { return d.c.Close(ctx) }
