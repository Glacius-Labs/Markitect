# Owner direction: agents collaborate through a Git remote

Date: 27 October 2026, about 14:00 UTC. Source: direct owner messages in the Claude Code session "Markitect evaluation and canonical model design", written in English. The coordinator read them in that session's transcript. The owner statements below are verbatim. The context between them is the recorder's summary and is labeled as such. The recorded meaning is the recorder's reading and adds no decision.

## 1. The idea

> I have some ideas to address the issues we have with Windows and running the containers and so on. Currently Grok Bot has a lot of popularity and Dots and Duet are also coming. So it seems the future is about long running AI Agents with their own environment. If we could make Markitect work like that it might be nice. Also Kubernetes or Docker Containers could work here as such simpler AI Agents. However, as far as i understand the main issues come from sharing files and such between all the containers. Why dont we let the Agents work like real engineers and just using git and let them pull, commit, push and so on instead of sharing volumes and translating the OS Issues back and forth?

Context (recorder's summary): the session answered that agents already hand work over through Git, each in its own container with its own repository, and that the Linux runner clones the owner's committed history from a read-only mount and returns the verified result as a branch that the owner reviews and merges.

## 2. A remote instead of mounts

> I mean why dont they use a remote git like GitHub. Before the run they create a branch from main and each one pulls from there and pushes there. They dont need to mount any git stuff

Context (recorder's summary): the session proposed GitHub only at the runner's edge. Under that proposal the runner would push one result branch, the owner would review it as a pull request and merge it, and the agents would get no push access.

## 3. No owner review

> Thats exactly how we dont want to do it. We dont want to review PRs and merge them. We just want to model and then have agents apply it. Thats why we want to spawn so many agents, so we can trust the result enough. Nowadays noone really looks at the PRs anyways. Each of the agents should work by its own just like regular programmers and their managers in a company

## 4. Answers to the open questions

Context (recorder's summary): at 14:16 UTC the owner answered two questions that the session had put to them, quoting each question before the answer, and added a third remark. The questions were whether a push token in every agent container is acceptable, and whether "main" always means main or a branch the owner names per run.

On the push token:

> It is, but we should also think about a remote less option, for example by having a local remote or something.

On the target branch:

> main should mean some branch you want to work on and can specify

The third remark:

> Also note, that we at some point agreed, that each agent becomes its individual harness, and that tooling could just be provided and configured for each agent, which they use so they dont use git themselves but just the tools that are allowed to them. Before we just jump into it we of course need to design this and discuss and see wether this direction would be better.

## Recorded meaning

Recorder's reading:

- **Agents collaborate through a Git remote, like engineers in a company.** Before a run, a run branch is created from the target branch. Each agent pulls from the remote, commits and pushes. Managers integrate their children's work the same way. No Git directory is mounted into a container.
- **No owner review of the result.** The owner models, and the agents apply the model. A verified result lands on the target branch without a pull request that the owner reviews or merges. Trust comes from the many agents working independently, not from the owner reading the diff.
- **Push credentials and a local remote.** A push token in every agent container is acceptable. A run should also work without a hosted remote, for example against a local remote.
- **The target branch.** "Main" stands for the branch the owner wants to work on, named per run.
- **Git or harness tools.** The owner recalls the agreement that every agent has its own harness, and asks whether agents should not use Git themselves but only the tools the host provides and configures for them. This direction is to be designed and discussed before anything is built.
- **Superseded and still open.** These statements reverse the earlier rule that Apply never commits, merges or pushes, and the runner's hand-back as a branch that the owner merges. They do not settle whether agents use Git or only harness tools, or what happens when the target branch moves during a run. Those mechanics are open design questions in the [delivery plan](../../implementation-plan.md#git-remote-collaboration-owner-direction-27-october).
