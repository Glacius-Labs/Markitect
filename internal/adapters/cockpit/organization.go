package cockpit

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// OrganizationView is an observation, never an execution or authority grant.
// Declared observations retain absence; only resolved observations apply defaults.
type OrganizationView struct {
	Mode         string                    `json:"mode"`
	Head         string                    `json:"head"`
	ModelID      string                    `json:"modelID"`
	ModelDigest  string                    `json:"modelDigest"`
	Status       string                    `json:"status"`
	Error        string                    `json:"error,omitempty"`
	Managers     []OrganizationManager     `json:"managers"`
	Declarations *OrganizationDeclarations `json:"declarations,omitempty"`
	Strategy     string                    `json:"strategy,omitempty"`
	Checks       []OrganizationCheck       `json:"checks"`
	Ownership    bool                      `json:"ownership"`
}

type OrganizationManager struct {
	Path        string              `json:"path"`
	Label       string              `json:"label"`
	Parent      string              `json:"parent,omitempty"`
	Children    []string            `json:"children"`
	Depth       int                 `json:"depth"`
	Definitions []LinkView          `json:"definitions"`
	Areas       []string            `json:"areas"`
	Execute     OrganizationProfile `json:"execute"`
	Verify      OrganizationProfile `json:"verify"`
	Team        *OrganizationTeam   `json:"team,omitempty"`
}

type OrganizationSource struct {
	File   string `json:"file"`
	Digest string `json:"digest"`
	Field  string `json:"field"`
}

type OrganizationProfile struct {
	Name     string              `json:"name,omitempty"`
	Model    string              `json:"model,omitempty"`
	Identity string              `json:"identity,omitempty"`
	Source   *OrganizationSource `json:"source,omitempty"`
}

type OrganizationProfiles struct {
	Execute OrganizationProfile `json:"execute"`
	Verify  OrganizationProfile `json:"verify"`
}

type OrganizationDeclarations struct {
	Defaults OrganizationProfiles            `json:"defaults"`
	Packages map[string]OrganizationProfiles `json:"packages"`
	Team     *OrganizationTeam               `json:"team,omitempty"`
	Strategy *OrganizationSource             `json:"strategy,omitempty"`
}

type OrganizationTeam struct {
	MaxHelpers int                  `json:"maxHelpers"`
	Helpers    []OrganizationHelper `json:"helpers"`
	Source     *OrganizationSource  `json:"source,omitempty"`
}

type OrganizationHelper struct {
	Role    string              `json:"role"`
	Writes  bool                `json:"writes"`
	Profile OrganizationProfile `json:"profile"`
}

type OrganizationCheck struct {
	Name       string              `json:"name"`
	Profile    string              `json:"profile"`
	Directory  string              `json:"directory"`
	Args       []string            `json:"args"`
	RequiredBy string              `json:"requiredBy,omitempty"`
	Timeout    string              `json:"timeout"`
	Source     *OrganizationSource `json:"source,omitempty"`
}

// OrganizationResolver performs the same bounded capture as run confirmation.
// It starts no Run, Agent or check. The server starts it only on request.
type OrganizationResolver func(context.Context) (*OrganizationView, error)

type organizationCache struct {
	mu        sync.Mutex
	resolve   OrganizationResolver
	value     *OrganizationView
	err       string
	resolving bool
}

func (c *organizationCache) get(request bool) (*OrganizationView, string, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if request && c.resolve != nil && !c.resolving {
		c.value, c.err, c.resolving = nil, "", true
		go c.run()
	}
	return c.value, c.err, c.resolving
}

func (c *organizationCache) run() {
	var value *OrganizationView
	var err error
	defer func() {
		if recovered := recover(); recovered != nil {
			err = fmt.Errorf("organization resolution failed: %v", recovered)
		}
		c.mu.Lock()
		defer c.mu.Unlock()
		c.value, c.resolving = value, false
		if err != nil {
			c.err = err.Error()
		}
	}()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	value, err = c.resolve(ctx)
}

func (s *Server) handleOrganization(w http.ResponseWriter, r *http.Request) {
	project, observing := s.project.get(false)
	var declared *OrganizationView
	if project != nil {
		declared = project.Organization
	}
	resolved, problem, resolving := s.organization.get(r.URL.Query().Get("resolve") == "1")
	writeJSON(w, struct {
		Declared   *OrganizationView `json:"declared"`
		Resolved   *OrganizationView `json:"resolved"`
		Observing  bool              `json:"observing"`
		Resolvable bool              `json:"resolvable"`
		Resolving  bool              `json:"resolving"`
		Error      string            `json:"error,omitempty"`
	}{declared, resolved, observing, s.organization.resolve != nil, resolving, problem})
}
