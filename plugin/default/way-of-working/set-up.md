# Set up the project

Type: [Activity](../core/activity.md)

A project begins to work this way. It is done once, by a person with the Assistant; afterwards the [daily run](day/daily-run.md) and the [start of work](day/start-of-work.md) carry the project.

## Participants
[Admin](team/admin.md): decides and answers.
[Assistant](team/assistant.md): leads through the steps and writes.

## Trigger
A person says they want to work this way, in a repository whose model does not hold the way of working yet. The skill for this activity is the one skill that does not lie in the project: it comes with Markitect's plugin for the provider, together with the default, and the person starts it there.

## Steps
1. Ask which provider the agents run on, and bring in the default from the plugin: the module `core`, this module with the skills built from it for that provider ([a skill per activity](team/a-skill-per-activity.md)), and the marker of the model. Nothing has to be built for the first day.
2. Enter the people: one [person](../core/person.md) per human in [the team](team/README.md), with the roles they fill and how they are reached when something is critical. A project with one person enters them as Admin.
3. Name the modules of the product: for each a README that says what it is for, the role responsible and where it is realized, linked from the README of the model, and a module of the same path in the plan ([names in the work](work/names.md)). For a project that starts empty, the person names them roughly and works them out in the first [planning](planning/planning.md) of the model; for a project that exists already, the [extraction](planning/extraction.md) writes them.
4. Add the module for the project's infrastructure ([a module for infrastructure](plan/infrastructure-modules.md)): the build, the tests, the repository's own files, how the application is started for trying, with which test data and how behaviour that depends on time is shown, how a feature is switched off and on, and how the project is released. In a project that starts empty these documents say what is wanted, and the first stories build it.
5. Write the ignore file: what is not the project's own, such as the history of the repository, build output and fetched dependencies.
6. Set the repository's settings and complete the list of what is critical for this project ([repository settings](deviations/repository-settings.md), [what is critical](deviations/critical.md)).
7. Change a level's AI model only where the project wants another ([levels](team/levels/README.md)).
8. Set up the routine that starts the daily run, and say under which identity it runs and writes to `main`, what it needs to do so, and by which means an agent sends a message to a person ([when shown](briefing/when-shown.md)). Name the tool that looks for secrets.
9. Enter the meetings, when the project has any ([meetings](briefing/meetings/README.md)), and the watches for its own sources ([add a watch](observing/add-a-watch.md)).
10. Run `markitect check` until it reports nothing. Write everything to `main`, and leave a [report](observing/report.md) in the context.
11. Start the daily run once. The watches on what happens on `main` record on their first run what they find and report nothing, so that what the setting up changed does not come back as deviations.
12. Tell the person how a day looks from now on: in the morning the Assistant says what happened and proposes where to begin; the daily run turns the circle without them; what needs them reaches them as a decision, and what is critical at once. For a project that starts empty, the first thing to begin with is the planning of the model.

## Result
The model holds the way of working, the people, the modules of the product and of its infrastructure; the skills lie where the agents find them; the repository is set, and the daily run has run once. The person can begin with the [start of work](day/start-of-work.md).

## Notes
Setting up changes the model without a planning; that is its way, once ([three ways to main](work/straight-to-main.md)). What the person does not know yet is left open and said so in the report, not guessed.
