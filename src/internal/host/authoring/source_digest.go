package authoring

import "fmt"

func DigestEncodings(resources []*Resource) (map[string][]byte, error) {
	out := make(map[string][]byte, len(resources))
	for _, r := range resources {
		if r == nil {
			continue
		}
		data, err := Encode(*r)
		if err != nil {
			return nil, fmt.Errorf("encode subject %s: %w", r.GraphKey(), err)
		}
		out[r.GraphKey()] = data
	}
	return out, nil
}
