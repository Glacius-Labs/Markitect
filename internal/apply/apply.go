// Package apply owns public Apply availability during the S2 transition.
package apply

import "errors"

var ErrUnavailable = errors.New("Apply unavailable: the standalone V1 Run is implemented; workflow and publication composition await S3")
