package agentexec

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"
	"unicode/utf8"
)

const (
	maxJSONBytes     = 32 << 20
	maxContextBytes  = 8 << 20
	maxArtifactBytes = 8 << 20
	maxArtifactCount = 128
	maxScopeCount    = 128
	maxFieldBytes    = 4096
)

func strictDecode(data []byte, dst any) error {
	if len(data) == 0 || len(data) > maxJSONBytes {
		return errors.New("JSON size is outside the supported bound")
	}
	if err := rejectDuplicateKeys(data); err != nil {
		return err
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	decoder.UseNumber()
	if err := decoder.Decode(dst); err != nil {
		return fmt.Errorf("invalid JSON protocol value: %w", err)
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return errors.New("JSON protocol value contains trailing data")
	}
	return nil
}

func rejectDuplicateKeys(data []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	if err := scanJSONValue(decoder); err != nil {
		return err
	}
	if _, err := decoder.Token(); err != io.EOF {
		if err == nil {
			return errors.New("JSON contains trailing data")
		}
		return fmt.Errorf("invalid JSON: %w", err)
	}
	return nil
}

func scanJSONValue(decoder *json.Decoder) error {
	token, err := decoder.Token()
	if err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}
	delim, isDelim := token.(json.Delim)
	if !isDelim {
		return nil
	}
	switch delim {
	case '{':
		seen := make(map[string]struct{})
		for decoder.More() {
			keyToken, err := decoder.Token()
			if err != nil {
				return fmt.Errorf("invalid JSON object key: %w", err)
			}
			key, ok := keyToken.(string)
			if !ok {
				return errors.New("invalid JSON object key")
			}
			if _, exists := seen[key]; exists {
				return fmt.Errorf("duplicate JSON key %q", key)
			}
			seen[key] = struct{}{}
			if err := scanJSONValue(decoder); err != nil {
				return err
			}
		}
		end, err := decoder.Token()
		if err != nil || end != json.Delim('}') {
			return errors.New("invalid JSON object")
		}
	case '[':
		for decoder.More() {
			if err := scanJSONValue(decoder); err != nil {
				return err
			}
		}
		end, err := decoder.Token()
		if err != nil || end != json.Delim(']') {
			return errors.New("invalid JSON array")
		}
	default:
		return errors.New("invalid JSON delimiter")
	}
	return nil
}

func canonicalObject(raw json.RawMessage, label string, max int) (json.RawMessage, error) {
	if len(raw) == 0 || len(raw) > max {
		return nil, fmt.Errorf("%s is empty or exceeds its size bound", label)
	}
	if err := rejectDuplicateKeys(raw); err != nil {
		return nil, fmt.Errorf("%s: %w", label, err)
	}
	var value any
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	if err := decoder.Decode(&value); err != nil {
		return nil, fmt.Errorf("%s is invalid JSON: %w", label, err)
	}
	object, ok := value.(map[string]any)
	if !ok || object == nil {
		return nil, fmt.Errorf("%s must be a JSON object", label)
	}
	canonical, err := json.Marshal(object)
	if err != nil {
		return nil, fmt.Errorf("%s cannot be encoded: %w", label, err)
	}
	return canonical, nil
}

