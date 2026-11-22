# Observer

One observer of the way of working: it watches exactly one topic in one source and writes a [report](report.md) into the context on its schedule. The agent `observe` runs it, in a fresh session; each observer is a kind of agent of its own, with its folder `.markitect/context/observe-NAME/` for its reports and its checkpoint. Take it for a topic worth watching again and again. Do not take it for a single look, which is a story, nor for a second topic of an existing observer: one observer, one topic. An observer selects and states, with numbers and places; it never classifies, interprets, fixes or decides.

## Observes
The one topic, in one sentence. Example: `How the plan items move.`

## Source
Where the source lies and with which tool it is read, inside the repository or outside it; the access the person who starts the Housekeeper needs; and what marks how far a run has observed, a commit, a run or a file, where the next run begins. Example: `The plan in .markitect/plan/ and its Git history on main. Observed to: the commit.`

## Worth a statement
What the observer states and from which threshold, and what is noise and stays out. Thresholds are starting values, changed through findings. Example: `A story in progress without a commit for 3 days.`

## Never in the report
What must never stand in a report, and how an excerpt looks without it. Customer data and secrets never do; where an excerpt cannot be shown without them, the report holds only the reference. Example: `Names of borrowers; a loan is named by its number.`

## Schedule
How often it runs: `daily`, `weekly`, or every so many days. It is due when its last run lies further back than this. Example: `weekly`.

## Model
The model and the effort, as for an [agent](agent.md). Example: `Sonnet, effort high`.
