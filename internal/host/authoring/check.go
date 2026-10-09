package authoring

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"
)

const CheckNamePattern = `^[A-Za-z0-9][A-Za-z0-9._-]*$`
const CheckExecutablePattern = `^[^/\\\s]+$`
const DefaultCheckTimeoutSeconds = 600
const MaxCheckTimeoutSeconds = 5400

var checkNamePattern = regexp.MustCompile(CheckNamePattern)
var checkExecutablePattern = regexp.MustCompile(CheckExecutablePattern)

func ValidateCheck(check Check) error {
	if check.TimeoutSeconds != nil && (*check.TimeoutSeconds < 1 || *check.TimeoutSeconds > MaxCheckTimeoutSeconds) {
		return fmt.Errorf("timeoutSeconds must be between 1 and %d", MaxCheckTimeoutSeconds)
	}
	if !checkNamePattern.MatchString(check.Name) {
		return fmt.Errorf("name must match %q", CheckNamePattern)
	}
	if len(check.Run) == 0 {
		return fmt.Errorf("run must contain at least one argument")
	}
	if strings.TrimSpace(check.Run[0]) == "" || !checkExecutablePattern.MatchString(check.Run[0]) {
		return fmt.Errorf("run[0] must be a bare PATH command name without whitespace or path separators")
	}
	for _, r := range check.Run[0] {
		if unicode.IsSpace(r) {
			return fmt.Errorf("run[0] must be a bare PATH command name without whitespace or path separators")
		}
	}
	for i, arg := range check.Run {
		if strings.ContainsRune(arg, '\x00') {
			return fmt.Errorf("run[%d] must not contain NUL", i)
		}
	}
	return nil
}
