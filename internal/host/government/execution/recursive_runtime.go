package execution

import (
	"fmt"
	"sort"

	"github.com/Glacius-Labs/Markitect/internal/core"
	"github.com/Glacius-Labs/Markitect/internal/host/authoring"
	"github.com/Glacius-Labs/Markitect/internal/host/government"
)

const (
	maxRuntimeAreas      = 128
	maxRecursionDepth    = 32
	maxRecursionFanout   = 32
	maxRecursionCalls    = 4096
	maxRecursionParallel = 16
	maxRecursionRepairs  = 2
)

// RecursiveRuntime supplies distinct actor slots and technical obligations
// for every non-root node in the frozen plan. It is optional so a G2 runtime
// retains its existing meaning when recursion is omitted.
type RecursiveRuntime struct {
	Limits      government.DelegationLimits `json:"limits"`
	Parallelism int                         `json:"parallelism"`
	MaxRepairs  int                         `json:"maxRepairs"`
	Areas       []AreaRunner                `json:"areas"`
}

// AreaRunner freezes the local actors and checks for one non-root Area.
type AreaRunner struct {
	Area     core.DefinitionIdentity `json:"area"`
	Executor RunnerSpec              `json:"executor"`
	Verifier RunnerSpec              `json:"verifier"`
	Checks   []authoring.Check       `json:"checks"`
}

func validateRecursiveRuntime(recursive RecursiveRuntime, rootSlots map[string]struct{}) error {
	if recursive.Limits.MaxDepth < 1 || recursive.Limits.MaxDepth > maxRecursionDepth {
		return fmt.Errorf("limits.maxDepth must be between 1 and %d", maxRecursionDepth)
	}
	if recursive.Limits.MaxFanout < 1 || recursive.Limits.MaxFanout > maxRecursionFanout {
		return fmt.Errorf("limits.maxFanout must be between 1 and %d", maxRecursionFanout)
	}
	if recursive.Limits.MaxCalls < 1 || recursive.Limits.MaxCalls > maxRecursionCalls {
		return fmt.Errorf("limits.maxCalls must be between 1 and %d", maxRecursionCalls)
	}
	if recursive.Parallelism < 1 || recursive.Parallelism > maxRecursionParallel {
		return fmt.Errorf("parallelism must be between 1 and %d", maxRecursionParallel)
	}
	if recursive.MaxRepairs < 0 || recursive.MaxRepairs > maxRecursionRepairs {
		return fmt.Errorf("maxRepairs must be between 0 and %d", maxRecursionRepairs)
	}
	if len(recursive.Areas) == 0 || len(recursive.Areas) > maxRuntimeAreas {
		return fmt.Errorf("areas must contain 1 to %d non-root entries", maxRuntimeAreas)
	}
	areas := make(map[string]struct{}, len(recursive.Areas))
	for index, area := range recursive.Areas {
		if area.Area.APIVersion == "" || area.Area.Kind != "Area" || area.Area.Name == "" {
			return fmt.Errorf("areas[%d].area requires a complete Area identity", index)
		}
		key := area.Area.Key()
		if _, exists := areas[key]; exists {
			return fmt.Errorf("areas[%d] duplicates an Area identity", index)
		}
		areas[key] = struct{}{}
		for _, configured := range []struct {
			name   string
			runner RunnerSpec
		}{{"executor", area.Executor}, {"verifier", area.Verifier}} {
			if err := validateRunner(configured.runner); err != nil {
				return fmt.Errorf("areas[%d].%s: %w", index, configured.name, err)
			}
			if _, exists := rootSlots[configured.runner.SlotID]; exists {
				return fmt.Errorf("runner slot %q is assigned more than once", configured.runner.SlotID)
			}
			rootSlots[configured.runner.SlotID] = struct{}{}
		}
		if len(area.Checks) == 0 || len(area.Checks) > maxRuntimeChecks {
			return fmt.Errorf("areas[%d].checks must contain 1 to %d entries", index, maxRuntimeChecks)
		}
		checks := make(map[string]struct{}, len(area.Checks))
		for checkIndex, check := range area.Checks {
			if err := authoring.ValidateCheck(check); err != nil {
				return fmt.Errorf("areas[%d].checks[%d]: %w", index, checkIndex, err)
			}
			if _, exists := checks[check.Name]; exists {
				return fmt.Errorf("areas[%d].checks[%d] duplicates a check name", index, checkIndex)
			}
			checks[check.Name] = struct{}{}
		}
	}
	return nil
}

