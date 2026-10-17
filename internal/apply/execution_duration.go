package apply

import (
	"math/big"
	"regexp"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/organization"
)

var executionDuration = regexp.MustCompile(`^\+?(?:([0-9]+(?:\.[0-9]*)?|\.[0-9]+)(ns|us|µs|μs|ms|s|m|h))+$`)
var executionDurationPart = regexp.MustCompile(`([0-9]+(?:\.[0-9]*)?|\.[0-9]+)(ns|us|µs|μs|ms|s|m|h)`)

func (d *executionDecoder) duration(field executionField) (organization.Setting[time.Duration], bool) {
	text, ok := d.text(field, false)
	if !ok {
		return organization.Setting[time.Duration]{}, false
	}
	if !executionDuration.MatchString(text.Value) {
		d.issue(model.Failed, "execution.duration", field, "Expected a positive explicit duration string.")
		return organization.Setting[time.Duration]{}, false
	}
	// Parse decimal quantities exactly. time.ParseDuration silently truncates
	// sub-nanosecond fractions, which would change the authored declaration.
	value := new(big.Rat)
	for _, part := range executionDurationPart.FindAllStringSubmatch(text.Value, -1) {
		quantity, _ := new(big.Rat).SetString(part[1])
		scale := time.Nanosecond
		switch part[2] {
		case "us", "µs", "μs":
			scale = time.Microsecond
		case "ms":
			scale = time.Millisecond
		case "s":
			scale = time.Second
		case "m":
			scale = time.Minute
		case "h":
			scale = time.Hour
		}
		value.Add(value, quantity.Mul(quantity, new(big.Rat).SetInt64(int64(scale))))
	}
	if value.Sign() <= 0 {
		d.issue(model.Failed, "execution.duration", field, "Declared durations must be positive.")
		return organization.Setting[time.Duration]{}, false
	}
	if !value.IsInt() || !value.Num().IsInt64() {
		d.issue(model.NotCompleted, "execution.duration_range", field, "The declared duration exceeds this decoder's exact nanosecond representation.")
		return organization.Setting[time.Duration]{}, false
	}
	return organization.Setting[time.Duration]{Value: time.Duration(value.Num().Int64()), Source: text.Source}, true
}
