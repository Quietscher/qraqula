package graphql

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClientExecute(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("expected application/json content type")
		}

		var req Request
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("failed to decode request: %v", err)
		}
		if req.Query != "{ hello }" {
			t.Errorf("expected query '{ hello }', got %q", req.Query)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(Response{
			Data: json.RawMessage(`{"hello":"world"}`),
		})
	}))
	defer srv.Close()

	client := NewClient()
	result, err := client.Execute(context.Background(), srv.URL, Request{Query: "{ hello }"}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.StatusCode != 200 {
		t.Errorf("expected status 200, got %d", result.StatusCode)
	}
	if string(result.Response.Data) != `{"hello":"world"}` {
		t.Errorf("unexpected data: %s", result.Response.Data)
	}
	if result.Duration <= 0 {
		t.Error("expected positive duration")
	}
	if result.Size <= 0 {
		t.Error("expected positive size")
	}
}

func TestClientExecuteWithHeaders(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Errorf("expected Authorization header, got %q", r.Header.Get("Authorization"))
		}
		if r.Header.Get("X-Custom") != "value" {
			t.Errorf("expected X-Custom header, got %q", r.Header.Get("X-Custom"))
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(Response{Data: json.RawMessage(`{}`)})
	}))
	defer srv.Close()

	client := NewClient()
	headers := map[string]string{
		"Authorization": "Bearer test-token",
		"X-Custom":      "value",
	}
	_, err := client.Execute(context.Background(), srv.URL, Request{Query: "{ hello }"}, headers)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestClientExecuteCancellation(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	}))
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	client := NewClient()
	_, err := client.Execute(ctx, srv.URL, Request{Query: "{ hello }"}, nil)
	if err == nil {
		t.Fatal("expected error from cancelled context")
	}
}
