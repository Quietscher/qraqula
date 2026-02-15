package graphql

import (
	"encoding/json"
	"time"
)

type Request struct {
	Query     string         `json:"query"`
	Variables map[string]any `json:"variables,omitempty"`
}

type Response struct {
	Data   json.RawMessage `json:"data,omitempty"`
	Errors []Error         `json:"errors,omitempty"`
}

type Error struct {
	Message string `json:"message"`
}

type Result struct {
	Response   Response
	StatusCode int
	Duration   time.Duration
	Size       int
}

func (r Response) HasErrors() bool {
	return len(r.Errors) > 0
}
