package cockpit

import "net/http"

// ModelView is the checked canonical model the project observation prepared:
// every model scope with its Packages and Definitions, and the registry that
// says which files realize which Definitions. It is read, never written.
type ModelView struct {
	Models   []ModelScope    `json:"models"`
	Registry []RegistryEntry `json:"registry"`
	// RegistryStatus is the status of reading the registry; a registry that
	// did not pass may list fewer files than the project has.
	RegistryStatus string `json:"registryStatus"`
}

// ModelScope is one model of the selection: the local model or a dependency.
type ModelScope struct {
	ID          string           `json:"id"`
	Digest      string           `json:"digest,omitempty"`
	Local       bool             `json:"local"`
	Status      string           `json:"status"`
	Packages    []string         `json:"packages"`
	Definitions []DefinitionView `json:"definitions"`
}

// DefinitionView is one checked Definition. Links are the targets of the
// body's links. Segments are the body cut at the links the model check located:
// text, and each link with its target and the text it shows. A body whose
// links were not all located is one text segment.
type DefinitionView struct {
	Package  string        `json:"package"`
	ID       string        `json:"id"`
	Kind     string        `json:"kind"`
	Title    string        `json:"title,omitempty"`
	Body     string        `json:"body"`
	Source   string        `json:"source"`
	Links    []LinkView    `json:"links"`
	Segments []BodySegment `json:"segments"`
	Spec     []FieldView   `json:"spec"`
}

// LinkView is the target of one body link.
type LinkView struct {
	Package string `json:"package"`
	ID      string `json:"id"`
}

// BodySegment is a run of body text or one link with the text it shows.
type BodySegment struct {
	Text string    `json:"text"`
	Link *LinkView `json:"link,omitempty"`
}

// FieldView is one spec field with its value as text.
type FieldView struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// RegistryEntry is one registered artifact with the Definitions it realizes.
type RegistryEntry struct {
	Shard    string   `json:"shard"`
	Path     string   `json:"path"`
	Form     string   `json:"form"`
	Realizes []string `json:"realizes"`
	Why      string   `json:"why"`
}

// handleModel serves the model of the last project observation.
func (s *Server) handleModel(w http.ResponseWriter, r *http.Request) {
	p, observing := s.project.get(false)
	switch {
	case s.project.observe == nil:
		http.Error(w, "the cockpit was started without a repository", http.StatusNotImplemented)
	case p == nil || p.Model == nil:
		w.Header().Set("Retry-After", "2")
		if observing {
			http.Error(w, "the project is being observed", http.StatusServiceUnavailable)
			return
		}
		http.Error(w, "the project observation produced no checked model", http.StatusServiceUnavailable)
	default:
		writeJSON(w, struct {
			ObservedAt any        `json:"observedAt"`
			Head       string     `json:"head"`
			Model      *ModelView `json:"model"`
		}{p.ObservedAt, p.Head, p.Model})
	}
}
