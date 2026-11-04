# A fresh start for realization

Direct owner statements in the sessions "Markitect Project discussion" and "Markitect concept discussion", 4 November 2026, kept verbatim. They are the basis of [draft concept 4.0](../v4-concept.md).

## While test 1 ran

> Okay in my opinion we are going way to complex here again. The evaluation suite is too complex, the docker version is too complex with the git pushing and so on. Im currently considering redesigning the evaluation suite to something simple. My goal wouldnt be anymore to prove that its better or something like that. My goal now would be to make one clean and simple version work reliably with Sonnet. I mean we are evaluating different versions for more than a month now. Somehow always on the evaluation suite we run hundreds of evaluations with some checkers and so on and this takes a huge amount of time and the results seem to not help us really and it gets frustrating.
>
> Lets shrink the versions for apply down to the two simplest approaches. I definately want to try the native version where depending on the AI provider we just init a harness with skills and so on and rely on the agent to follow the process and hope the provider works on making the llm follow instructions and rules. If this version can work reliably i would be happy already.

## The order of work

> We will first cleanup the different versions and the eval suite we currently have. We just remove them for now so we have a fresh start. I hope our architecture is clean so that we can just delete the package for each version and create a new one and register it behind the interface. At least this is how i expect the code to look more or less.
>
> Then we will talk about how apply should work and create a concept for it and implement it. Then we will talk about how our evaluation should look like. We will take a much simpler approach here

## Leaving the old version in place for now

Asked whether the engine behind `apply` and `verify` should be removed, the owner answered "No, keep it for now" and then:

> Lets just leave it as is for now. Lets discuss first how we can do it

## No dependence on GitHub

> We dont want to rely on github

## The native version

> Okay so this would mean Markitect is just the specification and the tools on it combined with a harness and the tooling that builds on the spec. This sounds quite simple and usable. I think we should definately try this version. Also instead of land we should take a better name, maybe commit or accept or so.

## What the owner wants in the end

> Human uses a claude session and designs the spec or changes it.
> Once the spec is done we want to apply or maybe use different commands.
> Apply should make sure to use the tools of markitect. We always want that the model lets "compile" itself, that apply does not just change the model, that it uses the checklist and maybe the impact utilities and so on.
> Also we want to periodically run several of those things, for example bug hunt or cleanup or the command to check, that the model and the code are matching.

## The commands

> Submit is good.

> maybe we should add update. Also we can remove desktop for now. Check should also have a severity flag I propose.

> yes update the draft. Also im not sure about graph. The idea of lint and so on was to have the same analysis tools as we have in code. So for example coupling crap and so on that we know from clean architecture and whatever fits here. So the end goal was to have some analysis to see if the spec is good. Also when you say we saw 85% of code base linked together we maybe should think about wether the problem is that the model is bad and links to much together to a big ball of mud.

Recorder's context: the next statement answers the recorder's proposal that `check --severity warning` show stale registry entries and unused Definitions.

> no compile should fail if stale registry entries exist or definition nothing uses exist i would propose.
> And yes lets use analyze. We will then have to discuss it in detail later, for now do as you think makes sense.

## How `apply` runs

> Wait this goes into the wrong direction. I either edit the model or i run apply. While apply is running i dont want to ask questions in between or interrupt it. Also it should just take the model and apply it. I dont want it to be too interactive, because then i have to sit there and watch instead of modelling

Recorder's context: the next statement answers the recorder's proposal to start `apply` with `claude -p`.

> why would we use this here? I thought we go native? The process should be the following:
>
> * I model the spec with claude.
> * At some point im happy with how it looks.
> * I use the apply skill
> * Claude opens a branch and applies the changes maybe in its own session
>
> I dont want to model in a spec where claude is currently applying. This just makes things chaotic.

Asked whether `submit` should bring the owner's branch up by itself when the final check passes, the owner answered "Yes".

## Own worktree, and advice instead of following

In the session "Markitect concept discussion", answering a recommendation to run `apply` in a worktree of its own:

> Yes i think own worktree. I think all the discussions confused you too much. I want you to recommend how we should build it and advise me. Not just follow my words blindly. The whole idea about Markitect is to model the project and then have tooling that use the spec for different functions like making sure it compiles or looking at impact or generating a checklist. In the end we apply and want that the spec as is gets applied reliably. We dont want to babysit, review or slow down the process in any other way and concentrate on what agents cant do by themselves: design and model.

## One draft

In the session "Markitect Project discussion", asked whether the first draft and that recommendation should stay apart or become one concept, the owner answered "Merge into one". The recommendation's open points were not answered there, so they stand as questions in the draft.
