package run

import (
	"encoding/json"
	"fmt"
	"path"
	"regexp"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/project"
)

// The agent types of Markitect's four roles (DEC-100). The harness defines them.
const (
	BuilderAgent   = "markitect-builder"
	ReviewerAgent  = "markitect-reviewer"
	ArchitectAgent = "markitect-architect"
	AuditorAgent   = "markitect-auditor"
)

// SettingsFile is the project's Claude Code settings, and HookCommand the hook in
// them that runs Guard. The agents are subagents of the owner's own session, so
// the hook has to be in the project's settings.
const (
	SettingsFile = ".claude/settings.json"
	HookCommand  = "markitect hook pretool"
)

// MarkRule is the permission rule that starts the hook before a shell command
// that records a verdict.
const MarkRule = "Bash(markitect mark*)"

// EditTools are the tools that write files, and AgentTools the tools that start
// an agent, as the matchers of the hooks that hold the roles' rights. Claude Code
// named its agent tool Task before it named it Agent.
const (
	EditTools  = "Edit|Write|MultiEdit|NotebookEdit"
	AgentTools = "Agent|Task"
)

// rights are what a subagent of a role may do. A file is code when it lies
// outside Markitect's folder, a registry page, the record, or else the spec.
// Nobody writes the record by an edit: markitect mark writes it.
type rights struct {
	code, spec, registry, mark bool
	// starts are the agents it may start; any when all.
	starts []string
	all    bool
}

// roles are the four roles in a 2×2 of rights (DEC-100): the builder writes code,
// the architect the spec, the reviewer judges the code and alone records, and the
// auditor judges the spec. Any other subagent has the builder's rights to write,
// and starts any agent but the two that write, which only the owner's session
// starts. The owner's own session is no subagent and stays free.
var roles = map[string]rights{
	BuilderAgent:   {code: true, registry: true, starts: []string{ReviewerAgent}},
	ReviewerAgent:  {mark: true},
	ArchitectAgent: {spec: true, registry: true},
	AuditorAgent:   {},
}

var otherSubagent = rights{code: true, registry: true, all: true}

// rightsOf returns the rights of a subagent's role, whatever the case of its name.
func rightsOf(agent string) rights {
	if r, ok := roles[strings.ToLower(agent)]; ok {
		return r
	}
	return otherSubagent
}

// may says whether a role may start an agent of a type.
func (r rights) may(start string) bool {
	start = strings.ToLower(start)
	if r.all {
		return start != BuilderAgent && start != ArchitectAgent
	}
	for _, s := range r.starts {
		if s == start {
			return true
		}
	}
	return false
}

// HasGuard says whether settings run HookCommand before a shell command that
// records a verdict, before every tool that writes a file and before every tool
// that starts an agent; a mention elsewhere does not count. A handler without a
// rule runs before every call of its tools, and so covers them.
func HasGuard(settings []byte) bool {
	var s struct {
		Hooks struct {
			PreToolUse []struct {
				Matcher string `json:"matcher"`
				Hooks   []struct {
					Type    string `json:"type"`
					If      string `json:"if"`
					Command string `json:"command"`
				} `json:"hooks"`
			} `json:"PreToolUse"`
		} `json:"hooks"`
	}
	if json.Unmarshal(settings, &s) != nil {
		return false
	}
	mark, covered := false, map[string]bool{}
	for _, m := range s.Hooks.PreToolUse {
		tools := map[string]bool{}
		all := m.Matcher == "" || m.Matcher == "*"
		for _, t := range strings.Split(m.Matcher, "|") {
			tools[strings.TrimSpace(t)] = true
		}
		for _, h := range m.Hooks {
			if h.Type != "command" || strings.TrimSpace(h.Command) != HookCommand {
				continue
			}
			if (all || tools["Bash"]) && (h.If == "" || h.If == MarkRule) {
				mark = true
			}
			if h.If != "" {
				continue
			}
			for _, t := range guardedTools() {
				if all || tools[t] {
					covered[t] = true
				}
			}
		}
	}
	return mark && len(covered) == len(guardedTools())
}

// guardedTools are the tools the hook runs before on every call.
func guardedTools() []string {
	return append(strings.Split(EditTools, "|"), strings.Split(AgentTools, "|")...)
}

// markCommand finds a call of markitect mark in a shell command.
var markCommand = regexp.MustCompile(`(?i)(^|[\s;&|("'` + "`" + `])markitect(\.exe)?\s+mark(\s|$)`)

