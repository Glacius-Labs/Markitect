package projectadoption

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"
)

const maxRecordBytes = 32 << 20

var idPattern = regexp.MustCompile(`^[a-z0-9](?:[a-z0-9-]{0,62}[a-z0-9])?$`)

func validID(value string) bool { return idPattern.MatchString(value) }

func DecodeDiscovery(data []byte) (Discovery, error) {
	var value Discovery
	if err := decodeClosedJSON(data, &value); err != nil {
		return Discovery{}, err
	}
	if err := ValidateDiscovery(value); err != nil {
		return Discovery{}, err
	}
	return value, nil
}

func DecodeDiscoveryRequest(data []byte) (DiscoveryRequest, error) {
	var value DiscoveryRequest
	if err := decodeClosedJSON(data, &value); err != nil {
		return DiscoveryRequest{}, err
	}
	if err := validateDiscoveryRequest(value); err != nil {
		return DiscoveryRequest{}, err
	}
	return value, nil
}

func DecodeDistillation(data []byte, discovery Discovery) (Distillation, error) {
	var value Distillation
	if err := decodeClosedJSON(data, &value); err != nil {
		return Distillation{}, err
	}
	if err := ValidateDistillation(discovery, value); err != nil {
		return Distillation{}, err
	}
	return value, nil
}

func DecodeResolution(data []byte, discovery Discovery, report Distillation) (Resolution, error) {
	var value Resolution
	if err := decodeClosedJSON(data, &value); err != nil {
		return Resolution{}, err
	}
	if err := ValidateResolution(discovery, report, value); err != nil {
		return Resolution{}, err
	}
	return value, nil
}

func DecodeAdoptionPlan(data []byte) (AdoptionPlan, error) {
	var value AdoptionPlan
	if err := decodeClosedJSON(data, &value); err != nil {
		return AdoptionPlan{}, err
	}
	if err := ValidateAdoptionPlan(value); err != nil {
		return AdoptionPlan{}, err
	}
	return value, nil
}

func EncodeDiscovery(value Discovery) ([]byte, error) { return encodeClosedJSON(value) }
func EncodeDiscoveryRequest(value DiscoveryRequest) ([]byte, error) {
	return encodeClosedJSON(value)
}
func EncodeDistillation(value Distillation) ([]byte, error) { return encodeClosedJSON(value) }
func EncodeResolution(value Resolution) ([]byte, error)     { return encodeClosedJSON(value) }
func EncodeAdoptionPlan(value AdoptionPlan) ([]byte, error) { return encodeClosedJSON(value) }

func SealDistillation(value *Distillation) {
	if value == nil {
		return
	}
	copy := *value
	copy.Digest = ""
	value.Digest = digestValue(copy)
}

func SealResolution(value *Resolution) {
	if value == nil {
		return
	}
	copy := *value
	copy.Digest = ""
	value.Digest = digestValue(copy)
}

func ProposalDigest(proposal ModelProposal) string { return digestValue(proposal) }

func encodeClosedJSON(value any) ([]byte, error) {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(data, '\n'), nil
}

func decodeClosedJSON(data []byte, target any) error {
	if len(data) == 0 || len(data) > maxRecordBytes || !json.Valid(data) {
		return fmt.Errorf("record must be valid JSON within %d bytes", maxRecordBytes)
	}
	if err := rejectDuplicateFields(data); err != nil {
		return err
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return fmt.Errorf("decode closed adoption record: %w", err)
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return errors.New("adoption record must contain exactly one JSON value")
	}
	return nil
}

func rejectDuplicateFields(data []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	if err := consumeJSONValue(decoder); err != nil {
		return err
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return errors.New("adoption record must contain exactly one JSON value")
	}
	return nil
}

func consumeJSONValue(decoder *json.Decoder) error {
	token, err := decoder.Token()
	if err != nil {
		return err
	}
	delim, ok := token.(json.Delim)
	if !ok {
		return nil
	}
	switch delim {
	case '{':
		seen := map[string]bool{}
		for decoder.More() {
			keyToken, err := decoder.Token()
			if err != nil {
				return err
			}
			key, ok := keyToken.(string)
			if !ok {
				return errors.New("JSON object key is not a string")
			}
			folded := strings.ToLower(key)
			if seen[folded] {
				return fmt.Errorf("duplicate or case-aliased JSON field %q", key)
			}
			seen[folded] = true
			if err := consumeJSONValue(decoder); err != nil {
				return err
			}
		}
		closing, err := decoder.Token()
		if err != nil || closing != json.Delim('}') {
			return errors.New("malformed JSON object")
		}
	case '[':
		for decoder.More() {
			if err := consumeJSONValue(decoder); err != nil {
				return err
			}
		}
		closing, err := decoder.Token()
		if err != nil || closing != json.Delim(']') {
			return errors.New("malformed JSON array")
		}
	default:
		return errors.New("unexpected JSON delimiter")
	}
	return nil
}
