package protocol

import (
	"encoding/json"
	"unicode/utf8"

	"makerspace/policies"
)

// validate checks one raw argument against its spec (check 1).
func validate(spec policies.ArgSpec, raw json.RawMessage, present bool) (policies.Value, bool) {
	if !present {
		return policies.Value{}, false
	}
	if spec.Kind == policies.KindInteger {
		lit := string(raw)
		if len(lit) == 0 || lit[0] == '-' || len(lit) > 18 {
			return policies.Value{}, false
		}
		n, ok := policies.Digits(lit)
		if !ok || n < spec.Min || n > spec.Max {
			return policies.Value{}, false
		}
		return policies.Value{N: n}, true
	}
	if len(raw) == 0 || raw[0] != '"' {
		return policies.Value{}, false
	}
	var s string
	if err := json.Unmarshal(raw, &s); err != nil || !utf8.ValidString(s) {
		return policies.Value{}, false
	}
	switch spec.Kind {
	case policies.KindIdentifier:
		return policies.Value{S: s}, policies.IsIdentifier(s)
	case policies.KindText:
		return policies.Value{S: s}, policies.IsText(s)
	case policies.KindTimestamp:
		n, ok := policies.ParseTime(s)
		return policies.Value{S: s, N: n}, ok
	case policies.KindDate:
		n, ok := policies.ParseDay(s)
		return policies.Value{S: s, N: n}, ok
	case policies.KindMonth:
		n, ok := policies.ParseMonth(s)
		return policies.Value{S: s, N: n}, ok
	case policies.KindEnum:
		for _, e := range spec.Enum {
			if e == s {
				return policies.Value{S: s}, true
			}
		}
	}
	return policies.Value{}, false
}
