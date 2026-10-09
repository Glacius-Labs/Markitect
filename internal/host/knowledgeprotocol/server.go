// Package knowledgeprotocol implements Markitect's bounded, read-only MCP
// stdio transport for the project knowledge service.
package knowledgeprotocol

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"reflect"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/host/knowledgeevidence"
	"github.com/Glacius-Labs/Markitect/internal/host/projectgraph"
)

const (
	latestProtocolVersion = "2025-11-25"
	maxMessageBytes       = 1 << 20
)

var supportedProtocolVersions = map[string]bool{
	"2025-11-25": true,
	"2025-06-18": true,
	"2024-11-05": true,
}

// ToolRequest is a closed service-neutral request produced only after scope,
// action, record selection and traversal limits have been validated.
type ToolRequest struct {
	Scope   projectgraph.Selection
	Query   projectgraph.Request
	Records knowledgeevidence.Selection
}

// Handler executes one already validated read-only knowledge request.
type Handler func(ToolRequest) (any, error)

type rpcRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type rpcResponse struct {
	JSONRPC string    `json:"jsonrpc"`
	ID      any       `json:"id"`
	Result  any       `json:"result,omitempty"`
	Error   *rpcError `json:"error,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// Serve processes newline-delimited JSON-RPC messages until EOF. Each message
// is independently bounded; oversized input closes this one-shot stream.
func Serve(in io.Reader, out io.Writer, handler Handler) error {
	if in == nil || out == nil || handler == nil {
		return errors.New("knowledge MCP requires input, output and a handler")
	}
	scanner := bufio.NewScanner(in)
	scanner.Buffer(make([]byte, 4096), maxMessageBytes)
	state := connectionState{}
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) > maxMessageBytes {
			return errors.New("MCP message exceeds 1 MiB")
		}
		response, respond := process(line, handler, &state)
		if respond {
			if err := writeLine(out, response); err != nil {
				return err
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return errors.New("MCP message exceeds 1 MiB or could not be read")
	}
	return nil
}

type connectionState struct {
	initialized bool
	ready       bool
	version     string
}

func process(line []byte, handler Handler, state *connectionState) (rpcResponse, bool) {
	var request rpcRequest
	if err := decodeClosed(line, &request); err != nil || request.JSONRPC != "2.0" || request.Method == "" {
		return failure(nil, -32700, "invalid JSON-RPC request"), true
	}
	if len(request.ID) != 0 && !validID(request.ID) {
		return failure(nil, -32600, "invalid JSON-RPC request id"), true
	}
	if len(request.ID) == 0 {
		if request.Method == "notifications/initialized" {
			if state.initialized {
				state.ready = true
			}
			return rpcResponse{}, false
		}
		if request.Method == "notifications/cancelled" {
			return rpcResponse{}, false
		}
		// MCP notifications never receive responses, including unsupported ones.
		return rpcResponse{}, false
	}
	id := json.RawMessage(bytes.TrimSpace(request.ID))
	switch request.Method {
	case "initialize":
		if state.initialized {
			return failure(id, -32600, "initialize was already completed"), true
		}
		var params struct {
			ProtocolVersion string          `json:"protocolVersion"`
			Capabilities    json.RawMessage `json:"capabilities"`
			ClientInfo      json.RawMessage `json:"clientInfo"`
			Meta            json.RawMessage `json:"_meta,omitempty"`
		}
		if err := decodeClosed(request.Params, &params); err != nil || params.ProtocolVersion == "" {
			return failure(id, -32602, "invalid initialize parameters"), true
		}
		if supportedProtocolVersions[params.ProtocolVersion] {
			state.version = params.ProtocolVersion
		} else {
			state.version = latestProtocolVersion
		}
		state.initialized = true
		return success(id, map[string]any{
			"protocolVersion": state.version,
			"capabilities":    map[string]any{"tools": map[string]any{"listChanged": false}},
			"serverInfo":      map[string]string{"name": "markitect-project-knowledge", "version": "1"},
		}), true
	case "ping":
		return success(id, map[string]any{}), true
	case "tools/list":
		if !state.ready {
			return failure(id, -32002, "server is not initialized"), true
		}
		return success(id, map[string]any{"tools": toolDefinitions()}), true
	case "tools/call":
		if !state.ready {
			return failure(id, -32002, "server is not initialized"), true
		}
		var params struct {
			Name      string          `json:"name"`
			Arguments json.RawMessage `json:"arguments"`
			Meta      json.RawMessage `json:"_meta,omitempty"`
		}
		if err := decodeClosed(request.Params, &params); err != nil || params.Name == "" || len(params.Arguments) == 0 {
			return failure(id, -32602, "invalid tool call parameters"), true
		}
		toolRequest, err := parseToolRequest(params.Name, params.Arguments)
		if err != nil {
			return failure(id, -32602, "invalid knowledge tool arguments"), true
		}
		value, err := handler(toolRequest)
		if err != nil {
			return success(id, map[string]any{
				"content": []any{map[string]string{"type": "text", "text": "knowledge query failed; check the explicit scope, target and bounded selectors"}},
				"isError": true,
			}), true
		}
		encoded, err := json.Marshal(value)
		if err != nil {
			return success(id, map[string]any{
				"content": []any{map[string]string{"type": "text", "text": "knowledge result could not be encoded"}},
				"isError": true,
			}), true
		}
		result := map[string]any{
			"content": []any{map[string]string{"type": "text", "text": string(encoded)}},
			"isError": false,
		}
		if state.version == "2025-06-18" || state.version == "2025-11-25" {
			result["structuredContent"] = json.RawMessage(encoded)
		}
		return success(id, result), true
	default:
		return failure(id, -32601, "method not found"), true
	}
}

func parseToolRequest(name string, raw json.RawMessage) (ToolRequest, error) {
	action, ok := map[string]projectgraph.Action{
		"knowledge_graph":     projectgraph.ActionGraph,
		"knowledge_relations": projectgraph.ActionRelations,
		"knowledge_explain":   projectgraph.ActionExplain,
		"knowledge_trace":     projectgraph.ActionTrace,
		"knowledge_history":   projectgraph.ActionHistory,
		"knowledge_coverage":  projectgraph.ActionCoverage,
	}[name]
	if !ok {
		return ToolRequest{}, errors.New("unknown knowledge tool")
	}
	var args toolArguments
	if err := decodeClosed(raw, &args); err != nil {
		return ToolRequest{}, err
	}
	if (args.ManagerID == nil) == (args.ProjectScope == nil) {
		return ToolRequest{}, errors.New("exactly one knowledge scope is required")
	}
	selection := projectgraph.Selection{}
	if args.ManagerID != nil {
		if *args.ManagerID == "" {
			return ToolRequest{}, errors.New("manager id is empty")
		}
		selection.ManagerID = *args.ManagerID
	} else {
		if !*args.ProjectScope {
			return ToolRequest{}, errors.New("project scope must be true")
		}
		selection.ProjectScope = true
	}
	requiresTarget := action == projectgraph.ActionRelations || action == projectgraph.ActionExplain || action == projectgraph.ActionTrace || action == projectgraph.ActionHistory
	if requiresTarget != (args.TargetID != "") {
		return ToolRequest{}, errors.New("target selection does not match tool action")
	}
	if action != projectgraph.ActionTrace && (args.Reverse || args.Bidirectional || args.MaxDepth != nil || args.MaxSteps != nil || args.MaxResults != nil) {
		return ToolRequest{}, errors.New("traversal limits apply only to knowledge_trace")
	}
	if args.Reverse && args.Bidirectional {
		return ToolRequest{}, errors.New("reverse and bidirectional are mutually exclusive")
	}
	for _, value := range []*int{args.MaxDepth, args.MaxSteps, args.MaxResults} {
		if value != nil && *value <= 0 {
			return ToolRequest{}, errors.New("traversal limits must be positive")
		}
	}
	if args.MaxDepth != nil && *args.MaxDepth > 32 || args.MaxSteps != nil && *args.MaxSteps > 50000 || args.MaxResults != nil && *args.MaxResults > 5000 {
		return ToolRequest{}, errors.New("traversal limits exceed the supported maximum")
	}
	request := projectgraph.Request{Action: action, TargetID: args.TargetID, Reverse: args.Reverse}
	request.Bidirectional = args.Bidirectional
	if args.MaxDepth != nil {
		request.MaxDepth = *args.MaxDepth
	}
	if args.MaxSteps != nil {
		request.MaxSteps = *args.MaxSteps
	}
	if args.MaxResults != nil {
		request.MaxResults = *args.MaxResults
	}
	records := knowledgeevidence.Selection{IncludeBriefingHistory: args.BriefingHistory}
	if args.RunID != "" {
		records.RunIDs = []string{args.RunID}
	}
	if args.ExplorationID != "" {
		records.ExplorationIDs = []string{args.ExplorationID}
	}
	if args.SessionID != "" {
		records.BrownfieldSessionIDs = []string{args.SessionID}
	}
	return ToolRequest{Scope: selection, Query: request, Records: records}, nil
}

type toolArguments struct {
	ManagerID       *string `json:"managerId,omitempty"`
	ProjectScope    *bool   `json:"projectScope,omitempty"`
	TargetID        string  `json:"targetId,omitempty"`
	Reverse         bool    `json:"reverse,omitempty"`
	Bidirectional   bool    `json:"bidirectional,omitempty"`
	MaxDepth        *int    `json:"maxDepth,omitempty"`
	MaxSteps        *int    `json:"maxSteps,omitempty"`
	MaxResults      *int    `json:"maxResults,omitempty"`
	RunID           string  `json:"runId,omitempty"`
	ExplorationID   string  `json:"explorationId,omitempty"`
	SessionID       string  `json:"sessionId,omitempty"`
	BriefingHistory bool    `json:"briefingHistory,omitempty"`
}

func toolDefinitions() []map[string]any {
	definitions := make([]map[string]any, 0, 6)
	for _, item := range []struct {
		name, title, description string
		target                   bool
		trace                    bool
	}{
		{"knowledge_graph", "Project knowledge graph", "Read the explicitly scoped project knowledge graph.", false, false},
		{"knowledge_relations", "Knowledge relations", "Read relations directly incident to one visible node.", true, false},
		{"knowledge_explain", "Explain knowledge node", "Read the selected node and its visible relation witnesses.", true, false},
		{"knowledge_trace", "Trace knowledge graph", "Traverse a bounded visible knowledge relation path.", true, true},
		{"knowledge_history", "Knowledge history", "Read explicit history claims for one visible node.", true, false},
		{"knowledge_coverage", "Knowledge coverage", "Read scoped model, artifact, check and record coverage states.", false, false},
	} {
		definitions = append(definitions, map[string]any{
			"name": item.name, "title": item.title, "description": item.description,
			"inputSchema": inputSchema(item.target, item.trace),
			"annotations": map[string]any{"readOnlyHint": true, "destructiveHint": false, "openWorldHint": false},
		})
	}
	return definitions
}

func inputSchema(target, trace bool) map[string]any {
	properties := map[string]any{
		"managerId":       map[string]any{"type": "string", "minLength": 1},
		"projectScope":    map[string]any{"type": "boolean", "const": true},
		"runId":           map[string]string{"type": "string"},
		"explorationId":   map[string]string{"type": "string"},
		"sessionId":       map[string]string{"type": "string"},
		"briefingHistory": map[string]any{"type": "boolean", "description": "Opt in to scoped briefing history."},
	}
	if target {
		properties["targetId"] = map[string]any{"type": "string", "minLength": 1}
	}
	if trace {
		properties["reverse"] = map[string]string{"type": "boolean"}
		properties["bidirectional"] = map[string]string{"type": "boolean"}
		properties["maxDepth"] = map[string]any{"type": "integer", "minimum": 1, "maximum": 32}
		properties["maxSteps"] = map[string]any{"type": "integer", "minimum": 1, "maximum": 50000}
		properties["maxResults"] = map[string]any{"type": "integer", "minimum": 1, "maximum": 5000}
	}
	required := []string{}
	if target {
		required = append(required, "targetId")
	}
	return map[string]any{
		"type": "object", "properties": properties, "additionalProperties": false,
		"oneOf": []any{
			map[string]any{"required": []string{"managerId"}, "not": map[string]any{"required": []string{"projectScope"}}},
			map[string]any{"required": []string{"projectScope"}, "not": map[string]any{"required": []string{"managerId"}}},
		},
		"required": required,
	}
}

func decodeClosed(data []byte, target any) error {
	if err := validateJSONKeys(data, reflect.TypeOf(target)); err != nil {
		return err
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return errors.New("trailing JSON data")
	}
	return nil
}

func validateJSONKeys(data []byte, target reflect.Type) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	if err := scanJSONValue(decoder, target); err != nil {
		return err
	}
	if _, err := decoder.Token(); !errors.Is(err, io.EOF) {
		return errors.New("trailing JSON data")
	}
	return nil
}

// scanJSONValue preserves object-key occurrences while decoding. encoding/json
// otherwise keeps only the last duplicate and matches struct fields without
// regard to case, which can silently replace a selected scope or target.
func scanJSONValue(decoder *json.Decoder, target reflect.Type) error {
	token, err := decoder.Token()
	if err != nil {
		return err
	}
	delimiter, ok := token.(json.Delim)
	if !ok {
		return nil
	}
	switch delimiter {
	case '{':
		target = indirectType(target)
		seenExact := map[string]bool{}
		seenFields := map[string]string{}
		for decoder.More() {
			keyToken, err := decoder.Token()
			if err != nil {
				return err
			}
			key, ok := keyToken.(string)
			if !ok {
				return errors.New("JSON object key is invalid")
			}
			if seenExact[key] {
				return fmt.Errorf("duplicate JSON object key %q", key)
			}
			seenExact[key] = true
			childType := reflect.Type(nil)
			if target != nil {
				switch target.Kind() {
				case reflect.Struct:
					fieldType, canonicalName, found := jsonFieldType(target, key)
					if found {
						folded := strings.ToLower(canonicalName)
						if previous, duplicate := seenFields[folded]; duplicate {
							return fmt.Errorf("JSON fields %q and %q name the same struct field", previous, key)
						}
						seenFields[folded] = key
						if key != canonicalName {
							return fmt.Errorf("JSON field %q must use canonical spelling %q", key, canonicalName)
						}
						childType = fieldType
					}
				case reflect.Map:
					childType = target.Elem()
				}
			}
			if err := scanJSONValue(decoder, childType); err != nil {
				return err
			}
		}
		end, err := decoder.Token()
		if err != nil || end != json.Delim('}') {
			return errors.New("unterminated JSON object")
		}
	case '[':
		target = indirectType(target)
		childType := reflect.Type(nil)
		if target != nil && (target.Kind() == reflect.Array || target.Kind() == reflect.Slice) {
			childType = target.Elem()
		}
		for decoder.More() {
			if err := scanJSONValue(decoder, childType); err != nil {
				return err
			}
		}
		end, err := decoder.Token()
		if err != nil || end != json.Delim(']') {
			return errors.New("unterminated JSON array")
		}
	default:
		return errors.New("unexpected JSON delimiter")
	}
	return nil
}

func indirectType(value reflect.Type) reflect.Type {
	for value != nil && value.Kind() == reflect.Pointer {
		value = value.Elem()
	}
	if value == reflect.TypeOf(json.RawMessage{}) {
		return nil
	}
	return value
}

func jsonFieldType(structType reflect.Type, key string) (reflect.Type, string, bool) {
	for i := 0; i < structType.NumField(); i++ {
		field := structType.Field(i)
		if field.PkgPath != "" {
			continue
		}
		name := strings.Split(field.Tag.Get("json"), ",")[0]
		if name == "-" {
			continue
		}
		if name == "" {
			name = field.Name
		}
		if strings.EqualFold(name, key) {
			return field.Type, name, true
		}
	}
	return nil, "", false
}

func validID(id json.RawMessage) bool {
	trimmed := bytes.TrimSpace(id)
	if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) || trimmed[0] == '{' || trimmed[0] == '[' {
		return false
	}
	if trimmed[0] == '"' {
		var value string
		return json.Unmarshal(trimmed, &value) == nil
	}
	var value any
	decoder := json.NewDecoder(bytes.NewReader(trimmed))
	decoder.UseNumber()
	if err := decoder.Decode(&value); err != nil {
		return false
	}
	_, ok := value.(json.Number)
	return ok
}

func success(id, result any) rpcResponse { return rpcResponse{JSONRPC: "2.0", ID: id, Result: result} }
func failure(id any, code int, message string) rpcResponse {
	return rpcResponse{JSONRPC: "2.0", ID: id, Error: &rpcError{Code: code, Message: message}}
}

func writeLine(out io.Writer, response rpcResponse) error {
	data, err := json.Marshal(response)
	if err != nil {
		return fmt.Errorf("encode MCP response: %w", err)
	}
	if _, err := out.Write(append(data, '\n')); err != nil {
		return errors.New("write MCP response")
	}
	return nil
}
