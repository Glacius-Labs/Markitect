package authoring

var usesKinds = map[string]map[string]bool{
	"Skill":    {"Workflow": true, "Text": true},
	"Workflow": {"Workflow": true, "Skill": true, "Agent": true, "Text": true},
	"Agent":    {"Skill": true, "Workflow": true, "Text": true},
	"Contract": {"Text": true},
}
