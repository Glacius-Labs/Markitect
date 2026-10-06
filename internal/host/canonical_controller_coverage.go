package host

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/Glacius-Labs/Markitect/internal/host/agentexec"
	"github.com/Glacius-Labs/Markitect/internal/host/records"
)

const (
	canonicalVerifierObservationSubjectLimit = 4096
)

// canonicalVerifierObservationSubject is an explicit, kind-tagged subject for
// one item a Verifier must account for. Check identities expose their exact
// fields alongside the collision-resistant opaque Subject token.
type canonicalVerifierObservationSubject struct {
	Subject string `json:"subject"`
	Kind    string `json:"kind"`
	ID      string `json:"id"`
	Version string `json:"version,omitempty"`
	Digest  string `json:"digest,omitempty"`
}

// canonicalVerifierRequiredObservationSubjects builds a deterministic set of
// typed references for the exact scope, policies, artifacts, and fixed checks
// selected for one verifier invocation. It does not include a synthetic agent
// check; callers pass only fixed-check identities.
func canonicalVerifierRequiredObservationSubjects(scopeIDs, policyIDs, artifactPaths []string, checks []records.CheckIdentity) ([]canonicalVerifierObservationSubject, error) {
	result := make([]canonicalVerifierObservationSubject, 0, len(scopeIDs)+len(policyIDs)+len(artifactPaths)+len(checks))
	addStrings := func(kind, prefix string, values []string) error {
		seen := make(map[string]bool, len(values))
		for _, value := range values {
			if value == "" || strings.TrimSpace(value) != value || !utf8.ValidString(value) {
				return fmt.Errorf("%s observation identity must be nonempty, trimmed UTF-8", kind)
			}
			if seen[value] {
				return fmt.Errorf("duplicate %s observation identity %q", kind, value)
			}
			seen[value] = true
			subject := prefix + value
			if len(subject) > canonicalVerifierObservationSubjectLimit {
				return fmt.Errorf("%s observation identity exceeds the protocol subject bound", kind)
			}
			result = append(result, canonicalVerifierObservationSubject{Subject: subject, Kind: kind, ID: value})
		}
		return nil
	}
	if err := addStrings("scope", "scope:", scopeIDs); err != nil {
		return nil, err
	}
	if err := addStrings("policy", "policy:", policyIDs); err != nil {
		return nil, err
	}
	if err := addStrings("artifact", "artifact:", artifactPaths); err != nil {
		return nil, err
	}
	seenCheckIDs := make(map[string]bool, len(checks))
	for _, check := range checks {
		if check.ID == "" || strings.TrimSpace(check.ID) != check.ID || !utf8.ValidString(check.ID) ||
			check.Version == "" || strings.TrimSpace(check.Version) != check.Version || !utf8.ValidString(check.Version) ||
			!validSHA256(check.Digest) {
			return nil, fmt.Errorf("fixed-check observation identity %q has invalid id, version, or sha256 digest", check.ID)
		}
		if seenCheckIDs[check.ID] {
			return nil, fmt.Errorf("duplicate fixed-check observation identity %q", check.ID)
		}
		seenCheckIDs[check.ID] = true
		encoded, err := json.Marshal(struct {
			ID      string `json:"id"`
			Version string `json:"version"`
			Digest  string `json:"digest"`
		}{ID: check.ID, Version: check.Version, Digest: check.Digest})
		if err != nil {
			return nil, fmt.Errorf("encode fixed-check observation identity: %w", err)
		}
		subject := "check:" + base64.RawURLEncoding.EncodeToString(encoded)
		if len(subject) > canonicalVerifierObservationSubjectLimit {
			return nil, fmt.Errorf("fixed-check observation identity %q exceeds the protocol subject bound", check.ID)
		}
		result = append(result, canonicalVerifierObservationSubject{Subject: subject, Kind: "check", ID: check.ID, Version: check.Version, Digest: check.Digest})
	}
	if len(result) > agentexec.MaxVerifierObservations {
		return nil, fmt.Errorf("required verifier observation subjects exceed the %d-entry protocol bound", agentexec.MaxVerifierObservations)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Subject < result[j].Subject })
	for i := 1; i < len(result); i++ {
		if result[i-1].Subject == result[i].Subject {
			return nil, fmt.Errorf("duplicate typed observation subject %q", result[i].Subject)
		}
	}
	return result, nil
}