// Guard is the project's hook before a tool call. It holds the rights of the
// four roles (DEC-100), and guards the record and the spec, not the process:
//
//   - only the reviewer runs markitect mark, so the agent that built does not
//     verify its own work;
//   - a subagent writes only what its role may: the builder and any other
//     subagent code and the registry pages, the architect the spec and the
//     registry pages, the reviewer and the auditor nothing, and nobody the
//     record, which mark writes, or the files that steer the agents;
//   - the builder starts only the reviewer, the other roles start no agent, and
//     no subagent starts the builder or the architect.
//
// The owner's own session is no subagent and stays free, but for mark. Its input
// is what the provider tells a hook, which names the agent a subagent's call
// comes from. It returns what the hook prints, empty to let the call run. It
// reads the call as text, so it guards against mistakes, not against an agent
// that hides a command.
func Guard(input []byte) []byte {
	var in struct {
		AgentType string         `json:"agent_type"`
		AgentID   string         `json:"agent_id"`
		ToolName  string         `json:"tool_name"`
		ToolInput map[string]any `json:"tool_input"`
	}
	if json.Unmarshal(input, &in) != nil {
		return nil
	}
	if command, ok := in.ToolInput["command"].(string); ok && markCommand.MatchString(command) && in.AgentType != ReviewerAgent {
		return deny(fmt.Sprintf("Only the %s agent records verdicts, so the agent that built does not verify its own work. Start it, and it verifies and records.", ReviewerAgent))
	}
	if in.AgentID == "" {
		return nil
	}
	r := rightsOf(in.AgentType)
	if isAgentTool(in.ToolName) {
		start, _ := in.ToolInput["subagent_type"].(string)
		if !r.may(start) {
			return deny(startRefusal(in.AgentType, start))
		}
		return nil
	}
	for _, key := range []string{"file_path", "notebook_path"} {
		file, _ := in.ToolInput[key].(string)
		if file == "" {
			continue
		}
		if why := writeRefusal(in.AgentType, r, file); why != "" {
			return deny(why)
		}
	}
	return nil
}

// isAgentTool says whether a tool starts an agent.
func isAgentTool(tool string) bool {
	for _, t := range strings.Split(AgentTools, "|") {
		if tool == t {
			return true
		}
	}
	return false
}

// startRefusal says why a subagent may not start an agent.
func startRefusal(agent, start string) string {
	if start == "" {
		start = "a general-purpose agent"
	}
	switch strings.ToLower(agent) {
	case BuilderAgent:
		return fmt.Sprintf("The builder starts only the %s agent, which verifies and records; it may not start %s.", ReviewerAgent, start)
	case ReviewerAgent, ArchitectAgent, AuditorAgent:
		return fmt.Sprintf("The %s agent starts no agent; do the work yourself, or say in your report what is needed.", agent)
	}
	return fmt.Sprintf("Only the owner's session starts the %s agent.", start)
}

// writeRefusal says why a subagent may not write a file, or "" when it may.
func writeRefusal(agent string, r rights, file string) string {
	switch kindOf(file) {
	case recordFile:
		return fmt.Sprintf("%s is written only by markitect mark, which the %s agent runs.", file, ReviewerAgent)
	case harnessFile:
		return fmt.Sprintf("%s steers the agents: Markitect's skills, agents and settings and the project's instruction files are the owner's, and no agent changes them. Say what should change.", file)
	case registryPage:
		if r.registry {
			return ""
		}
	case specFile:
		if r.spec {
			return ""
		}
		if r.code {
			return fmt.Sprintf("%s is the owner's: an agent that builds changes nothing under %s except the registry pages. If the spec is wrong, say so and leave it as it is.", file, project.Dir)
		}
	case codeFile:
		if r.code {
			return ""
		}
		if r.spec {
			return fmt.Sprintf("%s is code, which the builder writes; the %s agent changes only what lies under %s.", file, ArchitectAgent, project.Dir)
		}
	}
	return fmt.Sprintf("The %s agent changes no file; it reports what it found.", agent)
}

// The kinds of file the rights distinguish.
const (
	codeFile = iota
	specFile
	registryPage
	recordFile
	harnessFile
)

// kindOf says what kind of file a path names, whatever folder it starts from:
// one in Markitect's folder of a project is a registry page, the record or the
// spec; one that steers the agents is the harness; any other is code.
func kindOf(file string) int {
	segments := strings.Split(path.Clean(strings.ReplaceAll(file, `\`, "/")), "/")
	for i, s := range segments {
		if !strings.EqualFold(s, project.Dir) {
			continue
		}
		lower := strings.ToLower(strings.Join(segments[i:], "/"))
		registry := strings.ToLower(project.RegistryDir) + "/"
		switch {
		case lower == strings.ToLower(project.ReviewFile):
			return recordFile
		case lower == strings.ToLower(project.RootPage), strings.HasPrefix(lower, registry) && strings.HasSuffix(lower, ".md"):
			return registryPage
		}
		return specFile
	}
	if steersAgents(segments) {
		return harnessFile
	}
	return codeFile
}

// steersAgents says whether a path names a file that steers the agents (DEC-100):
// Claude Code's project settings, which hold the hook, Markitect's skills and
// agents, and the instruction files agents read by themselves. A builder whose
// work comes back by itself must not change them, since nobody reads that diff.
func steersAgents(segments []string) bool {
	n := len(segments)
	if base := strings.ToLower(segments[n-1]); base == "agents.md" || base == "claude.md" {
		return true
	}
	for i := 0; i+1 < n; i++ {
		dir, next := strings.ToLower(segments[i]), strings.ToLower(segments[i+1])
		switch {
		case dir == ".claude" && i+2 == n && (next == "settings.json" || next == "settings.local.json"):
			return true
		case (dir == ".claude" || dir == ".agents") && (next == "agents" || next == "skills") && i+2 < n &&
			strings.HasPrefix(strings.ToLower(segments[i+2]), "markitect-"):
			return true
		}
	}
	return false
}

// deny is the hook's answer that refuses the call, and why.
func deny(why string) []byte {
	out, err := json.Marshal(map[string]any{"hookSpecificOutput": map[string]string{
		"hookEventName":            "PreToolUse",
		"permissionDecision":       "deny",
		"permissionDecisionReason": why,
	}})
	if err != nil {
		return nil
	}
	return out
}
