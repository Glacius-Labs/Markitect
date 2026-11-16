# Providers

Markitect supports Claude Code only. A provider needs subagents with their own agent files, and a hook before a tool call that names the agent; another provider comes once Claude's version is reliable, and the tool does not change for it.

No token is needed: sessions run on the user's own stored login in the unmodified program, and each uses the user's subscription quota. Markitect pins no version of Claude Code, so a change of its defaults can arrive unannounced. Markitect starts no session and touches no credentials, so each user simply runs Claude Code on their own login, with Markitect as a tool it calls.

There is no app. If one comes, it works over the same commands.
