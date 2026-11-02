package policies

import "fmt"

// Refusal is the code and the reason of a refused request. The codes shared by
// every module are the ones of the refusal-codes page; each area module adds
// its own.
type Refusal struct{ Code, Reason string }

// Refuse builds a refusal with a formatted reason.
func Refuse(code, format string, a ...any) *Refusal {
	return &Refusal{Code: code, Reason: fmt.Sprintf(format, a...)}
}
