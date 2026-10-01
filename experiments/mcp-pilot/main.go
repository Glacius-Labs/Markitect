// Command markitect-mcp is a deliberately small, local MCP stdio pilot.
// Repository, revision, and CLI are fixed at process startup; tool arguments
// can select only read-only Markitect queries.
package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"
)

const (
	protocolVersion       = "2025-11-25"
	legacyProtocolVersion = "2025-06-18"
	maxMessageBytes       = 1 << 20
	maxOutputBytes        = 2 << 20
)

var fullCommit = regexp.MustCompile(`^(?:[0-9a-f]{40}|[0-9a-f]{64})$`)

type server struct {
	cli      string
	repo     string
	revision string
	initDone bool
	ready    bool
}

type request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type limitedBuffer struct {
	buf bytes.Buffer
	max int
}

func (b *limitedBuffer) Write(p []byte) (int, error) {
	if b.buf.Len()+len(p) > b.max {
		return 0, errors.New("Markitect output exceeded the 2 MiB pilot limit")
	}
	return b.buf.Write(p)
}

func main() {
	cliArg := flag.String("markitect", "markitect", "path to the Markitect executable")
	repoArg := flag.String("repo", "", "absolute path to the repository to expose")
	revisionArg := flag.String("revision", "", "full immutable Git commit SHA to expose")
	flag.Parse()
	if flag.NArg() != 0 {
		fatal("unexpected positional arguments")
	}
	if *repoArg == "" || *revisionArg == "" {
		fatal("--repo and --revision are required")
	}
	if !filepath.IsAbs(*repoArg) {
		fatal("--repo must be an absolute path")
	}
	if !fullCommit.MatchString(*revisionArg) {
		fatal("--revision must be a full 40- or 64-character lowercase commit SHA")
	}
	repo, err := filepath.Abs(*repoArg)
	if err != nil {
		fatal("cannot resolve --repo: " + err.Error())
	}
	info, err := os.Stat(repo)
	if err != nil || !info.IsDir() {
		fatal("--repo must name an existing directory")
	}
	repo, err = canonicalGitRepoRoot(repo)
	if err != nil {
		fatal(err.Error())
	}
	cli, err := exec.LookPath(*cliArg)
	if err != nil {
		fatal("cannot find Markitect executable: " + err.Error())
	}
	if !filepath.IsAbs(cli) {
		cli, err = filepath.Abs(cli)
		if err != nil {
			fatal("cannot resolve Markitect executable: " + err.Error())
		}
	}
	if err := validateCommit(repo, *revisionArg); err != nil {
		fatal(err.Error())
	}
	s := &server{cli: cli, repo: repo, revision: *revisionArg}
	if err := s.serve(os.Stdin, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "markitect-mcp:", err)
		os.Exit(1)
	}
}

func fatal(message string) {
	fmt.Fprintln(os.Stderr, "markitect-mcp:", message)
	os.Exit(2)
}

func validateCommit(repo, revision string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", "--no-replace-objects", "-c", "safe.directory="+filepath.ToSlash(repo), "-C", repo, "rev-parse", "--verify", revision+"^{commit}")
	cmd.Env = cleanGitEnv()
	var out limitedBuffer
	out.max = 256
	cmd.Stdout = &out
	cmd.Stderr = io.Discard
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("revision is not a commit in --repo: %w", err)
	}
	if strings.TrimSpace(out.buf.String()) != revision {
		return errors.New("Git resolved --revision to a different commit; expected an exact full SHA")
	}
	return nil
}

