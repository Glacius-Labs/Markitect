package protocol

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"unicode/utf8"
)

// request is a syntactically valid request envelope. The arguments are still
// raw JSON; the operation decides which are required.
type request struct {
	op   string
	args map[string]json.RawMessage
}

// parseRequest reads one request line. It returns the request id when the
// line is an object with a string id (nil otherwise), even if the line is an
// invalid request, so that the response can echo it.
func parseRequest(line []byte) (req request, id any, err error) {
	if !utf8.Valid(line) {
		return request{}, nil, errors.New("the line is not valid UTF-8")
	}
	fields, err := decodeObject(line)
	if err != nil {
		return request{}, nil, fmt.Errorf("the line is not a JSON object: %w", err)
	}
	if text, ok := decodeString(fields["id"]); ok {
		id = text
	}

	for name := range fields {
		if name != "id" && name != "op" && name != "args" {
			return request{}, id, fmt.Errorf("unknown request field %q", name)
		}
	}
	if text, ok := id.(string); !ok || text == "" {
		return request{}, id, errors.New(`"id" must be a nonempty string`)
	}
	op, ok := decodeString(fields["op"])
	if !ok {
		return request{}, id, errors.New(`"op" must be a string`)
	}
	rawArgs, present := fields["args"]
	if !present {
		return request{}, id, errors.New(`"args" is required`)
	}
	args, err := decodeObject(rawArgs)
	if err != nil {
		return request{}, id, errors.New(`"args" must be an object`)
	}
	return request{op: op, args: args}, id, nil
}

// decodeObject decodes a JSON object into its raw members. JSON null and
// every other kind of value are errors.
func decodeObject(raw []byte) (map[string]json.RawMessage, error) {
	if !bytes.HasPrefix(bytes.TrimSpace(raw), []byte("{")) {
		return nil, errors.New("not an object")
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		return nil, err
	}
	return fields, nil
}

// decodeString decodes a JSON string. It reports false for a missing value,
// null and every non-string value.
func decodeString(raw json.RawMessage) (string, bool) {
	if !bytes.HasPrefix(bytes.TrimSpace(raw), []byte(`"`)) {
		return "", false
	}
	var text string
	if err := json.Unmarshal(raw, &text); err != nil {
		return "", false
	}
	return text, true
}
