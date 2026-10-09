package agentexec

import (
	"encoding/json"
	"errors"
)

// PrepareInvocation normalizes explicit inputs and allocates a fresh nonce and
// invocation identity. It performs no process, filesystem or provider work.
func PrepareInvocation(request Request) (Invocation, []byte, error) {
	req, encoded, err := normalizeRequest(request)
	if err != nil {
		return Invocation{}, nil, err
	}
	return prepareInvocation(req, encoded)
}

func prepareInvocation(req Request, encoded []byte) (Invocation, []byte, error) {
	runID, err := randomID()
	if err != nil {
		return Invocation{}, nil, errors.New("could not allocate an invocation identity")
	}
	nonce, err := randomID()
	if err != nil {
		return Invocation{}, nil, errors.New("could not allocate an invocation nonce")
	}
	// Do not retain mutable artifact bytes from the caller across the boundary.
	for i := range req.Artifacts {
		req.Artifacts[i].Content = append([]byte{}, req.Artifacts[i].Content...)
	}
	invocation := Invocation{APIVersion: APIVersion, RunID: runID, Nonce: nonce, InputDigest: digest(encoded), Request: req}
	wire, err := json.Marshal(invocation)
	if err != nil || len(wire) > maxJSONBytes {
		return Invocation{}, nil, errors.New("invocation could not be encoded within the protocol bound")
	}
	return invocation, wire, nil
}

// DecodeResponse applies the same closed protocol, role, nonce, evidence and
// native-work validation for every transport. A provider output schema alone
// does not establish that a response belongs to the current invocation.
func DecodeResponse(data []byte, invocation Invocation, workspaceMode string) (Response, error) {
	req, encoded, err := normalizeRequest(invocation.Request)
	if err != nil {
		return Response{}, err
	}
	if invocation.APIVersion != APIVersion || invocation.RunID == "" || invocation.Nonce == "" || invocation.InputDigest != digest(encoded) {
		return Response{}, errors.New("invocation binding is invalid")
	}
	var response Response
	if err := strictDecode(data, &response); err != nil {
		return Response{}, errors.New("external runner returned an invalid response")
	}
	if err := validateResponse(response, req, invocation, workspaceMode); err != nil {
		return Response{}, err
	}
	return response, nil
}
