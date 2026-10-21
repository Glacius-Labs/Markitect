package bootstrap

import (
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/adapters/cockpit"
	"github.com/Glacius-Labs/Markitect/internal/apply"
	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/organization"
	"github.com/Glacius-Labs/Markitect/internal/realization"
)

func organizationSource(s organization.SettingSource) *cockpit.OrganizationSource {
	if s.File == "" {
		return nil
	}
	return &cockpit.OrganizationSource{File: s.File, Digest: s.ContentID.String(), Field: s.Field}
}

func declaredProfile(s *organization.Setting[string]) cockpit.OrganizationProfile {
	if s == nil {
		return cockpit.OrganizationProfile{}
	}
	return cockpit.OrganizationProfile{Name: s.Value, Source: organizationSource(s.Source)}
}

// organizationStructure delegates hierarchy and area exclusions to their
// product owners. In particular it does not reconstruct either from paths.
func organizationStructure(checked model.Result, layout realization.Layout) *cockpit.OrganizationView {
	v := &cockpit.OrganizationView{Managers: []cockpit.OrganizationManager{}, Checks: []cockpit.OrganizationCheck{}, Ownership: layout.Declared(), Status: string(checked.Status)}
	topology, err := organization.BuildTopology(checked)
	if err != nil {
		v.Error = err.Error()
		return v
	}
	v.ModelID, v.ModelDigest = topology.ModelID(), topology.ModelDigest()
	depths := map[string]int{}
	for _, m := range topology.Managers() {
		label := "Root"
		owner := realization.RootOwner()
		if m.Package != organization.Root {
			label = m.Package[strings.LastIndex(m.Package, "/")+1:]
			owner, _ = realization.PackageOwner(m.Package) // Validated by BuildTopology.
			depths[m.Package] = depths[m.Parent] + 1
		}
		manager := cockpit.OrganizationManager{Path: m.Package, Label: label, Parent: m.Parent, Children: append([]string{}, m.Children...), Depth: depths[m.Package], Definitions: []cockpit.LinkView{}, Areas: append([]string{}, layout.Describe(owner)...)}
		for _, d := range m.Definitions {
			manager.Definitions = append(manager.Definitions, cockpit.LinkView{Package: d.Package, ID: d.ID})
		}
		v.Managers = append(v.Managers, manager)
	}
	return v
}

func organizationChecks(doc apply.AuthoredExecution) []cockpit.OrganizationCheck {
	out := []cockpit.OrganizationCheck{}
	if doc.Checks == nil {
		return out
	}
	for _, c := range doc.Checks.Value {
		check := cockpit.OrganizationCheck{Name: c.ID.Value, Profile: c.Profile.Value, Directory: c.Directory.Value.String(), Args: append([]string{}, c.Args.Value...), Timeout: durationText(c.Timeout.Value), Source: organizationSource(c.ID.Source)}
		// This source binds the whole declaration document, not just its ID.
		if check.Source != nil {
			check.Source.Field = ""
		}
		if c.RequiredBy != nil {
			check.RequiredBy = c.RequiredBy.Value.Package + "/" + c.RequiredBy.Value.ID
		}
		out = append(out, check)
	}
	return out
}

func declaredOrganization(a apply.PreviewPreparationAttempt) *cockpit.OrganizationView {
	layout, _ := a.Files().Layout()
	v := organizationStructure(a.Model().Checked(), layout)
	v.Mode, v.Head = "declared", a.Model().Inputs().Head
	if a.Files().Status() != model.Passed {
		v.Status = string(a.Files().Status())
		v.Error = "File check did not pass; validated ownership areas are unavailable. This does not mean ownership is undeclared."
	}
	doc, ok := a.Execution().Document()
	if !ok {
		v.Status = string(a.Execution().Status())
		if v.Error == "" {
			v.Error = "The execution declarations did not pass preparation; no settings are shown."
		}
		return v
	}
	v.Strategy, v.Checks = doc.Strategy.Value, organizationChecks(doc)
	d := &cockpit.OrganizationDeclarations{Defaults: cockpit.OrganizationProfiles{Execute: declaredProfile(doc.Defaults.Execute.Profile), Verify: declaredProfile(doc.Defaults.Verify.Profile)}, Packages: map[string]cockpit.OrganizationProfiles{}, Strategy: organizationSource(doc.Strategy.Source)}
	if doc.Packages != nil {
		for _, p := range doc.Packages.Value {
			d.Packages[p.Package] = cockpit.OrganizationProfiles{Execute: declaredProfile(p.Execute.Profile), Verify: declaredProfile(p.Verify.Profile)}
		}
	}
	if doc.Team != nil {
		d.Team = &cockpit.OrganizationTeam{MaxHelpers: doc.Team.MaxHelpers.Value, Source: organizationSource(doc.Team.MaxHelpers.Source), Helpers: []cockpit.OrganizationHelper{}}
		for _, h := range doc.Team.Helpers {
			d.Team.Helpers = append(d.Team.Helpers, cockpit.OrganizationHelper{Role: h.Role.Value, Writes: h.Writes.Value, Profile: declaredProfile(h.Profile)})
		}
	}
	v.Declarations = d
	for i := range v.Managers {
		m := &v.Managers[i]
		m.Execute, m.Verify, m.Team = d.Defaults.Execute, d.Defaults.Verify, d.Team
		if p, ok := d.Packages[m.Path]; ok {
			if p.Execute.Source != nil {
				m.Execute = p.Execute
			}
			if p.Verify.Source != nil {
				m.Verify = p.Verify
			}
		}
	}
	return v
}

func resolvedOrganization(a apply.CapturedSettingsAttempt) (*cockpit.OrganizationView, error) {
	layout, _ := a.Structural().Files().Layout()
	v := organizationStructure(a.Structural().Model().Checked(), layout)
	v.Mode, v.Head = "resolved", a.Structural().Model().Inputs().Head
	strategy, _ := a.Settings().Strategy()
	v.Strategy = string(strategy)
	doc, _ := a.Structural().Execution().Document()
	v.Checks = organizationChecks(doc)
	settings, _ := a.Settings().Settings()
	profile := func(r organization.RoleSettings) (cockpit.OrganizationProfile, error) {
		name := r.Profile.Value.Name
		providerModel, err := executeProfileModel(a.Settings().ProfileObservations(), name)
		return cockpit.OrganizationProfile{Name: name, Model: providerModel, Identity: r.Profile.Value.Identity, Source: organizationSource(r.Profile.Source)}, err
	}
	for i := range v.Managers {
		m := &v.Managers[i]
		s, _ := settings.Manager(m.Path)
		var err error
		if m.Execute, err = profile(s.Execute); err != nil {
			return nil, err
		}
		if m.Verify, err = profile(s.Verify); err != nil {
			return nil, err
		}
		m.Team = &cockpit.OrganizationTeam{MaxHelpers: s.MaxHelpers, Helpers: []cockpit.OrganizationHelper{}}
		for _, h := range s.Helpers {
			p, err := profile(h.RoleSettings)
			if err != nil {
				return nil, err
			}
			m.Team.Helpers = append(m.Team.Helpers, cockpit.OrganizationHelper{Role: h.Role, Writes: h.Writes, Profile: p})
		}
	}
	return v, nil
}