// areaRuntime selects the root contract or the exact configured non-root
// contract. The boolean is false if this runtime cannot run the requested node.
func areaRuntime(rt Runtime, node, root core.DefinitionIdentity) (RunnerSpec, RunnerSpec, []authoring.Check, bool) {
	if node.Key() == root.Key() {
		return rt.Executor, rt.Verifier, rt.Checks, true
	}
	if rt.Recursion == nil {
		return RunnerSpec{}, RunnerSpec{}, nil, false
	}
	for _, configured := range rt.Recursion.Areas {
		if configured.Area.Key() == node.Key() {
			return configured.Executor, configured.Verifier, configured.Checks, true
		}
	}
	return RunnerSpec{}, RunnerSpec{}, nil, false
}

// configuredRunners returns every configured actor in a stable order for
// orchestration-wide executable/runtime pinning.
func configuredRunners(rt Runtime) []RunnerSpec {
	runners := []RunnerSpec{rt.Executor, rt.Verifier}
	if rt.Recursion != nil {
		areas := append([]AreaRunner(nil), rt.Recursion.Areas...)
		sort.Slice(areas, func(i, j int) bool { return areas[i].Area.Key() < areas[j].Area.Key() })
		for _, area := range areas {
			runners = append(runners, area.Executor, area.Verifier)
		}
	}
	ressorts := append([]RessortRunner(nil), rt.Ressorts...)
	sort.Slice(ressorts, func(i, j int) bool { return ressorts[i].Ressort.Key() < ressorts[j].Ressort.Key() })
	for _, ressort := range ressorts {
		runners = append(runners, ressort.Runner)
	}
	return runners
}

type recursiveRuntimeFingerprint struct {
	Limits      government.DelegationLimits `json:"limits"`
	Parallelism int                         `json:"parallelism"`
	MaxRepairs  int                         `json:"maxRepairs"`
	Areas       []areaRunnerFingerprint     `json:"areas"`
}

type areaRunnerFingerprint struct {
	Area     core.DefinitionIdentity `json:"area"`
	Executor runnerFingerprint       `json:"executor"`
	Verifier runnerFingerprint       `json:"verifier"`
	Checks   []checkFingerprint      `json:"checks"`
}

func fingerprintRecursiveRuntime(recursive RecursiveRuntime) (recursiveRuntimeFingerprint, error) {
	value := recursiveRuntimeFingerprint{
		Limits: recursive.Limits, Parallelism: recursive.Parallelism, MaxRepairs: recursive.MaxRepairs,
	}
	areas := append([]AreaRunner(nil), recursive.Areas...)
	sort.Slice(areas, func(i, j int) bool { return areas[i].Area.Key() < areas[j].Area.Key() })
	for _, area := range areas {
		executor, err := fingerprintRunner(area.Executor)
		if err != nil {
			return recursiveRuntimeFingerprint{}, fmt.Errorf("Area %q executor fingerprint: %w", area.Area.Name, err)
		}
		verifier, err := fingerprintRunner(area.Verifier)
		if err != nil {
			return recursiveRuntimeFingerprint{}, fmt.Errorf("Area %q verifier fingerprint: %w", area.Area.Name, err)
		}
		fingerprint := areaRunnerFingerprint{Area: area.Area, Executor: executor, Verifier: verifier}
		checks := append([]authoring.Check(nil), area.Checks...)
		sort.Slice(checks, func(i, j int) bool { return checks[i].Name < checks[j].Name })
		for _, check := range checks {
			path, digest, err := fingerprintExecutable(check.Run[0])
			if err != nil {
				return recursiveRuntimeFingerprint{}, fmt.Errorf("Area %q check %q executable fingerprint: %w", area.Area.Name, check.Name, err)
			}
			timeout := authoring.DefaultCheckTimeoutSeconds
			if check.TimeoutSeconds != nil {
				timeout = *check.TimeoutSeconds
			}
			fingerprint.Checks = append(fingerprint.Checks, checkFingerprint{check.Name, append([]string(nil), check.Run...), timeout, path, digest})
		}
		value.Areas = append(value.Areas, fingerprint)
	}
	return value, nil
}