// validateCanonicalVerifierObservationCoverage validates observation identity
// independently of semantic truth. Failed, incomplete, and escalated responses
// may report a partial set, but a passing response must account for every exact
// required subject and each such observation must pass.
func validateCanonicalVerifierObservationCoverage(observations []agentexec.Observation, required []canonicalVerifierObservationSubject, overallOutcome string) error {
	if overallOutcome != agentexec.OutcomePassed && overallOutcome != agentexec.OutcomeFailed &&
		overallOutcome != agentexec.OutcomeIncomplete && overallOutcome != agentexec.OutcomeEscalated {
		return fmt.Errorf("invalid verifier outcome %q for observation coverage", overallOutcome)
	}
	wanted := make(map[string]canonicalVerifierObservationSubject, len(required))
	for _, subject := range required {
		if !validCanonicalVerifierObservationSubject(subject) {
			return errors.New("required verifier observation identity is blank, invalid, or oversized")
		}
		if _, duplicate := wanted[subject.Subject]; duplicate {
			return fmt.Errorf("duplicate required verifier observation identity %q", subject.Subject)
		}
		wanted[subject.Subject] = subject
	}
	if overallOutcome == agentexec.OutcomePassed && len(wanted) == 0 {
		return errors.New("passing verifier has no required observation subjects")
	}
	seen := make(map[string]bool, len(observations))
	for _, observation := range observations {
		if observation.Subject == "" || len(observation.Subject) > canonicalVerifierObservationSubjectLimit || !utf8.ValidString(observation.Subject) {
			return errors.New("verifier observation subject is blank, invalid, or oversized")
		}
		if _, expected := wanted[observation.Subject]; !expected {
			return fmt.Errorf("unexpected verifier observation identity %q", observation.Subject)
		}
		if seen[observation.Subject] {
			return fmt.Errorf("duplicate verifier observation identity %q", observation.Subject)
		}
		seen[observation.Subject] = true
		switch observation.Outcome {
		case agentexec.OutcomePassed, agentexec.OutcomeFailed, agentexec.OutcomeIncomplete, agentexec.OutcomeEscalated:
		default:
			return fmt.Errorf("invalid verifier observation outcome %q", observation.Outcome)
		}
		if strings.TrimSpace(observation.Detail) == "" || !utf8.ValidString(observation.Detail) {
			return fmt.Errorf("verifier observation %q requires a nonempty UTF-8 detail", observation.Subject)
		}
		if overallOutcome == agentexec.OutcomePassed && observation.Outcome != agentexec.OutcomePassed {
			return fmt.Errorf("passing verifier contains non-passing observation for %q", observation.Subject)
		}
	}
	if overallOutcome == agentexec.OutcomePassed {
		if len(seen) != len(wanted) {
			missing := make([]string, 0, len(wanted)-len(seen))
			for subject := range wanted {
				if !seen[subject] {
					missing = append(missing, subject)
				}
			}
			sort.Strings(missing)
			return fmt.Errorf("passing verifier omitted required observations: %s", strings.Join(missing, ", "))
		}
	}
	return nil
}

func validCanonicalVerifierObservationSubject(subject canonicalVerifierObservationSubject) bool {
	if subject.Subject == "" || len(subject.Subject) > canonicalVerifierObservationSubjectLimit || !utf8.ValidString(subject.Subject) ||
		subject.ID == "" || !utf8.ValidString(subject.ID) {
		return false
	}
	switch subject.Kind {
	case "scope":
		return subject.Subject == "scope:"+subject.ID && subject.Version == "" && subject.Digest == ""
	case "policy":
		return subject.Subject == "policy:"+subject.ID && subject.Version == "" && subject.Digest == ""
	case "artifact":
		return subject.Subject == "artifact:"+subject.ID && subject.Version == "" && subject.Digest == ""
	case "check":
		if subject.Version == "" || !validSHA256(subject.Digest) {
			return false
		}
		encoded, err := json.Marshal(struct {
			ID      string `json:"id"`
			Version string `json:"version"`
			Digest  string `json:"digest"`
		}{ID: subject.ID, Version: subject.Version, Digest: subject.Digest})
		return err == nil && subject.Subject == "check:"+base64.RawURLEncoding.EncodeToString(encoded)
	default:
		return false
	}
}
