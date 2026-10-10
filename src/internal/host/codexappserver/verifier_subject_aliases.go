package codexappserver

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"

	"github.com/Glacius-Labs/Markitect/src/internal/host/agentexec"
)

// nativeVerifierSubjectAliases binds verifier observation subjects only when
// the trusted request explicitly supplies requiredSubjects. Generic native
// verifier callers without that field retain the existing free-text contract.
func nativeVerifierSubjectAliases(inv agentexec.Invocation) (map[string]string, error) {
	if inv.Request.Role != agentexec.RoleVerifier {
		return nil, nil
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(inv.Request.Context, &fields); err != nil || fields == nil {
		return nil, errors.New("native verifier request context is invalid")
	}
	raw, supplied := fields["requiredSubjects"]
	if !supplied {
		return nil, nil
	}
	if string(raw) == "null" {
		return nil, errors.New("native verifier required subjects must be an array")
	}
	var subjects []string
	if err := json.Unmarshal(raw, &subjects); err != nil || subjects == nil {
		return nil, errors.New("native verifier required subjects must be an array")
	}
	if len(subjects) > nativeResponseArrayMaxItems {
		return nil, fmt.Errorf("native verifier required subjects exceed the supported %d values", nativeResponseArrayMaxItems)
	}
	set := make(map[string]struct{}, len(subjects))
	for _, subject := range subjects {
		if subject == "" {
			return nil, errors.New("native verifier required subject is empty")
		}
		if _, exists := set[subject]; exists {
			return nil, errors.New("native verifier required subjects contain a duplicate")
		}
		set[subject] = struct{}{}
	}
	canonical := append([]string(nil), subjects...)
	sort.Strings(canonical)
	for generation := 0; generation <= len(canonical); generation++ {
		prefix := fmt.Sprintf("verifier-subject-%06d-", generation)
		aliases := make(map[string]string, len(canonical))
		collision := false
		for index, subject := range canonical {
			alias := fmt.Sprintf("%s%06d", prefix, index)
			if _, exists := set[alias]; exists {
				collision = true
				break
			}
			aliases[alias] = subject
		}
		if !collision {
			return aliases, nil
		}
	}
	return nil, errors.New("could not construct a collision-free verifier subject alias namespace")
}

func decodeNativeVerifierObservationSubjects(raw json.RawMessage, aliases map[string]string) (json.RawMessage, error) {
	if aliases == nil {
		return raw, nil
	}
	var observations []json.RawMessage
	if err := json.Unmarshal(raw, &observations); err != nil || observations == nil {
		return nil, errors.New("native verifier observations must be an array")
	}
	if len(observations) > nativeResponseArrayMaxItems {
		return nil, fmt.Errorf("native verifier observations exceed the supported %d values", nativeResponseArrayMaxItems)
	}
	seenAliases := map[string]bool{}
	seenSubjects := map[string]bool{}
	for index, rawObservation := range observations {
		observation, err := decodeNativeSemanticObject(rawObservation, map[string]bool{"subject": true, "outcome": true, "detail": true})
		if err != nil {
			return nil, errors.New("native verifier observation must be a closed JSON object")
		}
		for _, key := range []string{"subject", "outcome", "detail"} {
			if _, ok := observation[key]; !ok {
				return nil, errors.New("native verifier observation omitted a required field")
			}
		}
		var alias string
		if err := json.Unmarshal(observation["subject"], &alias); err != nil || alias == "" {
			return nil, errors.New("native verifier observation subject alias is invalid")
		}
		if seenAliases[alias] {
			return nil, errors.New("native verifier observations contain a duplicate subject alias")
		}
		seenAliases[alias] = true
		canonical, ok := aliases[alias]
		if !ok {
			return nil, errors.New("native verifier observations contain an unknown subject alias")
		}
		if seenSubjects[canonical] {
			return nil, errors.New("native verifier observations duplicate a canonical subject")
		}
		seenSubjects[canonical] = true
		observation["subject"], _ = json.Marshal(canonical)
		observations[index], err = json.Marshal(observation)
		if err != nil {
			return nil, errors.New("native verifier observation could not be encoded")
		}
	}
	wire, err := json.Marshal(observations)
	if err != nil {
		return nil, errors.New("native verifier observations could not be encoded")
	}
	return wire, nil
}
