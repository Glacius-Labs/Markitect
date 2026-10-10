package publish

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

type APIError struct {
	Status  int
	Message string
}

func (e *APIError) Error() string { return fmt.Sprintf("GitHub API HTTP %d: %s", e.Status, e.Message) }
func apiJSON(r Runner, ctx context.Context, endpoint string, target any) error {
	out, err := r.Run(ctx, "", "api", "--hostname", "github.com", "--include", "-H", "X-GitHub-Api-Version: 2026-03-10", endpoint)
	body, status, parseErr := splitHTTPResponse(out)
	if parseErr != nil {
		if err != nil {
			return err
		}
		return parseErr
	}
	if status < 200 || status >= 300 {
		message := strings.TrimSpace(string(body))
		if len(message) > 512 {
			message = message[:512]
		}
		return &APIError{Status: status, Message: message}
	}
	if err != nil {
		return err
	}
	if err := json.Unmarshal(body, target); err != nil {
		return fmt.Errorf("decode GitHub API response for %s: %w", endpoint, err)
	}
	return nil
}
func call(r Runner, ctx context.Context, dir string, args ...string) error {
	_, err := r.Run(ctx, dir, args...)
	return err
}
func isNotFound(err error) bool {
	var apiErr *APIError
	return errors.As(err, &apiErr) && apiErr.Status == 404
}
func splitHTTPResponse(data []byte) ([]byte, int, error) {
	lineEnd := bytes.IndexByte(data, '\n')
	if lineEnd < 0 {
		return nil, 0, errors.New("GitHub API response omitted an HTTP status line")
	}
	fields := strings.Fields(strings.TrimSpace(string(data[:lineEnd])))
	if len(fields) < 2 || !strings.HasPrefix(fields[0], "HTTP/") {
		return nil, 0, errors.New("GitHub API response has an invalid HTTP status line")
	}
	status, err := strconv.Atoi(fields[1])
	if err != nil {
		return nil, 0, errors.New("GitHub API response has an invalid HTTP status")
	}
	sep := bytes.Index(data, []byte("\r\n\r\n"))
	if sep >= 0 {
		return data[sep+4:], status, nil
	}
	sep = bytes.Index(data, []byte("\n\n"))
	if sep >= 0 {
		return data[sep+2:], status, nil
	}
	return nil, status, errors.New("GitHub API response omitted header/body separator")
}