func canonicalGitRepoRoot(repo string) (string, error) {
	resolvedRepo, err := filepath.EvalSymlinks(repo)
	if err != nil {
		return "", fmt.Errorf("cannot resolve --repo path: %w", err)
	}
	resolvedRepo, err = filepath.Abs(resolvedRepo)
	if err != nil {
		return "", fmt.Errorf("cannot resolve --repo path: %w", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", "--no-replace-objects", "-c", "safe.directory="+filepath.ToSlash(resolvedRepo), "-C", resolvedRepo, "rev-parse", "--show-toplevel")
	cmd.Env = cleanGitEnv()
	var out limitedBuffer
	out.max = 4096
	cmd.Stdout = &out
	cmd.Stderr = io.Discard
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("--repo must be inside a Git checkout: %w", err)
	}
	gitRoot, err := filepath.Abs(strings.TrimSpace(out.buf.String()))
	if err != nil {
		return "", fmt.Errorf("cannot resolve Git repository root: %w", err)
	}
	gitRoot, err = filepath.EvalSymlinks(gitRoot)
	if err != nil {
		return "", fmt.Errorf("cannot resolve Git repository root: %w", err)
	}
	if !samePath(resolvedRepo, gitRoot) {
		return "", errors.New("--repo must name the canonical Git top-level directory")
	}
	return gitRoot, nil
}

func samePath(left, right string) bool {
	left = filepath.Clean(left)
	right = filepath.Clean(right)
	leftInfo, leftErr := os.Stat(left)
	rightInfo, rightErr := os.Stat(right)
	if leftErr == nil && rightErr == nil {
		return os.SameFile(leftInfo, rightInfo)
	}
	if runtime.GOOS == "windows" {
		return strings.EqualFold(left, right)
	}
	return left == right
}

func cleanGitEnv() []string {
	base := os.Environ()
	clean := make([]string, 0, len(base))
	for _, entry := range base {
		key, _, ok := strings.Cut(entry, "=")
		if ok && strings.HasPrefix(strings.ToUpper(key), "GIT_") {
			continue
		}
		clean = append(clean, entry)
	}
	return clean
}

func (s *server) serve(input io.Reader, output io.Writer) error {
	scanner := bufio.NewScanner(input)
	scanner.Buffer(make([]byte, 4096), maxMessageBytes)
	w := bufio.NewWriter(output)
	for scanner.Scan() {
		line := scanner.Bytes()
		var req request
		if err := json.Unmarshal(line, &req); err != nil {
			if err := writeRPC(w, nil, nil, &rpcError{Code: -32700, Message: "Parse error"}); err != nil {
				return err
			}
			continue
		}
		if req.JSONRPC != "2.0" || req.Method == "" {
			if len(req.ID) == 0 {
				continue
			}
			if err := writeRPC(w, req.ID, nil, &rpcError{Code: -32600, Message: "Invalid Request"}); err != nil {
				return err
			}
			continue
		}
		if len(req.ID) > 0 && string(req.ID) == "null" {
			if err := writeRPC(w, req.ID, nil, &rpcError{Code: -32600, Message: "Invalid Request"}); err != nil {
				return err
			}
			continue
		}
		result, rpcErr, reply := s.dispatch(req)
		if !reply {
			continue
		}
		if err := writeRPC(w, req.ID, result, rpcErr); err != nil {
			return err
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("read MCP input: %w", err)
	}
	return nil
}

func writeRPC(w *bufio.Writer, id json.RawMessage, result any, rpcErr *rpcError) error {
	response := map[string]any{"jsonrpc": "2.0", "id": id}
	if rpcErr != nil {
		response["error"] = rpcErr
	} else {
		response["result"] = result
	}
	data, err := json.Marshal(response)
	if err != nil {
		return err
	}
	if _, err := w.Write(append(data, '\n')); err != nil {
		return err
	}
	return w.Flush()
}

func (s *server) dispatch(req request) (any, *rpcError, bool) {
	if len(req.ID) == 0 {
		if req.Method == "notifications/initialized" && s.initDone {
			s.ready = true
		}
		return nil, nil, false
	}
	switch req.Method {
	case "initialize":
		if s.initDone {
			return nil, &rpcError{Code: -32600, Message: "Initialize may only be sent once"}, true
		}
		var params struct {
			ProtocolVersion string `json:"protocolVersion"`
		}
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return nil, &rpcError{Code: -32602, Message: "Invalid params"}, true
		}
		if params.ProtocolVersion == "" {
			return nil, &rpcError{Code: -32602, Message: "Invalid params: protocolVersion is required"}, true
		}
		s.initDone = true
		return map[string]any{
			"protocolVersion": negotiateProtocolVersion(params.ProtocolVersion),
			"capabilities":    map[string]any{"tools": map[string]any{}},
			"serverInfo":      map[string]string{"name": "markitect-readonly-pilot", "version": "0.1.0"},
		}, nil, true
	case "ping":
		if !s.initDone {
			return nil, &rpcError{Code: -32002, Message: "Server not initialized"}, true
		}
		return map[string]any{}, nil, true
	case "notifications/initialized", "notifications/cancelled":
		return nil, nil, false
	case "tools/list":
		if !s.ready {
			return nil, &rpcError{Code: -32002, Message: "Server not initialized"}, true
		}
		return map[string]any{"tools": toolDefinitions()}, nil, true
	case "tools/call":
		if !s.ready {
			return nil, &rpcError{Code: -32002, Message: "Server not initialized"}, true
		}
		var params struct {
			Name string `json:"name"`
		}
		if err := json.Unmarshal(req.Params, &params); err != nil || params.Name == "" {
			return nil, &rpcError{Code: -32602, Message: "Invalid params"}, true
		}
		if params.Name != "find" && params.Name != "explain" && params.Name != "context" {
			return nil, &rpcError{Code: -32601, Message: "Unknown tool: " + params.Name}, true
		}
		return s.callTool(req.Params), nil, true
	default:
		if !s.initDone {
			return nil, &rpcError{Code: -32002, Message: "Server not initialized"}, true
		}
		return nil, &rpcError{Code: -32601, Message: "Method not found"}, true
	}
}

func negotiateProtocolVersion(offered string) string {
	switch offered {
	case protocolVersion, legacyProtocolVersion:
		return offered
	default:
		return protocolVersion
	}
}

func toolDefinitions() []map[string]any {
	return []map[string]any{
		{
			"name":        "find",
			"annotations": readOnlyAnnotations(),
			"description": "Find resources in the fixed Markitect repository snapshot. Literal text and exact kind/namespace filters only.",
			"inputSchema": map[string]any{
				"type": "object", "additionalProperties": false,
				"properties": map[string]any{
					"query":     map[string]any{"type": "string", "maxLength": 512, "description": "Literal search text (required by this pilot)."},
					"kind":      map[string]any{"type": "string", "enum": []string{"Text", "Rule", "Contract", "Workflow", "Skill", "Agent"}},
					"namespace": map[string]any{"type": "string", "maxLength": 63},
				},
				"required": []string{"query"},
			},
		},
		{
			"name":        "explain",
			"annotations": readOnlyAnnotations(),
			"description": "Explain one resource's ownership and explicit direct relationships in the fixed snapshot.",
			"inputSchema": map[string]any{
				"type": "object", "additionalProperties": false,
				"properties": map[string]any{
					"kind":      map[string]any{"type": "string", "enum": []string{"Text", "Rule", "Contract", "Workflow", "Skill", "Agent", "Project"}},
					"name":      map[string]any{"type": "string", "minLength": 1, "maxLength": 63},
					"namespace": map[string]any{"type": "string", "maxLength": 63, "description": "Required for namespaced resources; omit for Project."},
				},
				"required": []string{"kind", "name"},
			},
		},
		{
			"name":        "context",
			"annotations": readOnlyAnnotations(),
			"description": "Compile a resource's declared dependency closure and file inputs from the fixed snapshot.",
			"inputSchema": map[string]any{
				"type": "object", "additionalProperties": false,
				"properties": map[string]any{
					"kind":      map[string]any{"type": "string", "enum": []string{"Text", "Rule", "Contract", "Workflow", "Skill", "Agent"}},
					"name":      map[string]any{"type": "string", "minLength": 1, "maxLength": 63},
					"namespace": map[string]any{"type": "string", "minLength": 1, "maxLength": 63},
				},
				"required": []string{"kind", "name", "namespace"},
			},
		},
	}
}

func readOnlyAnnotations() map[string]bool {
	return map[string]bool{
		"readOnlyHint":    true,
		"destructiveHint": false,
		"idempotentHint":  true,
		"openWorldHint":   false,
	}
}

func (s *server) callTool(raw json.RawMessage) map[string]any {
	var params struct {
		Name      string         `json:"name"`
		Arguments map[string]any `json:"arguments"`
	}
	if err := json.Unmarshal(raw, &params); err != nil || params.Name == "" || params.Arguments == nil {
		return toolError("Invalid tool arguments object.")
	}
	args := params.Arguments
	var cliArgs []string
	switch params.Name {
	case "find":
		if err := onlyKeys(args, "query", "kind", "namespace"); err != nil {
			return toolError(err.Error())
		}
		query, err := stringArg(args, "query", true, 512)
		if err != nil {
			return toolError(err.Error())
		}
		cliArgs = []string{"find", "--query", query}
		if _, ok := args["kind"]; ok {
			v, err := stringArg(args, "kind", false, 16)
			if err != nil || !validKind(v, false) {
				return toolError("kind must be one of Text, Rule, Contract, Workflow, Skill, or Agent")
			}
			cliArgs = append(cliArgs, "--kind", v)
		}
		if _, ok := args["namespace"]; ok {
			v, err := stringArg(args, "namespace", false, 63)
			if err != nil {
				return toolError("namespace must be a non-empty string of at most 63 characters")
			}
			cliArgs = append(cliArgs, "--namespace", v)
		}
	case "explain", "context":
		if err := onlyKeys(args, "kind", "name", "namespace"); err != nil {
			return toolError(err.Error())
		}
		kind, err := stringArg(args, "kind", true, 16)
		if err != nil || !validKind(kind, params.Name == "explain") {
			return toolError("kind is not supported for this tool")
		}
		name, err := stringArg(args, "name", true, 63)
		if err != nil {
			return toolError("name must be a non-empty string of at most 63 characters")
		}
		_, nsPresent := args["namespace"]
		namespaceRequired := params.Name == "context" || kind != "Project"
		if !namespaceRequired && nsPresent {
			return toolError("namespace must be omitted for Project")
		}
		if namespaceRequired && !nsPresent {
			return toolError("namespace is required for this resource")
		}
		cliArgs = []string{params.Name, "--kind", kind, "--name", name}
		if nsPresent {
			v, err := stringArg(args, "namespace", true, 63)
			if err != nil {
				return toolError("namespace must be a non-empty string of at most 63 characters")
			}
			cliArgs = append(cliArgs, "--namespace", v)
		}
	default:
		return toolError("Tool not found.")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	argv := append([]string(nil), cliArgs...)
	argv = append(argv, "--repo", s.repo, "--revision", s.revision)
	cmd := exec.CommandContext(ctx, s.cli, argv...)
	cmd.Env = cleanGitEnv()
	stdout := &limitedBuffer{max: maxOutputBytes}
	stderr := &limitedBuffer{max: 16 << 10}
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	err := cmd.Run()
	if err != nil {
		message := strings.TrimSpace(stderr.buf.String())
		if message == "" {
			message = "Markitect query failed."
		}
		return toolError(message)
	}
	return map[string]any{"content": []map[string]string{{"type": "text", "text": stdout.buf.String()}}, "isError": false}
}

func toolError(message string) map[string]any {
	return map[string]any{"content": []map[string]string{{"type": "text", "text": message}}, "isError": true}
}

func onlyKeys(args map[string]any, keys ...string) error {
	allowed := make(map[string]bool, len(keys))
	for _, key := range keys {
		allowed[key] = true
	}
	for key := range args {
		if !allowed[key] {
			return fmt.Errorf("unknown argument %q", key)
		}
	}
	return nil
}

func stringArg(args map[string]any, key string, required bool, max int) (string, error) {
	value, ok := args[key]
	if !ok {
		if required {
			return "", fmt.Errorf("%s is required", key)
		}
		return "", nil
	}
	text, ok := value.(string)
	if !ok || len(text) == 0 || len(text) > max {
		return "", fmt.Errorf("%s must be a non-empty string of at most %d characters", key, max)
	}
	return text, nil
}

func validKind(kind string, allowProject bool) bool {
	switch kind {
	case "Text", "Rule", "Contract", "Workflow", "Skill", "Agent":
		return true
	case "Project":
		return allowProject
	default:
		return false
	}
}
