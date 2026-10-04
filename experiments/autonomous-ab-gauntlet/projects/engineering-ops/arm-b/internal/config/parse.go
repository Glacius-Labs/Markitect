package config

import (
	"encoding/json"
	"fmt"
)

type Document struct {
	Name string            `json:"name"`
	Labels map[string]string `json:"labels"`
}

func Parse(data []byte) (Document, error) {
	var document Document
	if err := json.Unmarshal(data, &document); err != nil {
		return Document{}, fmt.Errorf("parse config: %w", err)
	}
	if document.Name == "" {
		return Document{}, fmt.Errorf("parse config: name is required")
	}
	return document, nil
}
