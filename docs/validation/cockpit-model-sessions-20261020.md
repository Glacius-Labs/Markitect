# Cockpit model sessions — 20 October 2026

**Scope:** the fifth slice of the [cockpit](../design/cockpit.md#working-through-the-cockpit), adopted by the owner as [DEC-065](../concepts/register.md#dec-065-work-through-the-cockpit-agent-proposed-model-changes-and-configuration): an Agent session that proposes model changes on the owner's prompts, implemented in [authoring](../../internal/authoring/README.md#model-sessions) and the [cockpit adapter](../../internal/adapters/cockpit/README.md#model-sessions). It reuses the [configuration proposals](cockpit-configuration-20261020.md) path. No provider turn and no container ran for this record.

## What was built

- `markitect cockpit --run-config PATH --runtime-profile PATH` enables model sessions. Bootstrap captures the committed source with its settings and the Runtime profile as an Apply does, composes the container engine, the Agents of the prepared profiles with their mapped credentials and the workspace helper from the run configuration, and keeps session evidence under the run configuration's state directory.
- A session runs Root's Execute profile. Each owner prompt is one Execute turn on the session's workspace, read-only except `.markitect/model` and `.markitect/registry`, bounded by the execution budget's turn timeout; the session takes at most the budget's Agent turns. Follow-up prompts resume the provider conversation and keep the workspace.
- The Agent has two tools: `check` answers with the changed files and the structural check of its workspace, `submit` makes a `model` proposal with its summary through the one proposal path. Changes outside the model and the registry are refused.
- `#/model` starts a session in two steps, shows its state, profile, limits, reported usage, tool uses, proposals and the Agent's text, takes follow-up prompts and closes the session; a model proposal's page shows the Agent's summary as the Agent's text.

## Demonstration

Only synthetic. In headless Chrome, the model page ran against a scripted session interface injected into the page (no Agent, no container, no provider): the two-step start with its cost note, the working state with polling, an answered turn with a check, a submitted proposal, usage and the Agent's text, a follow-up prompt, closing, and at 390 px in dark mode no horizontal scroll. No browser console errors were logged. Its usage figures were invented by the script and prove nothing about a provider.

## Checks

`go test` of `internal/authoring`, the cockpit, CLI, apply, bootstrap and tooling packages, `go vet`, `gofmt`, the architecture gate and `markitect check` passed locally on Windows. With scripted Agents and an in-memory workspace over the bootstrap fixture repository, the tests cover: the turn's profile, kind, read-only base with exactly the model and registry writable, timeout, home, workspace and transcript path; the check tool's changed files and passed check; a submitted proposal of kind `model` with the session, summary, passed check and only the model file changed; the resumed conversation on the next prompt; the Agent-turn limit; one open session at a time; a prompt refused while the session works; closing a working session, which stops the turn, closes the workspace and removes the home; changes outside the model, an empty summary and an unknown tool refused without a proposal; and the session record's kinds. The HTTP tests cover the cockpit's own JSON request for start, prompt and close, unknown fields, and the status of each refusal.

## Not established

A real session: an Agent of a Runtime profile in a container answering a prompt, its transcript, usage and proposal, and the owner's acceptance of a model proposal. That needs a provider turn, which the owner starts with a prompt. Owner use of the page is unassessed.
