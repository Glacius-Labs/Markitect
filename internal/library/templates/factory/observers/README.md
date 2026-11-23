# Observers

The observers of the way of working, one [observer](../../../packages/factory/observer.md) per document. Each watches exactly one topic in one source and writes a report into the context on its schedule; the agent [observe](../agents/observe.md) runs it, and the [Housekeeper](../agents/housekeeper.md) starts those that are due. The observers here watch sources every project with this way of working has:

- [dropped-material](dropped-material.md): what people and agents dropped into the context.
- [plan-items](plan-items.md): how the plan moves.
- [deviations](deviations.md): what happened on `main` against the usual way of the roles.
- [hotspots](hotspots.md): where development jams in the structure.
- [duplication](duplication.md): what is built more than once.
- [status](status.md): what moved between the spec and the files, new since its last run.

A project adds the observers for its own sources with [create-observer](../agents/create-observer.md), each as a document here, linked from this list: for example a regular run of the full test suite on `main`, where the project has one ([tests](../rules/building/tests.md)), or later logs, alerts, metrics, dependencies or feedback from users, once they exist and can be read without customer data.