func normalizeRequest(input Request) (Request, []byte, error) {
	req := input
	if req.Role != RoleExecutor && req.Role != RoleVerifier && req.Role != RoleInfer {
		return Request{}, nil, errors.New("role must be executor, verifier, or infer")
	}
	for name, value := range map[string]string{
		"sourceRevision": req.SourceRevision,
		"modelDigest":    req.ModelDigest,
		"modulePin":      req.ModulePin,
		"projectionId":   req.ProjectionID,
	} {
		if value == "" || len(value) > maxFieldBytes || !utf8.ValidString(value) {
			return Request{}, nil, fmt.Errorf("%s is empty, invalid, or too long", name)
		}
	}
	if !validDigest(req.ModelDigest) {
		return Request{}, nil, errors.New("modelDigest must be a lowercase sha256 digest")
	}
	if len(req.ScopeIDs) > maxScopeCount || len(req.PolicyIDs) > maxScopeCount {
		return Request{}, nil, errors.New("scopeIds and policyIds are limited to 128 entries")
	}
	var err error
	req.Context, err = canonicalObject(req.Context, "context", maxContextBytes)
	if err != nil {
		return Request{}, nil, err
	}
	req.ScopeIDs, err = sortedUniqueStrings(req.ScopeIDs, "scopeIds")
	if err != nil {
		return Request{}, nil, err
	}
	req.PolicyIDs, err = sortedUniqueStrings(req.PolicyIDs, "policyIds")
	if err != nil {
		return Request{}, nil, err
	}
	if len(req.Artifacts) > maxArtifactCount {
		return Request{}, nil, errors.New("artifacts are limited to 128 entries")
	}
	req.Artifacts = append([]Artifact{}, req.Artifacts...)
	sort.Slice(req.Artifacts, func(i, j int) bool { return req.Artifacts[i].Path < req.Artifacts[j].Path })
	seen := make(map[string]struct{}, len(req.Artifacts))
	total := 0
	for i := range req.Artifacts {
		artifact := &req.Artifacts[i]
		clean, key, err := portablePath(artifact.Path)
		if err != nil {
			return Request{}, nil, fmt.Errorf("artifact path: %w", err)
		}
		artifact.Path = clean
		if _, exists := seen[key]; exists {
			return Request{}, nil, fmt.Errorf("duplicate or aliased artifact path %q", clean)
		}
		seen[key] = struct{}{}
		if artifact.Mode != "0644" && artifact.Mode != "0755" && artifact.Mode != "0600" {
			return Request{}, nil, fmt.Errorf("artifact %q has unsupported mode", clean)
		}
		if len(artifact.Content) > maxArtifactBytes {
			return Request{}, nil, fmt.Errorf("artifact %q exceeds the 8 MiB bound", clean)
		}
		total += len(artifact.Content)
		if total > 16<<20 {
			return Request{}, nil, errors.New("artifact bytes exceed the 16 MiB aggregate bound")
		}
		if !validDigest(artifact.Digest) || digest(artifact.Content) != artifact.Digest {
			return Request{}, nil, fmt.Errorf("artifact %q digest does not match its supplied bytes", clean)
		}
		if artifact.Content == nil {
			artifact.Content = []byte{}
		}
	}
	encoded, err := json.Marshal(req)
	if err != nil {
		return Request{}, nil, err
	}
	if len(encoded) > maxJSONBytes {
		return Request{}, nil, errors.New("request exceeds the 32 MiB bound")
	}
	return req, encoded, nil
}

func sortedUniqueStrings(values []string, label string) ([]string, error) {
	out := append([]string{}, values...)
	sort.Strings(out)
	for i, value := range out {
		if value == "" || len(value) > maxFieldBytes || !utf8.ValidString(value) {
			return nil, fmt.Errorf("%s contains an invalid value", label)
		}
		if i > 0 && value == out[i-1] {
			return nil, fmt.Errorf("%s contains a duplicate value", label)
		}
	}
	return out, nil
}

func portablePath(raw string) (string, string, error) {
	if raw == "" || len(raw) > maxFieldBytes || !utf8.ValidString(raw) ||
		strings.Contains(raw, "\\") || strings.HasPrefix(raw, "/") || strings.Contains(raw, ":") {
		return "", "", errors.New("must be a relative portable slash path")
	}
	parts := strings.Split(raw, "/")
	for _, part := range parts {
		if part == "" || part == "." || part == ".." || strings.HasSuffix(part, ".") || strings.HasSuffix(part, " ") {
			return "", "", errors.New("contains an empty, dot, parent, or non-portable component")
		}
		for _, r := range part {
			if r < 32 || strings.ContainsRune("<>:\"|?*", r) {
				return "", "", errors.New("contains a non-portable character")
			}
		}
		base := strings.ToLower(strings.SplitN(part, ".", 2)[0])
		switch base {
		case "con", "prn", "aux", "nul", "com1", "com2", "com3", "com4", "com5", "com6", "com7", "com8", "com9", "lpt1", "lpt2", "lpt3", "lpt4", "lpt5", "lpt6", "lpt7", "lpt8", "lpt9":
			return "", "", errors.New("contains a reserved Windows path component")
		}
	}
	clean := strings.Join(parts, "/")
	return clean, strings.ToLower(clean), nil
}

func validDigest(value string) bool {
	if len(value) != len("sha256:")+64 || !strings.HasPrefix(value, "sha256:") {
		return false
	}
	for _, r := range value[len("sha256:"):] {
		if (r < '0' || r > '9') && (r < 'a' || r > 'f') {
			return false
		}
	}
	return true
}

func digest(data []byte) string {
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:])
}
