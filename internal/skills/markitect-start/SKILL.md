---
name: markitect-start
description: Use right after markitect init, when .markitect/model/project.md is still the placeholder, or when the owner asks to start, set up or design a new project. It finds out with the owner what the project is and starts its spec. Not for a repository that already has code; use markitect-extract for that.
---

# Starting a project

Take the owner from an empty skeleton to a spec that is clear enough to build. They decide; you ask, propose and write. Read the `markitect-modeling` skill first: it holds the form of the spec and the rule that you invent nothing.

Ask one question at a time, in plain words, and say why you ask when it isn't obvious. Skip what they have already told you. When they are unsure, offer a suggestion and let them accept or change it. Stop asking when another answer would not change the spec: the spec starts small and grows as the owner extends it.

## What to find out

1. **What it is.** What does the project do, for whom, and what problem does it solve?
2. **What it leaves out.** What does it deliberately not do? This keeps the first version small.
3. **What it must do.** The few things without which a first version is useless. Note what each one takes in and gives back, and what happens when it goes wrong.
4. **How it is used.** A command line, a web API, a user interface, a library? The `cli` and `rest` Libraries have Kinds for commands and endpoints: add the Library to `.markitect/manifest.yaml` and run `markitect tidy`.
5. **What it is built with.** Language, platform, where data lives. If they have no preference, propose something plain that fits and ask them to confirm.
6. **What must always hold.** Rules that apply everywhere, such as how the code is laid out, that it works offline, or how errors reach the user. Ask how the work should be tested. If they want tests, state that as a Rule, with the command that runs them, because the reviewer checks what the spec says.

## Write the spec

- Replace the placeholder in `.markitect/model/project.md` with what the project is, for whom, and what it leaves out. Link the Definitions it is made of.
- Write what the project must do as Features, its interfaces with the Kinds of the Libraries, and what must always hold as Rules. Write only what the owner said or accepted.
- A small project lives entirely in the model folder. Propose Packages only when it has several real areas, and stop for the owner's answer before you write them.
- Run `markitect check` and `markitect vet`, and fix what they report. Until something is built, `check` says that the spec compiles and nothing is built yet, which is what you want here.

## Hand over

Tell the owner in a few lines what the spec now says and what you left open, and ask what is wrong or missing. Change it until they accept it. Then tell them that `/markitect-build` has a builder agent build it in this folder. Do not start it yourself: it stays their decision.
